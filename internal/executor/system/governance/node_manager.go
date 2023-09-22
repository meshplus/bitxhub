package governance

import (
	"encoding/json"
	"errors"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/executor/system/common"
	"github.com/axiomesh/axiom-ledger/internal/ledger"
	vm "github.com/axiomesh/eth-kit/evm"
)

const (
	NodeManagementProposalGas uint64 = 30000
	NodeManagementVoteGas     uint64 = 21600

	// NodeProposalKey is key for NodeProposal storagemgr
	NodeProposalKey = "nodeProposalKey"

	// NodeMembersKey is key for node member storagemgr
	NodeMembersKey = "nodeMembersKey"
)

var (
	ErrNodeNumber              = errors.New("node members total count can't bigger than candidates count")
	ErrNotFoundNodeID          = errors.New("node id is not found")
	ErrNodeExtraArgs           = errors.New("unmarshal node extra arguments error")
	ErrNodeProposalNumberLimit = errors.New("node proposal number limit, only allow one node proposal")
	ErrNotFoundNodeProposal    = errors.New("node proposal not found for the id")
	ErrUnKnownProposalArgs     = errors.New("unknown proposal args")
	ErrRepeatedNodeID          = errors.New("repeated node id")
	ErrUpgradeExtraArgs        = errors.New("unmarshal node upgrade extra arguments error")
	ErrRepeatedDownloadUrl     = errors.New("repeated download url")
)

// NodeExtraArgs is Node proposal extra arguments
type NodeExtraArgs struct {
	Nodes []*NodeMember
}

// NodeProposalArgs is node proposal arguments
// For node add and remove
type NodeProposalArgs struct {
	BaseProposalArgs
	NodeExtraArgs
}

// NodeProposal is storagemgr of node proposal
type NodeProposal struct {
	BaseProposal
	NodeExtraArgs
	UpgradeExtraArgs
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

// UpgradeProposalArgs for node upgrade
type UpgradeProposalArgs struct {
	BaseProposalArgs
	UpgradeExtraArgs
}

type UpgradeExtraArgs struct {
	DownloadUrls []string
	CheckHash    string
}

var _ common.SystemContract = (*NodeManager)(nil)

type NodeManager struct {
	gov *Governance

	account        ledger.IAccount
	councilAccount ledger.IAccount
	stateLedger    ledger.StateLedger
	currentLog     *common.Log
	proposalID     *ProposalID
}

func NewNodeManager(cfg *common.SystemContractConfig) *NodeManager {
	gov, err := NewGov([]ProposalType{NodeUpgrade, NodeAdd, NodeRemove}, cfg.Logger)
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

	councilAddr := types.NewAddressByStr(common.CouncilManagerContractAddr)
	nm.councilAccount = stateLedger.GetOrCreateAccount(councilAddr)
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
		result, err = nm.propose(msg.From, v)
	case *VoteArgs:
		result, err = nm.vote(msg.From, v)
	default:
		return nil, errors.New("unknown proposal args")
	}

	return result, err
}

func (nm *NodeManager) getNodeProposalArgs(args *ProposalArgs) (*NodeProposalArgs, error) {
	nodeArgs := &NodeProposalArgs{
		BaseProposalArgs: args.BaseProposalArgs,
	}

	extraArgs := &NodeExtraArgs{}
	if err := json.Unmarshal(args.Extra, extraArgs); err != nil {
		return nil, ErrNodeExtraArgs
	}

	nodeArgs.NodeExtraArgs = *extraArgs
	return nodeArgs, nil
}

func (nm *NodeManager) getUpgradeArgs(args *ProposalArgs) (*UpgradeProposalArgs, error) {
	upgradeArgs := &UpgradeProposalArgs{
		BaseProposalArgs: args.BaseProposalArgs,
	}

	extraArgs := &UpgradeExtraArgs{}
	if err := json.Unmarshal(args.Extra, extraArgs); err != nil {
		return nil, ErrUpgradeExtraArgs
	}

	upgradeArgs.UpgradeExtraArgs = *extraArgs
	return upgradeArgs, nil
}

