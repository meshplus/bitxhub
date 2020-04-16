package tester

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/meshplus/bitxhub/internal/constant"

	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-model/pb"
	rpcx "github.com/meshplus/go-bitxhub-client"
	"github.com/stretchr/testify/suite"
)

type Interchain struct {
	suite.Suite
}

func (suite *Interchain) SetupSuite() {
}

func (suite *Interchain) TestHandleIBTP() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)
	k2, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)
	f, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	t, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	c1, err := rpcx.New(
		rpcx.WithPrivateKey(k1),
		rpcx.WithAddrs([]string{
			"localhost:60011",
			"localhost:60012",
			"localhost:60013",
			"localhost:60014",
		}),
	)
	suite.Assert().Nil(err)

	c2, err := rpcx.New(
		rpcx.WithPrivateKey(k2),
		rpcx.WithAddrs([]string{
			"localhost:60011",
			"localhost:60012",
			"localhost:60013",
			"localhost:60014",
		}),
	)
	suite.Assert().Nil(err)

	_, err = c1.InvokeBVMContract(constant.InterchainContractAddr.Address(), "Register",
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("婚姻链"),
		rpcx.String("趣链婚姻链"),
		rpcx.String("1.8"),
	)
	suite.Assert().Nil(err)
	_, err = c2.InvokeBVMContract(constant.InterchainContractAddr.Address(), "Register",
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("fabric"),
		rpcx.String("税务链"),
		rpcx.String("fabric婚姻链"),
		rpcx.String("1.4"),
	)
	suite.Assert().Nil(err)

	// register rule
	_, err = c1.InvokeBVMContract(constant.RuleManagerContractAddr.Address(), "RegisterRule", rpcx.String(f.Hex()), rpcx.String(""))
	suite.Assert().Nil(err)

	ib := &pb.IBTP{From: f.Hex(), To: t.Hex(), Index: 1, Timestamp: time.Now().UnixNano()}
	data, err := ib.Marshal()
	suite.Require().Nil(err)
	_, err = c1.InvokeBVMContract(rpcx.InterchainContractAddr, "HandleIBTP", rpcx.Bytes(data))
	suite.Require().Nil(err)
}

func (suite *Interchain) TestGetIBTPByID() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)
	k2, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)
	f, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	t, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	c1, err := rpcx.New(
		rpcx.WithPrivateKey(k1),
		rpcx.WithAddrs([]string{
			"localhost:60011",
		}),
	)
	suite.Assert().Nil(err)

	c2, err := rpcx.New(
		rpcx.WithPrivateKey(k2),
		rpcx.WithAddrs([]string{
			"localhost:60011",
		}),
	)
	suite.Assert().Nil(err)

	confByte, err := ioutil.ReadFile("./test_data/validator")
	suite.Assert().Nil(err)
	_, err = c1.InvokeBVMContract(rpcx.InterchainContractAddr, "Register",
		rpcx.String(string(confByte)),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("婚姻链"),
		rpcx.String("趣链婚姻链"),
		rpcx.String("1.8"),
	)
	suite.Assert().Nil(err)
	_, err = c2.InvokeBVMContract(rpcx.InterchainContractAddr, "Register",
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("fabric"),
		rpcx.String("税务链"),
		rpcx.String("fabric税务链"),
		rpcx.String("1.8"),
	)
	suite.Assert().Nil(err)

	contractByte, err := ioutil.ReadFile("./test_data/fabric_policy.wasm")
	suite.Assert().Nil(err)
	addr, err := c1.DeployContract(contractByte)
	suite.Assert().Nil(err)
	// register rule
	_, err = c1.InvokeBVMContract(constant.RuleManagerContractAddr.Address(), "RegisterRule", rpcx.String(f.Hex()), rpcx.String(addr.Hex()))
	suite.Assert().Nil(err)

	proof, err := ioutil.ReadFile("./test_data/proof")
	suite.Assert().Nil(err)
	ib := &pb.IBTP{From: f.Hex(), To: t.Hex(), Index: 1, Timestamp: time.Now().UnixNano(), Proof: proof}
	data, err := ib.Marshal()
	suite.Assert().Nil(err)
	_, err = c1.InvokeBVMContract(rpcx.InterchainContractAddr, "HandleIBTP", rpcx.Bytes(data))
	suite.Assert().Nil(err)

	ib.Index = 2
	data, err = ib.Marshal()
	suite.Assert().Nil(err)
	_, err = c1.InvokeBVMContract(rpcx.InterchainContractAddr, "HandleIBTP", rpcx.Bytes(data))
	suite.Assert().Nil(err)

	ib.Index = 3
	data, err = ib.Marshal()
	suite.Assert().Nil(err)
	_, err = c1.InvokeBVMContract(rpcx.InterchainContractAddr, "HandleIBTP", rpcx.Bytes(data))
	suite.Assert().Nil(err)

	ib.Index = 2
	ret, err := c1.InvokeBVMContract(rpcx.InterchainContractAddr, "GetIBTPByID", rpcx.String(ib.ID()))
	suite.Assert().Nil(err)
	suite.Assert().Equal(true, ret.IsSuccess(), string(ret.Ret))
}

func (suite *Interchain) TestAudit() {
	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	c, err := rpcx.New(
		rpcx.WithPrivateKey(k),
		rpcx.WithAddrs([]string{
			"localhost:60011",
			"localhost:60012",
			"localhost:60013",
			"localhost:60014",
		}),
	)
	suite.Require().Nil(err)

	ret, err := c.InvokeBVMContract(rpcx.InterchainContractAddr, "Audit",
		rpcx.String("0x123"),
		rpcx.Int32(1),
		rpcx.String("通过"),
	)
	suite.Require().Nil(err)
	suite.Contains(string(ret.Ret), "caller is not an admin account")
}

func TestInterchain(t *testing.T) {
	suite.Run(t, &Interchain{})
}
