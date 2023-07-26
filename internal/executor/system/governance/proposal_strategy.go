package governance

type ProposalStrategy uint8

const (
	// SimpleStrategy means proposal is approved if pass votes is greater than half of total votes
	SimpleStrategy ProposalStrategy = iota
)

var NowProposalStrategy = SimpleStrategy

func CalcProposalStatus(strategy ProposalStrategy, totalVotes, passVotes, rejectVotes, abstainVotes uint32) ProposalStatus {
	switch strategy {
	case SimpleStrategy:
		return calcProposalStatusBySimpleStrategy(totalVotes, passVotes, rejectVotes, abstainVotes)
	default:
		return Voting
	}
}

func calcProposalStatusBySimpleStrategy(totalVotes, passVotes, rejectVotes, abstainVotes uint32) ProposalStatus {
	// pass votes is more than half of total votes, approved
	if passVotes*2 > totalVotes {
		return Approved
	}

	// reject votes is more than half of total votes, rejected
	if rejectVotes*2 > totalVotes {
		return Rejected
	}

	// abstain votes is more than half of total votes, rejected
	if abstainVotes*2 > totalVotes {
		return Rejected
	}

	// all votes are submited
	if passVotes+rejectVotes+abstainVotes == totalVotes {
		// pass votes > reject votes and pass votes > abstian votes, then approved
		if passVotes > rejectVotes && passVotes > abstainVotes {
			return Approved
		}

		// otherwise, rejected
		return Rejected
	}

	// not ending, return voting
	return Voting
}
