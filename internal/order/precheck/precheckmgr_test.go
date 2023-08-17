package precheck

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/stretchr/testify/require"

	"github.com/axiomesh/axiom-kit/types"
	common2 "github.com/axiomesh/axiom/internal/order/common"
)

func TestTxPreCheckMgr_Start(t *testing.T) {
	tp, logger, cancel := newMockPreCheckMgr()
	defer cleanDb()
	tp.Start()

	s, err := types.GenerateSigner()
	require.Nil(t, err)
	toAddr := common.HexToAddress(to)

	t.Run("test wrong tx event type", func(t *testing.T) {
		// catch log output
		originalOutput := logger.Logger.Out
		var logOutput bytes.Buffer
		logger.Logger.SetOutput(&logOutput)

		tx, _, err := types.GenerateTransactionAndSigner(0, types.NewAddressByStr(to), big.NewInt(0), nil)
		require.Nil(t, err)

		wrongType := 100
		event := &common2.UncheckedTxEvent{
			EventType: wrongType,
			Event: &common2.TxWithResp{
				Tx:     tx,
				RespCh: make(chan *common2.TxResp),
			},
		}
		tp.PostUncheckedTxEvent(event)
		time.Sleep(200 * time.Millisecond)

		// restore log output
		logger.Logger.SetOutput(originalOutput)
		require.True(t, bytes.Contains(logOutput.Bytes(), []byte(ErrTxEventType)))
	})

	t.Run("test parse wrong local tx event type", func(t *testing.T) {
		originalOutput := logger.Logger.Out
		var logOutput bytes.Buffer
		logger.Logger.SetOutput(&logOutput)

		tx, _, err := types.GenerateTransactionAndSigner(0, types.NewAddressByStr(to), big.NewInt(0), nil)
		require.Nil(t, err)

		event := &common2.UncheckedTxEvent{
			EventType: common2.LocalTxEvent,
			Event:     tx,
		}
		tp.PostUncheckedTxEvent(event)
		time.Sleep(200 * time.Millisecond)

		logger.Logger.SetOutput(originalOutput)
		require.True(t, bytes.Contains(logOutput.Bytes(), []byte("receive invalid local TxEvent")))
	})

	t.Run("test precheck single tx", func(t *testing.T) {
		originalOutput := logger.Logger.Out
		var logOutput bytes.Buffer
		logger.Logger.SetOutput(&logOutput)

		tx, _, err := types.GenerateTransactionAndSigner(0, types.NewAddressByStr(to), big.NewInt(0), nil)
		require.Nil(t, err)

		event := &common2.UncheckedTxEvent{
			EventType: common2.RemoteTxEvent,
			Event:     tx,
		}
		tp.PostUncheckedTxEvent(event)
		time.Sleep(200 * time.Millisecond)
		logger.Logger.SetOutput(originalOutput)
		require.True(t, bytes.Contains(logOutput.Bytes(), []byte("receive invalid remote TxEvent")))
	})

	t.Run("test precheck multi tx Type", func(t *testing.T) {
		tx, err := generateLegacyTx(s, &toAddr, 0, nil, uint64(basicGas), 1, big.NewInt(0))
		require.Nil(t, err)

		transfer := big.NewInt(int64(basicGas))
		setBalance(s.Addr.String(), transfer)
		require.True(t, getBalance(s.Addr.String()).Cmp(transfer) == 0)

		event := createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		validTxs := <-tp.CommitValidTxs()
		require.True(t, validTxs.Local)
		require.Equal(t, 1, len(validTxs.Txs))
		require.True(t, getBalance(s.Addr.String()).Cmp(transfer) == 0)

		gasFeeCap := 1
		tx, err = generateDynamicFeeTx(s, &toAddr, nil, uint64(basicGas), big.NewInt(0), big.NewInt(int64(gasFeeCap)), big.NewInt(0))
		require.Nil(t, err)

		transfer = big.NewInt(int64(basicGas * gasFeeCap))
		setBalance(s.Addr.String(), transfer)
		require.True(t, getBalance(s.Addr.String()).Cmp(transfer) == 0)

		event = createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		validTxs = <-tp.CommitValidTxs()
		require.True(t, validTxs.Local)
		require.Equal(t, 1, len(validTxs.Txs))
		require.True(t, getBalance(s.Addr.String()).Cmp(transfer) == 0)
	})

	t.Run("test precheck multi remote tx event", func(t *testing.T) {
		setBalance(s.Addr.String(), big.NewInt(basicGas))
		txs, err := generateBatchTx(s, 10, 5)
		require.Nil(t, err)
		event := createRemoteTxEvent(txs)
		tp.PostUncheckedTxEvent(event)
		validTxs := <-tp.CommitValidTxs()
		require.False(t, validTxs.Local)
		require.Equal(t, 9, len(validTxs.Txs))
	})

	cancel()
	_, ok := <-tp.validTxsCh
	require.False(t, ok)
	_, ok = <-tp.verifyDataCh
	require.False(t, ok)
}

