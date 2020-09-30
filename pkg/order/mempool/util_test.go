package mempool

import (
	"fmt"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAccount(t *testing.T) {
	ast := assert.New(t)
	privKey := genPrivKey()
	address, _ := privKey.PublicKey().Address()
	tx := constructIBTPTx(uint64(1), &privKey)
	addr, err := getAccount(tx)
	ast.Nil(err)
	expectedAddr := fmt.Sprintf("%s-%s-%d", address, address, pb.IBTP_INTERCHAIN)
	ast.Equal(expectedAddr, addr)

	data := &pb.TransactionData{
		Payload: []byte("test"),
	}
	tx = &pb.Transaction{
		To: InterchainContractAddr,
		Data:data,
	}
	_, err = getAccount(tx)
	ast.NotNil(err.Error(), "unmarshal invoke payload faile")
}