func (nm *NodeManager) propose(addr ethcommon.Address, args *ProposalArgs) (*vm.ExecutionResult, error) {
	result := &vm.ExecutionResult{
		UsedGas: NodeManagementProposalGas,
	}

	if args.ProposalType == uint8(NodeUpgrade) {
		upgradeArgs, err := nm.getUpgradeArgs(args)
		if err != nil {
			return nil, err
		}

		result.ReturnData, result.Err = nm.proposeUpgrade(addr, upgradeArgs)

		return result, nil
	}

	nodeArgs, err := nm.getNodeProposalArgs(args)
	if err != nil {
		return nil, err
	}

	result.ReturnData, result.Err = nm.proposeNodeAddRemove(addr, nodeArgs)

	return result, nil
}

func (nm *NodeManager) proposeNodeAddRemove(addr ethcommon.Address, args *NodeProposalArgs) ([]byte, error) {
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
	isExist, council := checkInCouncil(nm.councilAccount, addr.String())
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

	b, err := nm.saveNodeProposal(proposal)
	if err != nil {
		return nil, err
	}

	// record log
	nm.gov.RecordLog(nm.currentLog, ProposeMethod, &proposal.BaseProposal, b)

	return b, nil
}

func (nm *NodeManager) proposeUpgrade(addr ethcommon.Address, args *UpgradeProposalArgs) ([]byte, error) {
	baseProposal, err := nm.gov.Propose(&addr, ProposalType(args.ProposalType), args.Title, args.Desc, args.BlockNumber)
	if err != nil {
		return nil, err
	}

	// check proposal has repeated download url
	if len(lo.Uniq[string](args.DownloadUrls)) != len(args.DownloadUrls) {
		return nil, ErrRepeatedDownloadUrl
	}

	// check addr if is exist in council
	isExist, council := checkInCouncil(nm.councilAccount, addr.String())
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
	proposal.DownloadUrls = args.DownloadUrls
	proposal.CheckHash = args.CheckHash
	proposal.TotalVotes = lo.Sum[uint64](lo.Map[*CouncilMember, uint64](council.Members, func(item *CouncilMember, index int) uint64 {
		return item.Weight
	}))

	b, err := nm.saveNodeProposal(proposal)
	if err != nil {
		return nil, err
	}

	// record log
	nm.gov.RecordLog(nm.currentLog, ProposeMethod, &proposal.BaseProposal, b)

	return b, nil
}

// Vote a proposal, return vote status
func (nm *NodeManager) vote(user ethcommon.Address, voteArgs *VoteArgs) (*vm.ExecutionResult, error) {
	result := &vm.ExecutionResult{UsedGas: NodeManagementVoteGas}

	// get proposal
	proposal, err := nm.loadNodeProposal(voteArgs.ProposalId)
	if err != nil {
		return nil, err
	}

	if proposal.Type == NodeUpgrade {
		result.ReturnData, result.Err = nm.voteUpgrade(user, proposal, &NodeVoteArgs{BaseVoteArgs: voteArgs.BaseVoteArgs})
		return result, nil
	}

	result.ReturnData, result.Err = nm.voteNodeAddRemove(user, proposal, &NodeVoteArgs{BaseVoteArgs: voteArgs.BaseVoteArgs})
	return result, nil
}

