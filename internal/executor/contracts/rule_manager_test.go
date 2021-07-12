package contracts

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	ledger2 "github.com/meshplus/eth-kit/ledger"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	appchainKeyPath = "../../../tester/test_data/appchain1.json"
)

func TestRuleManager_DefaultRule(t *testing.T) {
	rm, mockStub, rules, _, chains, chainsData, _ := rulePrepare(t)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("CheckPermission error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// 1: register, false
	// 2: CountAvailable, false
	// 3: ChangeStatus1, no rules=====
	// 4: register, false
	// 5: CountAvailable, false
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(5)
	// 1: ChangeStatus1 ok
	retRulesBindable := make([]*ruleMgr.Rule, 0)
	retRulesBindable = append(retRulesBindable, rules[3])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).Times(1)
	// 1: ChangeStatus2, no rules=====
	// 2: register, false
	// 3: CountAvailable, false
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(3)
	// 1: ChangeStatus1 ok
	retRulesBindable1 := make([]*ruleMgr.Rule, 0)
	retRulesBindable1 = append(retRulesBindable1, rules[4])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable1).Return(true).Times(1)
	// 11: ChangeStatus2 ok
	retRulesBinding := make([]*ruleMgr.Rule, 0)
	retRulesBinding = append(retRulesBinding, rules[1])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBinding).Return(true).Times(1)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	// check permission error
	res := rm.DefaultRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	// ChangeStatus1 error
	res = rm.DefaultRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// ChangeStatus2 error
	res = rm.DefaultRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.DefaultRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_UpdateMasterRule(t *testing.T) {
	rm, mockStub, rules, _, chains, chainsData, account := rulePrepare(t)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("CheckPermission error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Error("is available error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().GetAccount(rules[0].Address).Return(account).AnyTimes()
	// BindPre error=========
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)
	// BindPre ok
	retRulesBindable := make([]*ruleMgr.Rule, 0)
	retRulesBindable = append(retRulesBindable, rules[3])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).Times(1)
	// bindRule error=========
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	// BindPre ok
	// bindRule ok(GetRuleByAddr,ChangeStatus) 2
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).Times(3)
	// get master false=====
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	// BindPre ok
	// bindRule ok 2
	// get master ok
	// change status error=====
	retRulesBindable1 := make([]*ruleMgr.Rule, 0)
	retRulesBindable1 = append(retRulesBindable1, rules[4])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable1).Return(true).Times(5)

	// BindPre ok
	// bindRule ok 2
	retRulesBindable2 := make([]*ruleMgr.Rule, 0)
	retRulesBindable2 = append(retRulesBindable2, rules[6])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable2).Return(true).Times(3)
	// get master ok
	// change status ok
	retRulesAvailable := make([]*ruleMgr.Rule, 0)
	retRulesAvailable = append(retRulesAvailable, rules[0])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesAvailable).Return(true).Times(2)

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()

	// no old rule
	// check permission error
	res := rm.UpdateMasterRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// isAvailable error
	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	// BindPre error
	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// bindRule error
	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// get master error
	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// get master ok, change status error
	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_LogoutRule(t *testing.T) {
	rm, mockStub, rules, _, chains, chainsData, _ := rulePrepare(t)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Error("get Appchain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Error("is available error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesBindable := make([]*ruleMgr.Rule, 0)
	retRulesBindable = append(retRulesBindable, rules[3])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).AnyTimes()

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()

	// check permission(get account error) error
	res := rm.LogoutRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// isAvailable error
	res = rm.LogoutRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// change status(get object) error
	res = rm.LogoutRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.LogoutRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_Query(t *testing.T) {
	rm, mockStub, rules, rulesData, chains, _, _ := rulePrepare(t)

	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(5)
	mockStub.EXPECT().Get(RuleKey(chains[0].ID)).Return(false, nil).Times(1)

	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, rules).Return(true).AnyTimes()
	mockStub.EXPECT().Get(RuleKey(chains[0].ID)).Return(true, rulesData[0]).AnyTimes()

	res := rm.CountAvailableRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "0", string(res.Result))

	res = rm.CountRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "0", string(res.Result))

	res = rm.Rules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "", string(res.Result))

	res = rm.GetRuleByAddr(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.GetAvailableRuleAddr(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.IsAvailableRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	// In principle, it is not allowed to have two available, but this situation is only for the convenience of writing unit test
	res = rm.CountAvailableRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "3", string(res.Result))

	res = rm.CountRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "7", string(res.Result))

	res = rm.Rules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetRuleByAddr(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetAvailableRuleAddr(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.IsAvailableRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func rulePrepare(t *testing.T) (*RuleManager, *mock_stub.MockStub, []*ruleMgr.Rule, [][]byte, []*appchainMgr.Appchain, [][]byte, ledger2.IAccount) {
	// 1. prepare stub
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	rm := &RuleManager{
		Stub: mockStub,
	}

	// 2. prepare chain
	var chains []*appchainMgr.Appchain
	var chainsData [][]byte
	chainType := []string{string(governance.GovernanceAvailable), string(governance.GovernanceFrozen)}

	chainAdminKeyPath, err := repo.PathRootWithDefault(appchainKeyPath)
	assert.Nil(t, err)
	pubKey, err := getPubKey(chainAdminKeyPath)
	assert.Nil(t, err)

	for i := 0; i < 2; i++ {
		addr := appchainMethod + types.NewAddress([]byte{byte(i)}).String()

		chain := &appchainMgr.Appchain{
			Status:        governance.GovernanceStatus(chainType[i]),
			ID:            addr,
			Name:          "appchain" + addr,
			Validators:    "",
			ConsensusType: "",
			ChainType:     "hpc",
			Desc:          "",
			Version:       "",
			PublicKey:     pubKey,
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
	}

	// 3. prepare rule
	var rules []*ruleMgr.Rule
	var rulesData [][]byte

	ruleAddr := types.NewAddress([]byte{2}).String()
	for i := 0; i < 7; i++ {
		rule := &ruleMgr.Rule{
			Address: ruleAddr,
			ChainId: chains[0].ID,
			Status:  governance.GovernanceAvailable,
			Master:  true,
		}
		switch i {
		case 1:
			rule.Status = governance.GovernanceBinding
		case 3:
			rule.Status = governance.GovernanceBindable
		case 4:
			rule.Status = governance.GovernanceBindable
		case 5:
			rule.Status = governance.GovernanceAvailable
		case 6:
			rule.Status = governance.GovernanceBindable
		}

		data, err := json.Marshal(rule)
		assert.Nil(t, err)

		rulesData = append(rulesData, data)
		rules = append(rules, rule)
	}

	// 4. prepare account
	account := mockAccount(t)

	return rm, mockStub, rules, rulesData, chains, chainsData, account
}

func mockAccount(t *testing.T) ledger2.IAccount {
	addr := types.NewAddress(bytesutil.LeftPadBytes([]byte{1}, 20))
	code := bytesutil.LeftPadBytes([]byte{1}, 120)
	repoRoot, err := ioutil.TempDir("", "contract")
	require.Nil(t, err)
	blockchainStorage, err := leveldb.New(filepath.Join(repoRoot, "contract"))
	require.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	require.Nil(t, err)
	repo.DefaultConfig()
	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("contract_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), blockchainStorage, ldb, blockFile, accountCache, log.NewWithModule("ledger"))
	account := ldg.GetOrCreateAccount(addr)
	account.SetCodeAndHash(code)

	return account
}

func createMockRepo(t *testing.T) *repo.Repo {
	key := `-----BEGIN EC PRIVATE KEY-----
BcNwjTDCxyxLNjFKQfMAc6sY6iJs+Ma59WZyC/4uhjE=
-----END EC PRIVATE KEY-----`

	privKey, err := libp2pcert.ParsePrivateKey([]byte(key), crypto.Secp256k1)
	require.Nil(t, err)

	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return &repo.Repo{
		Key: &repo.Key{
			PrivKey: privKey,
			Address: address.String(),
		},
	}
}
