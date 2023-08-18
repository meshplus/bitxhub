package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProposalStrategy_CalcProposalStatus(t *testing.T) {
	var approvedVotes = [][]uint64{
		{100, 51, 49},
		{10, 6, 4},
		{9, 5, 4},
		{101, 100, 0},
		{999, 500, 0},
		{201, 121, 70},
		{100, 64, 30},
		{9, 5, 3},
		{10, 7, 1},
		{5, 3, 2},
	}

	for _, votes := range approvedVotes {
		var proposalStatus = CalcProposalStatus(SimpleStrategy, votes[0], votes[1], votes[2])
		assert.True(t, proposalStatus == Approved)
	}

	var rejectedVotes = [][]uint64{
		{100, 49, 51},
		{10, 4, 6},
		{9, 4, 5},
		{101, 10, 51},
		{999, 0, 500},
		{201, 70, 122},
		{100, 33, 55},
		{9, 3, 5},
		{10, 3, 6},
		{5, 1, 3},
	}

	for _, votes := range rejectedVotes {
		var proposalStatus = CalcProposalStatus(SimpleStrategy, votes[0], votes[1], votes[2])
		assert.True(t, proposalStatus == Rejected)
	}

	var votingVotes = [][]uint64{
		{100, 30, 30},
		{10, 4, 4},
		{9, 4, 0},
		{101, 10, 11},
		{999, 0, 400},
		{201, 70, 60},
		{100, 33, 30},
		{9, 4, 0},
		{10, 1, 1},
		{5, 1, 2},
	}

	for _, votes := range votingVotes {
		var proposalStatus = CalcProposalStatus(SimpleStrategy, votes[0], votes[1], votes[2])
		assert.True(t, proposalStatus == Voting)
	}
}
