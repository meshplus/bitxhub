package governance

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	vm "github.com/meshplus/eth-kit/evm"
	"github.com/sirupsen/logrus"
)

var (
	ErrMethodName = errors.New("no this method")
	ErrVoteResult = errors.New("vote result is invalid")
)

const jsondata = `
[
	{"type": "function", "name": "proposal", "inputs": [{"name": "proposalType", "type": "uint8"}, {"name": "title", "type": "string"}, {"name": "desc", "type": "string"}, {"name": "blockNumber", "type": "uint64"}, {"name": "extra", "type": "bytes"}], "outputs": [{"name": "proposalId", "type": "uint64"}]},
	{"type": "function", "name": "vote", "inputs": [{"name": "proposalId", "type": "uint64"}, {"name": "voteResult", "type": "uint8"}, {"name": "extra", "type": "bytes"}]}
]
`

const (
	ProposalMethod = "proposal"
	VoteMethod     = "vote"
)

var method2Sig = map[string]string{
	ProposalMethod: "proposal(uint8,string,string,uint64,bytes)",
	VoteMethod:     "vote(uint64,uint8,bytes)",
}

type ProposalType uint8

const (
	// NodeUpdate is a proposal for update and upgrade the node
	NodeUpdate ProposalType = iota
)

type VoteResult uint8

const (
	Pass VoteResult = iota
	Reject
	Abstain
)

type ProposalArg struct {
	ProposalType uint8
	Title        string
	Desc         string
	BlockNumber  uint64
	Extra        []byte
}

type VoteArg struct {
	ProposalId uint64
	VoteResult uint8
	Extra      []byte
}

type Governance struct {
	proposalType ProposalType
	logger       logrus.FieldLogger

	gabi       *abi.ABI
	method2Sig map[string]string
}

func NewGov(proposalType ProposalType, logger logrus.FieldLogger) (*Governance, error) {
	gabi, err := GetABI()
	if err != nil {
		return nil, err
	}

	return &Governance{
		proposalType: proposalType,
		logger:       logger,
		gabi:         gabi,
		method2Sig:   method2Sig,
	}, nil
}

// GetABI get system contract abi
func GetABI() (*abi.ABI, error) {
	gabi, err := abi.JSON(strings.NewReader(jsondata))
	if err != nil {
		return nil, err
	}
	return &gabi, nil
}

// GetMethodName quickly returns the name of a method.
// This is a quick way to get the name of a method.
// The method name is the first 4 bytes of the keccak256 hash of the method signature.
// If the method name is not found, the empty string is returned.
func (g *Governance) GetMethodName(data []byte) (string, error) {
	for methodName, methodSig := range g.method2Sig {
		id := crypto.Keccak256([]byte(methodSig))[:4]
		g.logger.Infof("method id: %v, get method id: %v", id, data[:4])
		if bytes.Equal(id, data[:4]) {
			return methodName, nil
		}
	}

	return "", ErrMethodName
}

// ParseArgs parse the arguments to specified interface by method name
func (g *Governance) ParseArgs(msg *vm.Message, methodName string, ret interface{}) error {
	if len(msg.Data) < 4 {
		return fmt.Errorf("msg data length is not improperly formatted: %q - Bytes: %+v", msg.Data, msg.Data)
	}

	// discard method id
	data := msg.Data[4:]

	var args abi.Arguments
	if method, ok := g.gabi.Methods[methodName]; ok {
		if len(data)%32 != 0 {
			return fmt.Errorf("gabi: improperly formatted output: %q - Bytes: %+v", data, data)
		}
		args = method.Inputs
	}

	if args == nil {
		return fmt.Errorf("gabi: could not locate named method: %s", methodName)
	}

	unpacked, err := args.Unpack(data)
	if err != nil {
		return err
	}
	return args.Copy(ret, unpacked)
}

// GetArgs get system contract arguments from a message
func (g *Governance) GetArgs(msg *vm.Message) (interface{}, error) {
	data := msg.Data
	if data == nil {
		return nil, vm.ErrExecutionReverted
	}

	method, err := g.GetMethodName(data)
	if err != nil {
		return nil, err
	}

	switch method {
	case ProposalMethod:
		proposalArg := &ProposalArg{}
		if err := g.ParseArgs(msg, ProposalMethod, proposalArg); err != nil {
			return nil, err
		}
		return proposalArg, nil
	case VoteMethod:
		voteArg := &VoteArg{}
		if err := g.ParseArgs(msg, VoteMethod, voteArg); err != nil {
			return nil, err
		}
		return voteArg, nil
	default:
		return nil, ErrMethodName
	}
}

func (g *Governance) Proposal(user types.Address, title, desc string, deadlineBlockNumber uint64) (*BaseProposal, error) {
	proposal := &BaseProposal{
		Type:        g.proposalType,
		Strategy:    NowProposalStrategy,
		Proposer:    user.String(),
		Title:       title,
		Desc:        desc,
		BlockNumber: deadlineBlockNumber,
	}

	return proposal, nil
}

// Vote a proposal, return vote status
func (g *Governance) Vote(user types.Address, proposal *BaseProposal, voteResult VoteResult) (ProposalStatus, error) {
	switch voteResult {
	case Pass:
		proposal.PassVotes = append(proposal.PassVotes, user.String())
	case Reject:
		proposal.RejectVotes = append(proposal.RejectVotes, user.String())
	case Abstain:
		proposal.AbstainVotes = append(proposal.AbstainVotes, user.String())
	default:
		return Rejected, ErrVoteResult
	}

	return CalcProposalStatus(proposal.Strategy, proposal.TotalVotes, uint32(len(proposal.PassVotes)), uint32(len(proposal.RejectVotes)), uint32(len(proposal.AbstainVotes))), nil
}
