package tester

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxid"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

const (
	appchainMethod           = "did:bitxhub:appchain1:."
	appchainAdminDIDPrefix   = "did:bitxhub:appchain"
	relaychainAdminDIDPrefix = "did:bitxhub:relayroot"
	relayAdminDID            = "did:bitxhub:relay:0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"
	docAddr                  = "/ipfs/QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
	docHash                  = "QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
)

type RegisterAppchain struct {
	suite.Suite
	api         api.CoreAPI
	privKey     crypto.PrivateKey
	adminKey    crypto.PrivateKey
	from        *types.Address
	normalNonce uint64
}

func (suite *RegisterAppchain) SetupSuite() {
	var err error
	suite.privKey, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	suite.from, err = suite.privKey.PublicKey().Address()
	suite.Require().Nil(err)
	suite.normalNonce = 1
}

// Appchain registers in bitxhub
func (suite *RegisterAppchain) TestRegisterAppchain() {
	pub, err := suite.privKey.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr, err := suite.privKey.PublicKey().Address()
	suite.Require().Nil(err)

	did := genUniqueAppchainDID(addr.String())
	args := []*pb.Arg{
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub)),
	}
	ret, err := invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	registerRes := &contracts.RegisterResult{}
	err = json.Unmarshal(ret.Ret, registerRes)
	suite.Require().Nil(err)
	chainId := registerRes.ChainID

	ret, err = invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	suite.Require().Equal("hyperchain", gjson.Get(string(ret.Ret), "chain_type").String())
}

func (suite *RegisterAppchain) TestFetchAppchains() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k1Nonce := uint64(1)
	k2Nonce := uint64(1)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	did := genUniqueAppchainDID(addr1.String())
	args := []*pb.Arg{
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	id1 := gjson.Get(string(ret.Ret), "chain_id").String()
	did2 := genUniqueAppchainDID(addr2.String())
	args = []*pb.Arg{
		pb.String(did2),
		pb.String(string(bitxid.DID(did2).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	}
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Nil(err)
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Appchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	k2Nonce++

	rec, err := invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "CountAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err := strconv.Atoi(string(rec.Ret))
	suite.Require().Nil(err)
	result := gjson.Parse(string(ret.Ret))
	suite.Require().GreaterOrEqual(num, len(result.Array()))
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "CountAvailableAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err = strconv.Atoi(string(ret.Ret))
	suite.Require().Nil(err)
	suite.Require().EqualValues(0, num)
	k2Nonce++

	//GetAppchain
	ret2, err := invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
	suite.Require().Nil(err)
	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
	appchain := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret2.Ret, appchain)
	suite.Require().Nil(err)
	suite.Require().Equal("hyperchain", appchain.ChainType)
	k2Nonce++
}

func (suite *RegisterAppchain) TestGetPubKeyByChainID() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k1Nonce := uint64(1)
	k2Nonce := uint64(1)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	did := genUniqueAppchainDID(addr1.String())
	args := []*pb.Arg{
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	did2 := genUniqueAppchainDID(addr2.String())
	args = []*pb.Arg{
		pb.String(did2),
		pb.String(string(bitxid.DID(did2).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	}
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Nil(err)
	k2Nonce++
	id2 := gjson.Get(string(ret.Ret), "chain_id").String()

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id2))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++
	appchain2 := appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain2)
	suite.Require().Nil(err)

	//GetPubKeyByChainID
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetPubKeyByChainID", pb.String(string(id2)))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	suite.Require().Equal([]byte(appchain2.PublicKey), ret.Ret)
	k1Nonce++
}

func (suite *RegisterAppchain) TestUpdateAppchains() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	k1Nonce := uint64(1)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)

	did := genUniqueAppchainDID(addr1.String())
	args := []*pb.Arg{
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	//Admin Chain
	path := "./test_data/config/node2/key.json"
	keyPath := filepath.Join(path)
	priAdmin, err := asym.RestorePrivateKey(keyPath, "bitxhub")
	suite.Require().Nil(err)
	pubAdmin, err := priAdmin.PublicKey().Bytes()
	suite.Require().Nil(err)
	adminAddr, err := priAdmin.PublicKey().Address()
	suite.Require().Nil(err)
	adminNonce := uint64(1)

	adminDID := genUniqueAppchainDID(adminAddr.String())
	args = []*pb.Arg{
		pb.String(adminDID),
		pb.String(string(bitxid.DID(adminDID).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("管理链"),
		pb.String("趣链管理链"),
		pb.String("1.0"),
		pb.String(string(pubAdmin)),
	}
	ret, err = invokeBVMContract(suite.api, priAdmin, adminNonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce++

	//UpdateAppchain
	args = []*pb.Arg{
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.9"),
		pb.String(string(pub1)),
	}
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "UpdateAppchain", args...)
	suite.Require().Nil(err)
	// this appchain is registing, can not be updated
	suite.Require().False(ret.IsSuccess())
	k1Nonce++
}

func genUniqueAppchainDID(addr string) string {
	return fmt.Sprintf("%s%s:%s", appchainAdminDIDPrefix, addr, addr)
}

func genUniqueRelaychainDID(addr string) string {
	return fmt.Sprintf("%s:%s", relaychainAdminDIDPrefix, addr)
}
