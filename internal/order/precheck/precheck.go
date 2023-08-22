package precheck

import "github.com/axiomesh/axiom/internal/order"

//go:generate mockgen -destination mock_precheck/mock_precheck.go -package mock_precheck -source precheck.go
type PreCheck interface {
	// Start starts the precheck service
	Start()

	// PostUncheckedTxEvent posts unchecked tx event to precheckMgr
	PostUncheckedTxEvent(ev *order.UncheckedTxEvent)

	// CommitValidTxs returns a channel of valid transactions
	CommitValidTxs() chan *ValidTxs
}
