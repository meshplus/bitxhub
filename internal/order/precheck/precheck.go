package precheck

import (
	"github.com/axiomesh/axiom/internal/order/common"
)

//go:generate mockgen -destination mock_precheck/mock_precheck.go -package mock_precheck -source precheck.go -typed
type PreCheck interface {
	// Start starts the precheck service
	Start()

	// PostUncheckedTxEvent posts unchecked tx event to precheckMgr
	PostUncheckedTxEvent(ev *common.UncheckedTxEvent)

	// CommitValidTxs returns a channel of valid transactions
	CommitValidTxs() chan *ValidTxs
}
