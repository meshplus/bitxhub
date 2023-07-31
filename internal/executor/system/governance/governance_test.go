package governance

import (
	"strings"
	"testing"

	"github.com/axiomesh/axiom-kit/types"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGovernance_GetABI(t *testing.T) {
	gabi, err := GetABI()
	assert.Nil(t, err)
	assert.NotNil(t, gabi)

	data, err := gabi.Pack(ProposalMethod, uint8(NodeUpdate), "title", "desc", uint64(1000), []byte(""))
	assert.Nil(t, err)
	assert.NotNil(t, data)
}

func TestGovernance_GetMethodName(t *testing.T) {
	logger := logrus.New()
	gov, err := NewGov(NodeUpdate, logger)
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	data, err := gov.gabi.Pack(ProposalMethod, uint8(NodeUpdate), "title", "desc", uint64(1000), []byte(""))
	assert.Nil(t, err)

	methodName, err := gov.GetMethodName(data)
	assert.Nil(t, err)

	assert.Equal(t, ProposalMethod, methodName)
}

func TestGovernance_GetErrMethodName(t *testing.T) {
	logger := logrus.New()
	gov, err := NewGov(NodeUpdate, logger)
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	errMethod := "no_this_method"
	data, err := gov.gabi.Pack(errMethod, uint8(NodeUpdate), "title", "desc", uint64(1000), []byte(""))
	assert.NotNil(t, err)

	test_jsondata := `[
		{"type": "function", "name": "___", "inputs": [{"name": "proposalType", "type": "uint8"}, {"name": "title", "type": "string"}, {"name": "desc", "type": "string"}, {"name": "blockNumber", "type": "uint64"}, {"name": "extra", "type": "bytes"}], "outputs": [{"name": "proposalId", "type": "uint64"}]}
	]`
	newAbi, err := abi.JSON(strings.NewReader(strings.ReplaceAll(test_jsondata, "___", errMethod)))
	assert.Nil(t, err)
	data, err = newAbi.Pack(errMethod, NodeUpdate, "title", "desc", uint64(1000), []byte(""))
	assert.Nil(t, err)

	_, err = gov.GetMethodName(data)
	assert.Equal(t, ErrMethodName, err)
}

func TestGovernance_ParseErrorArgs(t *testing.T) {
	logger := logrus.New()
	gov, err := NewGov(NodeUpdate, logger)
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	truedata, err := gov.gabi.Pack(ProposalMethod, uint8(NodeUpdate), "title", "desc", uint64(1000), []byte(""))
	assert.Nil(t, err)
	testcases := []struct {
		method string
		data   []byte
	}{
		{
			method: "proposal",
			data:   []byte{1},
		},
		{
			method: "proposal",
			data:   []byte{1, 2, 3, 4},
		},
		{
			method: "proposal",
			data:   []byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			method: "proposal",
			data:   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36},
		},
		{
			method: "test method",
			data:   truedata,
		},
		{
			method: "proposal",
			data:   truedata,
		},
	}

	for _, test := range testcases {
		err = gov.ParseArgs(&vm.Message{
			Data: test.data,
		}, test.method, nil)
		assert.NotNil(t, err)
	}
}

func TestGovernance_GetArgsForProposal(t *testing.T) {
	logger := logrus.New()
	gov, err := NewGov(NodeUpdate, logger)
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	title := "title"
	desc := "desc"
	blockNumber := uint64(1000)
	extra := []byte("hello")
	data, err := gov.gabi.Pack(ProposalMethod, uint8(NodeUpdate), title, desc, blockNumber, extra)
	assert.Nil(t, err)

	arg, err := gov.GetArgs(&vm.Message{
		Data: data,
	})
	assert.Nil(t, err)

	proposalArg, ok := arg.(*ProposalArg)
	assert.True(t, ok)

	assert.Equal(t, NodeUpdate, ProposalType(proposalArg.ProposalType))
	assert.Equal(t, title, proposalArg.Title)
	assert.Equal(t, desc, proposalArg.Desc)
	assert.Equal(t, blockNumber, proposalArg.BlockNumber)
	assert.Equal(t, extra, proposalArg.Extra)
}