func (nm *NodeManager) voteNodeAddRemove(user ethcommon.Address, proposal *NodeProposal, voteArgs *NodeVoteArgs) ([]byte, error) {
	// check user can vote
	isExist, _ := checkInCouncil(nm.councilAccount, user.String())
	if !isExist {
		return nil, ErrNotFoundCouncilMember
	}

	res := VoteResult(voteArgs.VoteResult)
	proposalStatus, err := nm.gov.Vote(&user, &proposal.BaseProposal, res)
	if err != nil {
		return nil, err
	}
	proposal.Status = proposalStatus

	b, err := nm.saveNodeProposal(proposal)
	if err != nil {
		return nil, err
	}

	// if proposal is approved, update the node members
	if proposal.Status == Approved {
		members, err := GetNodeMembers(nm.stateLedger)
		if err != nil {
			return nil, err
		}

		if proposal.Type == NodeAdd {
			members = append(members, proposal.Nodes...)
		}

		if proposal.Type == NodeRemove {
			// https://github.com/samber/lo
			// Use the Associate method to create a map with the node's NodeId as the key and the NodeMember object as the value
			nodeIdToNodeMap := lo.Associate(proposal.Nodes, func(node *NodeMember) (string, *NodeMember) {
				return node.NodeId, node
			})

			// The members slice is updated to filteredMembers, which does not contain members with the same NodeId as proposalNodes
			filteredMembers := lo.Reject(members, func(member *NodeMember, _ int) bool {
				_, exists := nodeIdToNodeMap[member.NodeId]
				return exists
			})

			members = filteredMembers
		}

		cb, err := json.Marshal(members)
		if err != nil {
			return nil, err
		}
		nm.account.SetState([]byte(NodeMembersKey), cb)
	}

	// record log
	nm.gov.RecordLog(nm.currentLog, VoteMethod, &proposal.BaseProposal, b)

	return b, nil
}

func (nm *NodeManager) voteUpgrade(user ethcommon.Address, proposal *NodeProposal, voteArgs *NodeVoteArgs) ([]byte, error) {
	// check user can vote
	isExist, _ := checkInCouncil(nm.councilAccount, user.String())
	if !isExist {
		return nil, ErrNotFoundCouncilMember
	}

	res := VoteResult(voteArgs.VoteResult)
	proposalStatus, err := nm.gov.Vote(&user, &proposal.BaseProposal, res)
	if err != nil {
		return nil, err
	}
	proposal.Status = proposalStatus

	b, err := nm.saveNodeProposal(proposal)
	if err != nil {
		return nil, err
	}

	// record log
	// if approved, guardian sync log, then update node and restart
	nm.gov.RecordLog(nm.currentLog, VoteMethod, &proposal.BaseProposal, b)

	return b, nil
}

func (nm *NodeManager) saveNodeProposal(proposal *NodeProposal) ([]byte, error) {
	b, err := json.Marshal(proposal)
	if err != nil {
		return nil, err
	}
	// save proposal
	nm.account.SetState([]byte(fmt.Sprintf("%s%d", NodeProposalKey, proposal.ID)), b)

	return b, nil
}

func (nm *NodeManager) loadNodeProposal(proposalID uint64) (*NodeProposal, error) {
	isExist, data := nm.account.GetState([]byte(fmt.Sprintf("%s%d", NodeProposalKey, proposalID)))
	if !isExist {
		return nil, ErrNotFoundNodeProposal
	}

	proposal := &NodeProposal{}
	if err := json.Unmarshal(data, proposal); err != nil {
		return nil, err
	}

	return proposal, nil
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

func (nm *NodeManager) CheckAndUpdateState(lastHeight uint64, stateLedger ledger.StateLedger) {
	nm.Reset(stateLedger)

	if isExist, data := nm.account.Query(NodeProposalKey); isExist {
		for _, proposalData := range data {
			proposal := &NodeProposal{}
			if err := json.Unmarshal(proposalData, proposal); err != nil {
				nm.gov.logger.Errorf("unmarshal council proposal error: %s", err)
				return
			}

			if proposal.Status == Approved || proposal.Status == Rejected {
				// proposal is finnished, no need update
				continue
			}

			if proposal.BlockNumber != 0 && proposal.BlockNumber <= lastHeight {
				// means proposal is out of deadline,status change to rejected
				proposal.Status = Rejected

				// remove node is special, proposal should be auto approved when out of deadline
				if proposal.Type == NodeRemove {
					proposal.Status = Approved
				}

				b, err := nm.saveNodeProposal(proposal)
				if err != nil {
					nm.gov.logger.Errorf("unmarshal node proposal error: %s", err)
				}

				nm.gov.RecordLog(nm.currentLog, VoteMethod, &proposal.BaseProposal, b)
				nm.gov.SaveLog(stateLedger, nm.currentLog)
			}
		}
	}
}

func InitNodeMembers(lg ledger.StateLedger, members []*NodeMember) error {
	// read member config, write to ViewLedger
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
