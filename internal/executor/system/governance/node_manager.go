package governance

import (
	"encoding/json"
	"errors"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/pkg/repo"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger"
)

const (
	NodeManagementProposalGas uint64 = 30000
	NodeManagementVoteGas     uint64 = 21600

	// NodeProposalKey is key for NodeProposal storage
	NodeProposalKey = "nodeProposalKey"

	// NodeMembersKey is key for node member storage
	NodeMembersKey = "nodeMembersKey"
)

var (
	ErrNodeNumber              = errors.New("node members total count can't bigger than candidates count")
	ErrNotFoundNodeMember      = errors.New("node member is not found")
	ErrNodeExtraArgs           = errors.New("unmarshal node extra arguments error")
	ErrNodeProposalNumberLimit = errors.New("node proposal number limit, only allow one node proposal")
	ErrNotFoundNodeProposal    = errors.New("node proposal not found for the id")
	ErrRepeatedNodeID          = errors.New("repeated node id")
)

// NodeExtraArgs is Node proposal extra arguments
type NodeExtraArgs struct {
	Nodes []*NodeMember
}

// NodeProposalArgs is node proposal arguments
type NodeProposalArgs struct {
	BaseProposalArgs
	NodeExtraArgs
}

// NodeProposal is storage of node proposal
type NodeProposal struct {
	BaseProposal
	Nodes []*NodeMember
}

type Node struct {
	Members []*NodeMember
}

type NodeMember struct {
	NodeId string
}

type NodeVoteArgs struct {
	BaseVoteArgs
}

var _ common.SystemContract = (*NodeManager)(nil)

type NodeManager struct {
	gov *Governance

	account     ledger.IAccount
	stateLedger ledger.StateLedger
	currentLog  *common.Log
	proposalID  *ProposalID
}

func NewNodeManager(logger logrus.FieldLogger) *NodeManager {
	gov, err := NewGov([]ProposalType{NodeUpdate, NodeAdd, NodeRemove}, logger)
	if err != nil {
		panic(err)
	}

	return &NodeManager{
		gov: gov,
	}
}

func (nm *NodeManager) Reset(stateLedger ledger.StateLedger) {
	addr := types.NewAddressByStr(common.NodeManagerContractAddr)
	nm.account = stateLedger.GetOrCreateAccount(addr)
	nm.stateLedger = stateLedger
	nm.currentLog = &common.Log{
		Address: addr,
	}
	nm.proposalID = NewProposalID(stateLedger)
}

func (nm *NodeManager) Run(msg *vm.Message) (*vm.ExecutionResult, error) {
	defer nm.gov.SaveLog(nm.stateLedger, nm.currentLog)

	// parse method and arguments from msg payload
	args, err := nm.gov.GetArgs(msg)
	if err != nil {
		return nil, err
	}

	var result *vm.ExecutionResult
	switch v := args.(type) {
	case *ProposalArgs:
		nodeArgs := &NodeProposalArgs{
			BaseProposalArgs: v.BaseProposalArgs,
		}

		extraArgs := &NodeExtraArgs{}
		if err = json.Unmarshal(v.Extra, extraArgs); err != nil {
			return nil, ErrNodeExtraArgs
		}

		nodeArgs.NodeExtraArgs = *extraArgs

		result, err = nm.propose(msg.From, nodeArgs)
	case *VoteArgs:
		voteArgs := &NodeVoteArgs{
			BaseVoteArgs: v.BaseVoteArgs,
		}

		result, err = nm.vote(msg.From, voteArgs)
	default:
		return nil, errors.New("unknown proposal args")
	}

	return result, err
}

