package governance

import (
	"encoding/binary"
	"errors"
	"strings"
	"sync"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/executor/system/common"
	"github.com/axiomesh/axiom-ledger/internal/ledger"
)

type ProposalStatus uint8

const (
	Voting ProposalStatus = iota
	Approved
	Rejected
)

const (
	ProposalIDKey = "proposalIDKey"

	Addr2NameSystemAddrKey = "addrKey"
	Addr2NameSystemNameKey = "nameKey"
)

var (
	ErrNilProposalAccount = errors.New("ProposalID must be reset then use")
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

// NewProposalID new proposal id from ledger
func NewProposalID(stateLedger ledger.StateLedger) *ProposalID {
	proposalID := &ProposalID{}
	// id is not initialized
	account := stateLedger.GetOrCreateAccount(types.NewAddressByStr(common.ProposalIDContractAddr))
	isExist, data := account.GetState([]byte(ProposalIDKey))
	if !isExist {
		proposalID.ID = 1
	} else {
		proposalID.ID = binary.BigEndian.Uint64(data)
	}

	proposalID.account = account

	return proposalID
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

type Addr2NameSystem struct {
	account ledger.IAccount
}

func NewAddr2NameSystem(stateLedger ledger.StateLedger) *Addr2NameSystem {
	addr2NameSystem := &Addr2NameSystem{}

	addr2NameSystem.account = stateLedger.GetOrCreateAccount(types.NewAddressByStr(common.Addr2NameContractAddr))

	return addr2NameSystem
}

// SetName set address to new name
func (ans *Addr2NameSystem) SetName(addr, name string) {
	ak := addrKey(addr)
	nk := nameKey(name)

	ans.account.SetState(ak, []byte(name))
	ans.account.SetState(nk, []byte(addr))
}

func (ans *Addr2NameSystem) GetName(addr string) (bool, string) {
	isExist, name := ans.account.GetState(addrKey(addr))
	return isExist, string(name)
}

func (ans *Addr2NameSystem) GetAddr(name string) (bool, string) {
	isExist, addr := ans.account.GetState(nameKey(name))
	return isExist, string(addr)
}

func addrKey(addr string) []byte {
	return []byte(strings.Join([]string{Addr2NameSystemAddrKey, addr}, "-"))
}

func nameKey(name string) []byte {
	return []byte(strings.Join([]string{Addr2NameSystemNameKey, name}, "-"))
}
