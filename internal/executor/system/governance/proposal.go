package governance

type ProposalStatus uint8

const (
	Voting ProposalStatus = iota
	Approved
	Rejected
)

type BaseProposal struct {
	ID          uint64
	Type        ProposalType
	Strategy    ProposalStrategy
	Proposer    string
	Title       string
	Desc        string
	BlockNumber uint64
	// totalVotes is total votes for this proposal
	// attention: some users may not vote for this proposal
	TotalVotes uint32
	// passVotes record user address for passed vote
	PassVotes    []string
	RejectVotes  []string
	AbstainVotes []string
	Status       ProposalStatus
}