func (nm *NodeManager) propose(addr ethcommon.Address, args *NodeProposalArgs) (*vm.ExecutionResult, error) {
	baseProposal, err := nm.gov.Propose(&addr, ProposalType(args.ProposalType), args.Title, args.Desc, args.BlockNumber)
	if err != nil {
		return nil, err
	}

	// check proposal has repeated nodes
	if len(lo.Uniq[string](lo.Map[*NodeMember, string](args.Nodes, func(item *NodeMember, index int) string {
		return item.NodeId
	}))) != len(args.Nodes) {
		return nil, ErrRepeatedNodeID
	}

	// check addr if is exist in council
	isExist, council := checkInCouncil(nm.account, addr.String())
	if !isExist {
		return nil, ErrNotFoundCouncilMember
	}

	// set proposal id
	proposal := &NodeProposal{
		BaseProposal: *baseProposal,
	}

	id, err := nm.proposalID.GetAndAddID()
	if err != nil {
		return nil, err
	}
	proposal.ID = id
	proposal.Nodes = args.Nodes
	proposal.TotalVotes = lo.Sum[uint64](lo.Map[*CouncilMember, uint64](council.Members, func(item *CouncilMember, index int) uint64 {
		return item.Weight
	}))

	b, err := json.Marshal(proposal)

	// save proposal
	nm.account.SetState([]byte(fmt.Sprintf("%s%d", NodeProposalKey, proposal.ID)), b)

	// record log
	nm.gov.RecordLog(nm.currentLog, ProposeMethod, &proposal.BaseProposal, b)

	return &vm.ExecutionResult{
		UsedGas:    NodeManagementProposalGas,
		ReturnData: b,
		Err:        err,
	}, nil
}

// Vote a proposal, return vote status
func (nm *NodeManager) vote(user ethcommon.Address, voteArgs *NodeVoteArgs) (*vm.ExecutionResult, error) {
	result := &vm.ExecutionResult{UsedGas: NodeManagementVoteGas}

	// check user can vote
	isExist, _ := checkInCouncil(nm.account, user.String())
	if !isExist {
		return nil, ErrNotFoundCouncilMember
	}

	// get proposal
	isExist, data := nm.account.GetState([]byte(fmt.Sprintf("%s%d", NodeProposalKey, voteArgs.ProposalId)))
	if !isExist {
		result.Err = ErrNotFoundNodeProposal
		return result, nil
	}

	proposal := &NodeProposal{}
	if err := json.Unmarshal(data, proposal); err != nil {
		return nil, err
	}

	res := VoteResult(voteArgs.VoteResult)
	proposalStatus, err := nm.gov.Vote(&user, &proposal.BaseProposal, res)
	if err != nil {
		return nil, err
	}
	proposal.Status = proposalStatus

	b, err := json.Marshal(proposal)
	if err != nil {
		return nil, err
	}
	// save proposal
	nm.account.SetState([]byte(fmt.Sprintf("%s%d", NodeProposalKey, proposal.ID)), b)

	// if proposal is approved, update the node members
	// TODO: need check block number
	if proposal.Status == Approved {
		// save council
		cb, err := json.Marshal(proposal.Nodes)
		if err != nil {
			return nil, err
		}
		nm.account.SetState([]byte(NodeMembersKey), cb)
	}

	// record log
	nm.gov.RecordLog(nm.currentLog, ProposeMethod, &proposal.BaseProposal, b)

	// return updated proposal
	result.ReturnData = b
	return result, nil
}

func (nm *NodeManager) EstimateGas(callArgs *types.CallArgs) (uint64, error) {
	args, err := nm.gov.GetArgs(&vm.Message{Data: *callArgs.Data})
	if err != nil {
		return 0, err
	}

	var gas uint64
	switch args.(type) {
	case *ProposalArgs:
		gas = NodeManagementProposalGas
	case *VoteArgs:
		gas = NodeManagementVoteGas
	default:
		return 0, errors.New("unknown proposal args")
	}

	return gas, nil
}

func InitNodeMembers(lg ledger.StateLedger, members []*repo.Member) error {
	// read member config, write to Ledger
	c, err := json.Marshal(members)
	if err != nil {
		return err
	}
	account := lg.GetOrCreateAccount(types.NewAddressByStr(common.NodeManagerContractAddr))
	account.SetState([]byte(NodeMembersKey), c)
	return nil
}

func GetNodeMembers(lg ledger.StateLedger) ([]*NodeMember, error) {
	account := lg.GetOrCreateAccount(types.NewAddressByStr(common.NodeManagerContractAddr))
	success, data := account.GetState([]byte(NodeMembersKey))
	if success {
		var members []*NodeMember
		if err := json.Unmarshal(data, &members); err != nil {
			return nil, err
		}
		return members, nil
	}
	return nil, errors.New("node member should be initialized in genesis")
}