func TestGovernance_GetArgsForVote(t *testing.T) {
	logger := logrus.New()
	gov, err := NewGov(NodeUpdate, logger)
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	proposalId := uint64(100)
	voteResult := Pass
	extra := []byte("hello")
	data, err := gov.gabi.Pack(VoteMethod, proposalId, uint8(voteResult), extra)
	assert.Nil(t, err)

	arg, err := gov.GetArgs(&vm.Message{
		Data: data,
	})
	assert.Nil(t, err)

	voteArg, ok := arg.(*VoteArg)
	assert.True(t, ok)

	assert.Equal(t, proposalId, voteArg.ProposalId)
	assert.Equal(t, voteResult, VoteResult(voteArg.VoteResult))
	assert.Equal(t, extra, voteArg.Extra)
}

func TestGovernance_GetErrArgs(t *testing.T) {
	gov, err := NewGov(NodeUpdate, logrus.New())
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	testjsondata := `
	[
		{"type": "function", "name": "this_method", "inputs": [{"name": "proposalType", "type": "uint8"}, {"name": "title", "type": "string"}, {"name": "desc", "type": "string"}, {"name": "blockNumber", "type": "uint64"}, {"name": "extra", "type": "bytes"}], "outputs": [{"name": "proposalId", "type": "uint64"}]}
	]
	`
	gabi, err := abi.JSON(strings.NewReader(testjsondata))
	gov.gabi = &gabi

	errMethod := "no_this_method"
	errMethodData, err := gov.gabi.Pack(errMethod, uint8(NodeUpdate), "title", "desc", uint64(1000), []byte(""))
	assert.NotNil(t, err)

	thisMethod := "this_method"
	thisMethodData, err := gov.gabi.Pack(thisMethod, uint8(NodeUpdate), "title", "desc", uint64(1000), []byte(""))
	assert.Nil(t, err)

	testcases := [][]byte{
		nil,
		{1, 2, 3, 4},
		errMethodData,
		thisMethodData,
	}

	for _, test := range testcases {
		_, err = gov.GetArgs(&vm.Message{Data: test})
		assert.NotNil(t, err)
	}
}

func TestGovernance_Proposal(t *testing.T) {
	logger := logrus.New()
	gov, err := NewGov(NodeUpdate, logger)
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	addr := types.NewAddressByStr("0x10000000000000000000000000000000000000000")
	title := "test title"
	desc := "test desc"
	blockNumber := uint64(10000)
	proposal, err := gov.Proposal(*addr, title, desc, blockNumber)
	assert.Nil(t, err)
	assert.NotNil(t, proposal)
	assert.Equal(t, proposal.Type, NodeUpdate)
	assert.Equal(t, proposal.Title, title)
	assert.Equal(t, proposal.Desc, desc)
	assert.Equal(t, proposal.BlockNumber, blockNumber)
}

func TestGovernance_Vote(t *testing.T) {
	logger := logrus.New()
	gov, err := NewGov(NodeUpdate, logger)
	assert.Nil(t, err)
	assert.NotNil(t, gov)

	addr := types.NewAddressByStr("0x1000000000000000000000000000000000000000")
	anotherAddr := types.NewAddressByStr("0x2000000000000000000000000000000000000000")
	proposal, err := gov.Proposal(*addr, "test title", "test desc", uint64(10000))
	assert.Nil(t, err)
	assert.NotNil(t, proposal)

	proposal.ID = 10000
	proposal.TotalVotes = 3

	testcases := []struct {
		result   VoteResult
		expected ProposalStatus
	}{
		{
			result:   Reject,
			expected: Rejected,
		},
		{
			result:   Abstain,
			expected: Rejected,
		},
		{
			result:   Pass,
			expected: Approved,
		},
	}

	for _, test := range testcases {
		status, err := gov.Vote(*addr, proposal, test.result)
		assert.Nil(t, err)
		assert.Equal(t, Voting, status)

		status, err = gov.Vote(*anotherAddr, proposal, test.result)
		assert.Nil(t, err)
		assert.Equal(t, test.expected, status)

		proposal.PassVotes = nil
		proposal.RejectVotes = nil
		proposal.AbstainVotes = nil
	}
}
