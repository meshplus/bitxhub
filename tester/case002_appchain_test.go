package tester

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
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
	suite.normalNonce = 0
}

//Appchain registers in bitxhub
func (suite *RegisterAppchain) TestRegisterAppchain() {
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2Nonce := uint64(0)

	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))

	pub, err := suite.privKey.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr, err := suite.privKey.PublicKey().Address()
	suite.Require().Nil(err)

	suite.Require().Nil(transfer(suite.Suite, suite.api, addr, 10000000000000))

	args := []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		//pb.String(hexutil.Encode(pub)),
		pb.String(base64.StdEncoding.EncodeToString(pub)),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	chainId := string(gRet.Extra)

	args = []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr2.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(base64.StdEncoding.EncodeToString(pub2)),
		pb.String("reason"),
	}
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	suite.Require().Equal("hyperchain", gjson.Get(string(ret.Ret), "chain_type").String())

	ret, err = invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "GetIdByAddr", pb.String(addr.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	did := genUniqueAppchainDID(addr.String())
	suite.Require().Equal(string(bitxid.DID(did).GetChainDID()), string(ret.Ret))
}

func (suite *RegisterAppchain) TestRegisterAppchain_NoPubKey() {
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2Nonce := uint64(0)

	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))

	addr, err := suite.privKey.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr, 10000000000000))

	args := []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr2.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(""),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	chainId := string(gRet.Extra)

	ret, err = invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	suite.Require().Equal("hyperchain", gjson.Get(string(ret.Ret), "chain_type").String())

	ret, err = invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "GetIdByAddr", pb.String(addr2.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	did := genUniqueAppchainDID(addr2.String())
	suite.Require().Equal(string(bitxid.DID(did).GetChainDID()), string(ret.Ret))
}

func (suite *RegisterAppchain) TestRegisterV2Appchain() {
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2Nonce := uint64(0)

	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	appchainMethod := fmt.Sprintf("appchain%s", addr2.String())
	appchainDID := fmt.Sprintf("did:bitxhub:%s:.", appchainMethod)

	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))

	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	ruleAddr, err := deployContract(suite.api, k2, k2Nonce, bytes)
	suite.Require().Nil(err)
	k2Nonce++

	// register rule
	ret, err := invokeBVMContract(suite.api, k2, k2Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRuleV2",
		pb.String(appchainDID),
		pb.String(ruleAddr.String()),
		pb.String("ruleUrl"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++

	args := []*pb.Arg{
		pb.String(appchainMethod),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(""),
		pb.String("reason"),
		pb.String(ruleAddr.String()),
		pb.String("ruleurl"),
	}
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterV2", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	chainId := string(gRet.Extra)

	ret, err = invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	suite.Require().Equal("hyperchain", gjson.Get(string(ret.Ret), "chain_type").String())

	ret, err = invokeBVMContract(suite.api, suite.privKey, suite.normalNonce, constant.AppchainMgrContractAddr.Address(), "GetIdByAddr", pb.String(addr2.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.normalNonce++
	did := genUniqueAppchainDID(addr2.String())
	suite.Require().Equal(string(bitxid.DID(did).GetChainDID()), string(ret.Ret))
}

//func (suite *RegisterAppchain) TestRegisterV2FabricAppchain() {
//	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
//	suite.Require().Nil(err)
//	k2Nonce := uint64(0)
//
//	addr2, err := k2.PublicKey().Address()
//	suite.Require().Nil(err)
//	appchainMethod := fmt.Sprintf("appchain%s", addr2.String())
//
//	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))
//
//	args := []*pb.Arg{
//		pb.String(appchainMethod),
//		pb.String(docAddr),
//		pb.String(docHash),
//		pb.String(""),
//		pb.String(""),
//		pb.String("fabric"),
//		pb.String("税务链"),
//		pb.String("趣链税务链"),
//		pb.String("1.8"),
//		pb.String(""),
//		pb.String("reason"),
//		pb.String(validator.FabricRuleAddr),
//	}
//	ret, err := invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterV2", args...)
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k2Nonce++
//	gRet := &governance.GovernanceResult{}
//	err = json.Unmarshal(ret.Ret, gRet)
//	suite.Require().Nil(err)
//	proposalId1 := gRet.ProposalID
//
//	path1 := "./test_data/config/node1/key.json"
//	path2 := "./test_data/config/node2/key.json"
//	path3 := "./test_data/config/node3/key.json"
//	keyPath1 := filepath.Join(path1)
//	keyPath2 := filepath.Join(path2)
//	keyPath3 := filepath.Join(path3)
//	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
//	suite.Require().Nil(err)
//	fromAdmin1, err := priAdmin1.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin2, err := priAdmin2.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin3, err := priAdmin3.PublicKey().Address()
//	suite.Require().Nil(err)
//	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
//	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())
//	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
//
//	addr1, err := k1.PublicKey().Address()
//	suite.Require().Nil(err)
//	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
//	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
//
//	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
//		pb.String(proposalId1),
//		pb.String(string(contracts.APPOVED)),
//		pb.String("reason"),
//	)
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	adminNonce1++
//
//	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
//		pb.String(proposalId1),
//		pb.String(string(contracts.APPOVED)),
//		pb.String("reason"),
//	)
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	adminNonce2++
//
//	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
//		pb.String(proposalId1),
//		pb.String(string(contracts.APPOVED)),
//		pb.String("reason"),
//	)
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	adminNonce3++
//
//}

func (suite *RegisterAppchain) TestFetchAppchains() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k1Nonce := uint64(0)
	k2Nonce := uint64(0)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))

	args := []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr1.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(base64.StdEncoding.EncodeToString(pub1)),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id1 := string(gRet.Extra)
	args = []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr2.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(base64.StdEncoding.EncodeToString(pub2)),
		pb.String("reason"),
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
	k1Nonce := uint64(0)
	k2Nonce := uint64(0)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))

	args := []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr1.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(base64.StdEncoding.EncodeToString(pub1)),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	args = []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr2.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(base64.StdEncoding.EncodeToString(pub2)),
		pb.String("reason"),
	}
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Nil(err)
	k2Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id2 := string(gRet.Extra)

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
	k1Nonce := uint64(0)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))

	args := []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr1.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(base64.StdEncoding.EncodeToString(pub1)),
		pb.String("reason"),
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
	adminAddr, err := priAdmin.PublicKey().Address()
	suite.Require().Nil(err)
	adminNonce := uint64(0)

	args = []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", adminAddr.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("管理链"),
		pb.String("趣链管理链"),
		pb.String("1.0"),
		pb.String(base64.StdEncoding.EncodeToString(pub1)),
		pb.String("reason"),
	}
	ret, err = invokeBVMContract(suite.api, priAdmin, adminNonce, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id1 := string(gRet.Extra)

	//UpdateAppchain
	args = []*pb.Arg{
		pb.String(id1),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String(""),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.9"),
		pb.String(string(pub1)),
		pb.String("reason"),
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
