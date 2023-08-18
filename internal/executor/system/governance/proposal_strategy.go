package governance

type ProposalStrategy uint8

const (
	// SimpleStrategy means proposal is approved if pass votes is greater than half of total votes
	SimpleStrategy ProposalStrategy = iota
)

var NowProposalStrategy = SimpleStrategy

func CalcProposalStatus(strategy ProposalStrategy, totalVotes, passVotes, rejectVotes uint64) ProposalStatus {
	switch strategy {
	case SimpleStrategy:
		return calcProposalStatusBySimpleStrategy(totalVotes, passVotes, rejectVotes)
	default:
		return Voting
	}
}

func calcProposalStatusBySimpleStrategy(totalVotes, passVotes, rejectVotes uint64) ProposalStatus {
	// pass votes is more than half of total votes, approved
	if passVotes*2 > totalVotes {
		return Approved
	}

	// reject votes is more than half of total votes, rejected
	if rejectVotes*2 > totalVotes {
		return Rejected
	}

	// not ending, return voting
	return Voting
}
