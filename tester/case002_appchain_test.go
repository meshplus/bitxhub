package tester

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

type RegisterAppchain struct {
	suite.Suite
	api     api.CoreAPI
	privKey crypto.PrivateKey
	from    types.Address
}

type Appchain struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Validators    string `json:"validators"`
	ConsensusType int32  `json:"consensus_type"`
	// 0 => registered, 1 => approved, -1 => rejected
	Status    int32  `json:"status"`
	ChainType string `json:"chain_type"`
	Desc      string `json:"desc"`
	Version   string `json:"version"`
	PublicKey string `json:"public_key"`
}

func (suite *RegisterAppchain) SetupSuite() {
	var err error
	suite.privKey, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	suite.from, err = suite.privKey.PublicKey().Address()
	suite.Require().Nil(err)
}

// Appchain registers in bitxhub
func (suite *RegisterAppchain) TestRegisterAppchain() {
	pub, err := suite.privKey.PublicKey().Bytes()
	suite.Require().Nil(err)

	args := []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub)),
	}

	ret, err := invokeBVMContract(suite.api, suite.privKey, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Equal("hyperchain", gjson.Get(string(ret.Ret), "chain_type").String())
}

func (suite *RegisterAppchain) TestFetchAppchains() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	args := []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	}
	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	appchain := Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain)
	suite.Require().Nil(err)
	id1 := appchain.ID

	args = []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	}
	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Nil(err)

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Appchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	rec, err := invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "CountAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err := strconv.Atoi(string(rec.Ret))
	suite.Require().Nil(err)
	result := gjson.Parse(string(ret.Ret))
	suite.Require().GreaterOrEqual(num, len(result.Array()))

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "CountApprovedAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err = strconv.Atoi(string(ret.Ret))
	suite.Require().Nil(err)
	suite.Require().EqualValues(0, num)

	//FetchAuditRecords
	ret, err = invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "FetchAuditRecords", pb.String(string(id1)))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	//AppChain
	ret, err = invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Appchain")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	//GetAppchain
	ret2, err := invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(string(id1)))
	suite.Require().Nil(err)
	suite.Require().True(ret2.IsSuccess())
	suite.Require().Equal(ret.Ret, ret2.Ret)
}

func (suite *RegisterAppchain) TestGetPubKeyByChainID() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	args := []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	}
	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	args = []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	}
	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Nil(err)

	appchain2 := Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain2)
	suite.Require().Nil(err)
	id2 := appchain2.ID

	//GetPubKeyByChainID
	ret, err = invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "GetPubKeyByChainID", pb.String(string(id2)))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	suite.Require().Equal([]byte(appchain2.PublicKey), ret.Ret)
}

func (suite *RegisterAppchain) TestUpdateAppchains() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)

	args := []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	}
	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	appchain := Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain)
	suite.Require().Nil(err)
	id1 := appchain.ID

	//Admin Chain
	path := "./test_data/config/node1/key.json"
	keyPath := filepath.Join(path)
	priAdmin, err := asym.RestorePrivateKey(keyPath, "bitxhub")
	suite.Require().Nil(err)
	pubAdmin, err := priAdmin.PublicKey().Bytes()
	suite.Require().Nil(err)

	args = []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("管理链"),
		pb.String("趣链管理链"),
		pb.String("1.0"),
		pb.String(string(pubAdmin)),
	}
	ret, err = invokeBVMContract(suite.api, priAdmin, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	//Audit
	ret, err = invokeBVMContract(suite.api, priAdmin, constant.AppchainMgrContractAddr.Address(), "Audit",
		pb.String(string(id1)),
		pb.Int32(1),
		pb.String("通过"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	//UpdateAppchain
	args = []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.9"),
		pb.String(string(pub1)),
	}
	ret, err = invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "UpdateAppchain", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
}

func TestRegisterAppchain(t *testing.T) {
	suite.Run(t, &RegisterAppchain{})
}
