package governance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProposalStrategy_CalcProposalStatus(t *testing.T) {
	var approvedVotes = [][]uint32{
		{100, 51, 49, 0},
		{10, 5, 4, 1},
		{9, 5, 0, 4},
		{101, 100, 0, 0},
		{999, 500, 0, 0},
		{201, 71, 70, 60},
		{100, 34, 33, 33},
		{9, 4, 3, 2},
		{10, 4, 3, 3},
		{5, 3, 2, 0},
	}

	for _, votes := range approvedVotes {
		var proposalStatus = CalcProposalStatus(SimpleStrategy, votes[0], votes[1], votes[2], votes[3])
		assert.True(t, proposalStatus == Approved)
	}

	var rejectedVotes = [][]uint32{
		{100, 49, 51, 0},
		{10, 4, 5, 1},
		{9, 4, 1, 4},
		{101, 10, 51, 0},
		{999, 0, 500, 0},
		{201, 70, 60, 71},
		{100, 33, 34, 33},
		{9, 3, 0, 5},
		{10, 3, 4, 3},
		{5, 1, 3, 0},
	}

	for _, votes := range rejectedVotes {
		var proposalStatus = CalcProposalStatus(SimpleStrategy, votes[0], votes[1], votes[2], votes[3])
		assert.True(t, proposalStatus == Rejected)
	}

	var votingVotes = [][]uint32{
		{100, 30, 30, 30},
		{10, 4, 4, 1},
		{9, 4, 0, 4},
		{101, 10, 11, 3},
		{999, 0, 400, 0},
		{201, 70, 60, 70},
		{100, 33, 30, 33},
		{9, 4, 0, 4},
		{10, 1, 1, 3},
		{5, 1, 2, 1},
	}

	for _, votes := range votingVotes {
		var proposalStatus = CalcProposalStatus(SimpleStrategy, votes[0], votes[1], votes[2], votes[3])
		assert.True(t, proposalStatus == Voting)
	}
}
