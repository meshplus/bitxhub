package mock_precheck

import (
	"go.uber.org/mock/gomock"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order/common"
	"github.com/axiomesh/axiom/internal/order/precheck"
)

func NewMockMinPreCheck(mockCtl *gomock.Controller, validTxsCh chan *precheck.ValidTxs) *MockPreCheck {
	mockPrecheck := NewMockPreCheck(mockCtl)
	mockPrecheck.EXPECT().Start().AnyTimes()
	mockPrecheck.EXPECT().PostUncheckedTxEvent(gomock.Any()).Do(func(ev *common.UncheckedTxEvent) {
		switch ev.EventType {
		case common.LocalTxEvent:
			txWithResp := ev.Event.(*common.TxWithResp)

			validTxsCh <- &precheck.ValidTxs{
				Local:       true,
				Txs:         []*types.Transaction{txWithResp.Tx},
				LocalRespCh: txWithResp.RespCh,
			}
		case common.RemoteTxEvent:
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
