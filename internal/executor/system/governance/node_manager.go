package governance

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

const (
	NodeManagementProposalGas uint64 = 30000
	NodeManagementVoteGas     uint64 = 21600
)

var (
	ErrNodeNumber              = errors.New("node members total count can't bigger than candidates count")
	ErrNotFoundNodeMember      = errors.New("node member is not found")
	ErrNodeExtraArgs           = errors.New("unmarshal node extra arguments error")
	ErrNodeProposalNumberLimit = errors.New("node proposal number limit, only allow one node proposal")
	ErrNotFoundNodeProposal    = errors.New("node proposal not found for the id")
)

const (
	// NodeProposalKey is key for NodeProposal storage
	NodeProposalKey = "councilProposalKey"

	// TODO: set used gas
	// NodeProposalGas is used gas for node proposal
	NodeProposalGas = 1000

	// NodeVoteGas is used gas for node vote
	NodeVoteGas = 100
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

	account ledger.IAccount
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
	nm.account = stateLedger.GetOrCreateAccount(types.NewAddressByStr(common.NodeManagerContractAddr))
	globalProposalID = GetInstanceOfProposalID(stateLedger)
}

func (nm *NodeManager) Run(msg *vm.Message) (*vm.ExecutionResult, error) {
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

	id, err := globalProposalID.GetAndAddID()
	if err != nil {
		return nil, err
	}

	// set proposal id
	proposal := &NodeProposal{
		BaseProposal: *baseProposal,
	}
	proposal.ID = id
	// check nodeMember if is exist
	isExist, data := nm.account.GetState([]byte(common.NodeMemberContractAddr))
	if !isExist {
		return nil, errors.New("council should be initialized in genesis")
	}
	node := &Node{}
	if err = json.Unmarshal(data, node); err != nil {
		return nil, err
	}

	council := &Council{}
	if err = json.Unmarshal(data, council); err != nil {
		return nil, err
	}

	// TODO check addr if is exist in council

	proposal.TotalVotes = lo.Sum[uint64](lo.Map[*CouncilMember, uint64](council.Members, func(item *CouncilMember, index int) uint64 {
		return item.Weight
	}))
	proposal.Nodes = args.Nodes

	b, err := json.Marshal(proposal)

	// save proposal
	nm.account.SetState([]byte(fmt.Sprintf("%s%d", NodeProposalKey, proposal.ID)), b)

	return &vm.ExecutionResult{
		UsedGas:    NodeProposalGas,
		ReturnData: b,
		Err:        err,
	}, nil
}

// Vote a proposal, return vote status
func (nm *NodeManager) vote(user ethcommon.Address, voteArgs *NodeVoteArgs) (*vm.ExecutionResult, error) {
	result := &vm.ExecutionResult{UsedGas: NodeVoteGas}

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
		result.Err = err
		return result, nil
	}
	proposal.Status = proposalStatus

	// TODO: check user can vote
	// check user if is already voted

	b, err := json.Marshal(proposal)
	if err != nil {
		return nil, err
	}
	// save proposal
	nm.account.SetState([]byte(fmt.Sprintf("%s%d", NodeProposalKey, proposal.ID)), b)

	// if proposal is approved, update the node members
	// TODO: need check block number
	if proposal.Status == Approved {
		node := &Node{
			Members: proposal.Nodes,
		}

		// save council
		cb, err := json.Marshal(node)
		if err != nil {
			return nil, err
		}
		nm.account.SetState([]byte(common.NodeMemberContractAddr), cb)
	}

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
