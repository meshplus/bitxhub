package mock_precheck

import (
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/order/precheck"
	"github.com/golang/mock/gomock"
)

func NewMockMinPreCheck(mockCtl *gomock.Controller, validTxsCh chan *precheck.ValidTxs) *MockPreCheck {
	mockPrecheck := NewMockPreCheck(mockCtl)
	mockPrecheck.EXPECT().Start().AnyTimes()
	mockPrecheck.EXPECT().PostUncheckedTxEvent(gomock.Any()).Do(func(ev *order.UncheckedTxEvent) {
		switch ev.EventType {
		case order.LocalTxEvent:
			txWithResp := ev.Event.(*order.TxWithResp)

			validTxsCh <- &precheck.ValidTxs{
				Local:       true,
				Txs:         []*types.Transaction{txWithResp.Tx},
				LocalRespCh: txWithResp.RespCh,
			}
		case order.RemoteTxEvent:
			txs := ev.Event.([]*types.Transaction)
			validTxsCh <- &precheck.ValidTxs{
				Local: false,
				Txs:   txs,
			}
		}
	}).AnyTimes()
	mockPrecheck.EXPECT().CommitValidTxs().Return(validTxsCh).AnyTimes()
	return mockPrecheck
}
