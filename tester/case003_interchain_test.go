package tester

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

type Interchain struct {
	suite.Suite
	api api.CoreAPI
}

func (suite *Interchain) SetupSuite() {
}

func (suite *Interchain) TestHandleIBTP() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)
	k2, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)
	f, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	t, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register",
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

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Register",
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

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	addr, err := deployContract(suite.api, k1, bytes)
	suite.Require().Nil(err)

	// register rule
	ret, err = invokeBVMContract(suite.api, k1, constant.RuleManagerContractAddr.Address(), "RegisterRule", pb.String(f.Hex()), pb.String(addr.Hex()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	ib := &pb.IBTP{From: f.Hex(), To: t.Hex(), Index: 1, Timestamp: time.Now().UnixNano()}
	data, err := ib.Marshal()
	suite.Require().Nil(err)

	ret, err = invokeBVMContract(suite.api, k1, constant.InterchainContractAddr.Address(), "HandleIBTP", pb.Bytes(data))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}

func (suite *Interchain) TestGetIBTPByID() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)
	k2, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)
	f, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	t, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	confByte, err := ioutil.ReadFile("./test_data/validator")
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register",
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

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Register",
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

	contractByte, err := ioutil.ReadFile("./test_data/fabric_policy.wasm")
	suite.Require().Nil(err)
	addr, err := deployContract(suite.api, k1, contractByte)
	suite.Require().Nil(err)

	// register rule
	_, err = invokeBVMContract(suite.api, k1, constant.RuleManagerContractAddr.Address(), "RegisterRule", pb.String(f.Hex()), pb.String(addr.Hex()))
	suite.Require().Nil(err)

	proof, err := ioutil.ReadFile("./test_data/proof")
	suite.Require().Nil(err)
	ib := &pb.IBTP{From: f.Hex(), To: t.Hex(), Index: 1, Payload: []byte("111"), Timestamp: time.Now().UnixNano(), Proof: proof}
	data, err := ib.Marshal()
	suite.Require().Nil(err)
	receipt, err := invokeBVMContract(suite.api, k1, constant.InterchainContractAddr.Address(), "HandleIBTP", pb.Bytes(data))
	suite.Require().Nil(err)
	suite.Require().EqualValues(true, receipt.IsSuccess(), string(receipt.Ret))

	ib.Index = 2
	data, err = ib.Marshal()
	suite.Require().Nil(err)
	receipt, err = invokeBVMContract(suite.api, k1, constant.InterchainContractAddr.Address(), "HandleIBTP", pb.Bytes(data))
	suite.Require().Nil(err)
	suite.Require().EqualValues(true, receipt.IsSuccess(), string(receipt.Ret))

	ib.Index = 3
	data, err = ib.Marshal()
	suite.Assert().Nil(err)
	_, err = invokeBVMContract(suite.api, k1, constant.InterchainContractAddr.Address(), "HandleIBTP", pb.Bytes(data))
	suite.Assert().Nil(err)

	ib.Index = 2
	ret, err = invokeBVMContract(suite.api, k1, constant.InterchainContractAddr.Address(), "GetIBTPByID", pb.String(ib.ID()))
	suite.Assert().Nil(err)
	suite.Assert().Equal(true, ret.IsSuccess(), string(ret.Ret))
}

func (suite *Interchain) TestAudit() {
	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k, constant.AppchainMgrContractAddr.Address(), "Audit",
		pb.String("0x123"),
		pb.Int32(1),
		pb.String("通过"),
	)
	suite.Require().Nil(err)
	suite.Contains(string(ret.Ret), "caller is not an admin account")
}

func (suite *Interchain) TestInterchain() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register",
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

	appchain := Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain)
	suite.Require().Nil(err)
	id1 := appchain.ID

	ret, err = invokeBVMContract(suite.api, k1, constant.InterchainContractAddr.Address(), "Interchain")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Equal(id1, gjson.Get(string(ret.Ret), "id").String())
	suite.Require().Equal("", gjson.Get(string(ret.Ret), "interchain_counter").String())
	suite.Require().Equal("", gjson.Get(string(ret.Ret), "receipt_counter").String())
	suite.Require().Equal("", gjson.Get(string(ret.Ret), "source_receipt_counter").String())
}

func (suite *Interchain) TestRegister() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, constant.InterchainContractAddr.Address(), "Register")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}

func TestInterchain(t *testing.T) {
	suite.Run(t, &Interchain{})
}
