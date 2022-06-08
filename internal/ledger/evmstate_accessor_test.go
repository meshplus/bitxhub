package ledger

import (
	"math/big"
	"testing"

	"github.com/meshplus/bitxhub-kit/hexutil"
	types2 "github.com/meshplus/eth-kit/types"
	"github.com/stretchr/testify/require"
)

func TestCreateBloom(t *testing.T) {

}

func TestNewBxhTxFromEth_NewMessageFromBxh(t *testing.T) {
	rawTx := "0xf86c8085147d35700082520894f927bb571eaab8c9a361ab405c9e4891c5024380880de0b6b3a76400008025a00b8e3b66c1e7ae870802e3ef75f1ec741f19501774bd5083920ce181c2140b99a0040c122b7ebfb3d33813927246cbbad1c6bf210474f5d28053990abff0fd4f53"
	tx := &types2.EthTransaction{}
	tx.Unmarshal(hexutil.Decode(rawTx))
	types2.InitEIP155Signer(big.NewInt(1))
	bxhrx := NewBxhTxFromEth(tx)
	require.NotNil(t, bxhrx)

	msg := NewMessageFromBxh(bxhrx)
	require.NotNil(t, msg)
}
