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
	"github.com/sirupsen/logrus"
)

var (
	ErrCouncilNumber              = errors.New("council members total count can't bigger than candidates count")
	ErrNotFoundCouncilMember      = errors.New("council member is not found")
	ErrCouncilExtraArgs           = errors.New("unmarshal council extra arguments error")
	ErrCouncilProposalNumberLimit = errors.New("council proposal number limit, only allow one council proposal")
	ErrNotFoundCouncilProposal    = errors.New("council proposal not found for the id")
)

const (
	// CouncilProposalKey is key for CouncilProposal storage
	CouncilProposalKey = "councilProposalKey"

	// CouncilKey is key for council storage
	CouncilKey = "councilKey"

	// TODO: set used gas
	// CouncilProposalGas is used gas for council proposal
	CouncilProposalGas = 1000

	// CouncilVoteGas is used gas for council vote
	CouncilVoteGas = 100
)

// CouncilExtraArgs is council proposal extra arguments
type CouncilExtraArgs struct {
	Candidates []*CouncilMember
}

// CouncilProposalArgs is council proposal arguments
type CouncilProposalArgs struct {
	BaseProposalArgs
	CouncilExtraArgs
}

// CouncilProposal is storage of council proposal
type CouncilProposal struct {
	BaseProposal
	Candidates []*CouncilMember
}

// Council is storage of council
type Council struct {
	Members []*CouncilMember
}

func (c *Council) getMemberAddresses() []string {
	addresses := make([]string, len(c.Members))
	for i, member := range c.Members {
		addresses[i] = member.Address
	}
	return addresses
}

func (c *Council) getWeightSum() uint64 {
	var sum uint64 = 0
	for _, member := range c.Members {
		sum += member.Weight
	}
	return sum
}

type CouncilMember struct {
	Address string
	Weight  uint64
}

// CouncilVoteArgs is council vote arguments
type CouncilVoteArgs struct {
	BaseVoteArgs
}

var _ common.SystemContract = (*CouncilManager)(nil)

type CouncilManager struct {
	gov *Governance

	account ledger.IAccount
}

func NewCouncilManager(logger logrus.FieldLogger) *CouncilManager {
	gov, err := NewGov([]ProposalType{CouncilElect}, logger)
	if err != nil {
		panic(err)
	}

	return &CouncilManager{
		gov: gov,
	}
}

func (cm *CouncilManager) Reset(stateLedger ledger.StateLedger) {
	cm.account = stateLedger.GetOrCreateAccount(types.NewAddressByStr(common.CouncilManagerContractAddr))
	globalProposalID = GetInstanceOfProposalID(stateLedger)
}

func (cm *CouncilManager) Run(msg *vm.Message) (*vm.ExecutionResult, error) {
	// parse method and arguments from msg payload
	args, err := cm.gov.GetArgs(msg)
	if err != nil {
		return nil, err
	}

	var result *vm.ExecutionResult
	switch v := args.(type) {
	case *ProposalArgs:
		councilArgs := &CouncilProposalArgs{
			BaseProposalArgs: v.BaseProposalArgs,
		}

		extraArgs := &CouncilExtraArgs{}
		if err = json.Unmarshal(v.Extra, extraArgs); err != nil {
			return nil, ErrCouncilExtraArgs
		}

		councilArgs.CouncilExtraArgs = *extraArgs

		result, err = cm.propose(msg.From, councilArgs)
	case *VoteArgs:
		voteArgs := &CouncilVoteArgs{
			BaseVoteArgs: v.BaseVoteArgs,
		}

		result, err = cm.vote(msg.From, voteArgs)
	default:
		return nil, errors.New("unknown proposal args")
	}

	return result, err
}

func (cm *CouncilManager) propose(addr ethcommon.Address, args *CouncilProposalArgs) (*vm.ExecutionResult, error) {
	baseProposal, err := cm.gov.Propose(&addr, ProposalType(args.ProposalType), args.Title, args.Desc, args.BlockNumber)
	if err != nil {
		return nil, err
	}

	id, err := globalProposalID.GetAndAddID()
	if err != nil {
		return nil, err
	}

	// set proposal id
	proposal := &CouncilProposal{
		BaseProposal: *baseProposal,
	}
	proposal.ID = id
	// check council if is exist
	isExist, data := cm.account.GetState([]byte(CouncilKey))
	if !isExist {
		return nil, errors.New("council should be initialized in genesis")
	}
	council := &Council{}
	if err = json.Unmarshal(data, council); err != nil {
		return nil, err
	}
	// check addr if is exist in council
	isExist = common.IsInSlice[string](addr.String(), council.getMemberAddresses())
	if !isExist {
		return nil, ErrNotFoundCouncilMember
	}

	proposal.TotalVotes = council.getWeightSum()
	proposal.Candidates = args.Candidates

	b, err := json.Marshal(proposal)

	// save proposal
	cm.account.SetState([]byte(fmt.Sprintf("%s%d", CouncilProposalKey, proposal.ID)), b)

	return &vm.ExecutionResult{
		UsedGas:    CouncilProposalGas,
		ReturnData: b,
		Err:        err,
	}, nil
}

// Vote a proposal, return vote status
func (cm *CouncilManager) vote(user ethcommon.Address, voteArgs *CouncilVoteArgs) (*vm.ExecutionResult, error) {
	result := &vm.ExecutionResult{UsedGas: CouncilVoteGas}

	// get proposal
	isExist, data := cm.account.GetState([]byte(fmt.Sprintf("%s%d", CouncilProposalKey, voteArgs.ProposalId)))
	if !isExist {
		result.Err = ErrNotFoundCouncilProposal
		return result, nil
	}

	proposal := &CouncilProposal{}
	if err := json.Unmarshal(data, proposal); err != nil {
		return nil, err
	}

	res := VoteResult(voteArgs.VoteResult)
	proposalStatus, err := cm.gov.Vote(&user, &proposal.BaseProposal, res)
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
	cm.account.SetState([]byte(fmt.Sprintf("%s%d", CouncilProposalKey, proposal.ID)), b)

	// if proposal is approved, update the council members
	// TODO: need check block number
	if proposal.Status == Approved {
		council := &Council{
			Members: proposal.Candidates,
		}

		// save council
		cb, err := json.Marshal(council)
		if err != nil {
			return nil, err
		}
		cm.account.SetState([]byte(CouncilKey), cb)
	}

	// return updated proposal
	result.ReturnData = b
	return result, nil
}