func TestTxPreCheckMgr_VerifySign(t *testing.T) {
	tp, _, _ := newMockPreCheckMgr()
	defer cleanDb()
	tp.Start()

	t.Run("test precheck illegal sign tx", func(t *testing.T) {
		tx, _, err := types.GenerateWrongSignTransactionAndSigner(true)
		require.Nil(t, err)

		event := &common2.UncheckedTxEvent{
			EventType: common2.LocalTxEvent,
			Event: &common2.TxWithResp{
				Tx:     tx,
				RespCh: make(chan *common2.TxResp),
			},
		}
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, ErrTxSign)
	})

	t.Run("test precheck illegal body tx", func(t *testing.T) {
		tx, sk, err := types.GenerateWrongSignTransactionAndSigner(false)
		require.Nil(t, err)
		require.NotEqual(t, tx.GetFrom().String(), sk.Addr.String())

		event := &common2.UncheckedTxEvent{
			EventType: common2.LocalTxEvent,
			Event: &common2.TxWithResp{
				Tx:     tx,
				RespCh: make(chan *common2.TxResp),
			},
		}
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrInsufficientFunds.Error())
	})
}

func TestTxPreCheckMgr_VerifyData(t *testing.T) {
	tp, _, _ := newMockPreCheckMgr()

	bigInt := new(big.Int)
	bigInt.Exp(big.NewInt(2), big.NewInt(257), nil).Sub(bigInt, big.NewInt(1))

	s, err := types.GenerateSigner()
	require.Nil(t, err)
	toAddr := common.HexToAddress(to)

	defer cleanDb()
	tp.Start()

	t.Run("test precheck too big gasFeeCap", func(t *testing.T) {
		gasFeeCap := bigInt
		tx, err := generateDynamicFeeTx(s, &toAddr, nil, 0, big.NewInt(0), gasFeeCap, big.NewInt(0))
		require.Nil(t, err)

		event := &common2.UncheckedTxEvent{
			EventType: common2.LocalTxEvent,
			Event: &common2.TxWithResp{
				Tx:     tx,
				RespCh: make(chan *common2.TxResp),
			},
		}
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrFeeCapVeryHigh.Error())
	})

	t.Run("test precheck too big gasTipCap", func(t *testing.T) {
		gasTipCap := bigInt
		tx, err := generateDynamicFeeTx(s, &toAddr, nil, 0, big.NewInt(0), big.NewInt(0), gasTipCap)
		require.Nil(t, err)

		event := createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrTipVeryHigh.Error())
	})

	t.Run("test precheck too big gasFeeCap and gasTipCap", func(t *testing.T) {
		gasTipCap := big.NewInt(5000)
		gasFeeCap := new(big.Int).Sub(gasTipCap, big.NewInt(1))
		tx, err := generateDynamicFeeTx(s, &toAddr, nil, 0, big.NewInt(0), gasFeeCap, gasTipCap)
		require.Nil(t, err)

		event := createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrTipAboveFeeCap.Error())
	})

	t.Run("test precheck too small gasFeeCap than baseFee", func(t *testing.T) {
		gasFeeCap := big.NewInt(-1)
		gasTipCap := new(big.Int).Sub(gasFeeCap, big.NewInt(1))
		tx, err := generateDynamicFeeTx(s, &toAddr, nil, 0, big.NewInt(0), gasFeeCap, gasTipCap)
		require.Nil(t, err)

		event := createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrFeeCapTooLow.Error())
	})

	t.Run("test insufficient fund for basic gas balance", func(t *testing.T) {
		tx, _, err := types.GenerateTransactionAndSigner(0, types.NewAddressByStr(to), big.NewInt(0), nil)
		require.Nil(t, err)
		event := createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrInsufficientFunds.Error())
	})

	t.Run("test insufficient fund for intrinsic gas", func(t *testing.T) {
		data := []byte("hello world")
		var gasLimit, gasPrice uint64 = 1, 1
		tx, err := generateLegacyTx(s, &toAddr, 0, data, gasLimit, gasPrice, big.NewInt(0))
		require.Nil(t, err)

		basicBalance := gasLimit * gasPrice
		// make sure the balance is enough for basic gas
		setBalance(s.Addr.String(), big.NewInt(int64(basicBalance)))

		event := createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrIntrinsicGas.Error())
	})

	t.Run("test insufficient fund for transfer", func(t *testing.T) {
		gasPrice := 1
		tx, err := generateLegacyTx(s, &toAddr, 0, nil, uint64(basicGas), uint64(gasPrice), big.NewInt(1))
		require.Nil(t, err)

		// make sure the balance is enough for basic gas
		setBalance(s.Addr.String(), big.NewInt(int64(basicGas)))

		event := createLocalTxEvent(tx)
		tp.PostUncheckedTxEvent(event)
		resp := <-event.Event.(*common2.TxWithResp).RespCh
		require.False(t, resp.Status)
		require.Contains(t, resp.ErrorMsg, core.ErrInsufficientFunds.Error(),
			"when gasFeeCap is not nil, preCheck gasFeeCap*gasLimit+value firstly")
	})
}
