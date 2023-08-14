package governance

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/eth-kit/ledger"
)

type ProposalStatus uint8

const (
	Voting ProposalStatus = iota
	Approved
	Rejected
)

const (
	ProposalIDKey = "proposalIDKey"
)

var (
	ErrNilProposalAccount = errors.New("ProposalID must be reset then use")
)

var globalProposalID *ProposalID

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
	TotalVotes uint64

	// passVotes record user address for passed vote
	PassVotes []string

	RejectVotes []string
	Status      ProposalStatus
}

type ProposalID struct {
	ID    uint64
	mutex sync.RWMutex

	account ledger.IAccount
}

// GetInstanceOfProposalID get instance of the global proposal id
func GetInstanceOfProposalID(stateLedger ledger.StateLedger) *ProposalID {
	// id is not initialized
	if globalProposalID == nil {
		globalProposalID = &ProposalID{}

		account := stateLedger.GetOrCreateAccount(types.NewAddressByStr(common.ProposalIDContractAddr))
		isExist, data := account.GetState([]byte(ProposalIDKey))
		if !isExist {
			globalProposalID.ID = 1
		} else {
			globalProposalID.ID = binary.BigEndian.Uint64(data)
		}

		globalProposalID.account = account
	}

	return globalProposalID
}

func (pid *ProposalID) GetID() uint64 {
	pid.mutex.RLock()
	defer pid.mutex.RUnlock()

	return pid.ID
}

func (pid *ProposalID) GetAndAddID() (uint64, error) {
	pid.mutex.Lock()
	defer pid.mutex.Unlock()

	oldID := pid.ID
	pid.ID++

	if pid.account == nil {
		return 0, ErrNilProposalAccount
	}

	// persist id
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, pid.ID)
	pid.account.SetState([]byte(ProposalIDKey), data)

	return oldID, nil
}
