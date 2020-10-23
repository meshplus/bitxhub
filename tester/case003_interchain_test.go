package tester

import (
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
)

type Interchain struct {
	suite.Suite
	api api.CoreAPI
}

func (suite *Interchain) SetupSuite() {
}

func (suite *Interchain) TestHandleIBTP() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	f, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	t, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := uint64(1)
	k2Nonce := uint64(1)
	ibtpNonce := uint64(1)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("fabric"),
		pb.String("税务链"),
		pb.String("fabric婚姻链"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	k2Nonce++

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	addr, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++

	// register rule
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule", pb.String(f.String()), pb.String(addr.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	k1Nonce++

	proof := []byte("true")
	proofHash := sha256.Sum256(proof)
	ib := &pb.IBTP{From: f.String(), To: t.String(), Index: ibtpNonce, Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err := genIBTPTransaction(k1, ib)
	suite.Require().Nil(err)

	tx.Extra = proof
	ret, err = sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	ibtpNonce++
}

func (suite *Interchain) TestGetIBTPByID() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	f, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	t, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := uint64(1)
	k2Nonce := uint64(1)
	ibtpNonce := uint64(1)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	confByte, err := ioutil.ReadFile("./test_data/validator")
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(string(confByte)),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("fabric"),
		pb.String("税务链"),
		pb.String("fabric税务链"),
		pb.String("1.8"),
		pb.String(string(pub2)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++

	contractByte, err := ioutil.ReadFile("./test_data/fabric_policy.wasm")
	suite.Require().Nil(err)
	addr, err := deployContract(suite.api, k1, k1Nonce, contractByte)
	suite.Require().Nil(err)
	k1Nonce++

	// register rule
	_, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule", pb.String(f.String()), pb.String(addr.String()))
	suite.Require().Nil(err)
	k1Nonce++

	proof, err := ioutil.ReadFile("./test_data/proof")
	suite.Require().Nil(err)

	proofHash := sha256.Sum256(proof)
	ib := &pb.IBTP{From: f.String(), To: t.String(), Index: ibtpNonce, Payload: []byte("111"), Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err := genIBTPTransaction(k1, ib)
	suite.Require().Nil(err)
	tx.Extra = proof
	receipt, err := sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().EqualValues(true, receipt.IsSuccess(), string(receipt.Ret))
	ibtpNonce++

	ib2 := &pb.IBTP{From: f.String(), To: t.String(), Index: ibtpNonce, Payload: []byte("111"), Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err = genIBTPTransaction(k1, ib2)
	suite.Require().Nil(err)
	tx.Extra = proof
	receipt, err = sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().EqualValues(true, receipt.IsSuccess(), string(receipt.Ret))
	ibtpNonce++

	ib3 := &pb.IBTP{From: f.String(), To: t.String(), Index: ibtpNonce, Payload: []byte("111"), Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err = genIBTPTransaction(k1, ib3)
	suite.Require().Nil(err)
	tx.Extra = proof
	receipt, err = sendTransactionWithReceipt(suite.api, tx)
	suite.Assert().Nil(err)
	ibtpNonce++

	ib.Index = 2
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.InterchainContractAddr.Address(), "GetIBTPByID", pb.String(ib.ID()))
	suite.Assert().Nil(err)
	suite.Assert().Equal(true, ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
}

func (suite *Interchain) TestAudit() {
	k, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	kNonce := uint64(1)

	ret, err := invokeBVMContract(suite.api, k, kNonce, constant.AppchainMgrContractAddr.Address(), "Audit",
		pb.String("0x123"),
		pb.Int32(1),
		pb.String("通过"),
	)
	suite.Require().Nil(err)
	suite.Contains(string(ret.Ret), "caller is not an admin account")
	kNonce++
}

func (suite *Interchain) TestInterchain() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k1Nonce := uint64(1)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	appchain := Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain)
	suite.Require().Nil(err)
	id1 := appchain.ID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.InterchainContractAddr.Address(), "Interchain")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	ic := &pb.Interchain{}
	err = ic.Unmarshal(ret.Ret)
	suite.Require().Nil(err)
	suite.Require().Equal(id1, ic.ID)
	suite.Require().Equal(0, len(ic.InterchainCounter))
	suite.Require().Equal(0, len(ic.ReceiptCounter))
	suite.Require().Equal(0, len(ic.SourceReceiptCounter))
	k1Nonce++
}

func (suite *Interchain) TestRegister() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k1Nonce := uint64(1)

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.InterchainContractAddr.Address(), "Register")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
}

func TestInterchain(t *testing.T) {
	suite.Run(t, &Interchain{})
}
