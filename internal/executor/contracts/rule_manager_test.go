package contracts

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

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

func TestRuleManager_BindRule(t *testing.T) {

	rm, mockStub, rules, _, chains, chainsData, account := rulePrepare(t)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Error("get Appchain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("CheckPermission error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Error("is available error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().GetAccount(rules[0].Address).Return(false, nil).Times(1)
	mockStub.EXPECT().GetAccount(rules[0].Address).Return(true, account).AnyTimes()
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesAvailable := make([]*ruleMgr.Rule, 0)
	retRulesAvailable = append(retRulesAvailable, rules[0])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesAvailable).Return(true).Times(1)

	retRulesBindable := make([]*ruleMgr.Rule, 0)
	retRulesBindable = append(retRulesBindable, rules[3])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()

	// getAppchain error
	res := rm.BindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// check permission error
	res = rm.BindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// isAvailable error
	res = rm.BindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// get account error
	res = rm.BindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// first rule && submit proposal error
	res = rm.BindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// has available rule
	res = rm.BindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.BindRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_UnindRule(t *testing.T) {

	rm, mockStub, rules, _, chains, chainsData, _ := rulePrepare(t)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Error("get Appchain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Error("is available error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesAvailable := make([]*ruleMgr.Rule, 0)
	retRulesAvailable = append(retRulesAvailable, rules[0])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesAvailable).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()

	// check permission(get account error) error
	res := rm.UnbindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// isAvailable error
	res = rm.UnbindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// submit proposal error
	res = rm.UnbindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// change status(get object) error
	res = rm.UnbindRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.UnbindRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_FreezeRule(t *testing.T) {

	rm, mockStub, rules, _, chains, chainsData, _ := rulePrepare(t)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Error("get Appchain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Error("is available error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesAvailable := make([]*ruleMgr.Rule, 0)
	retRulesAvailable = append(retRulesAvailable, rules[0])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesAvailable).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()

	// check permission(get account error) error
	res := rm.FreezeRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// isAvailable error
	res = rm.FreezeRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// submit proposal error
	res = rm.FreezeRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// change status(get object) error
	res = rm.FreezeRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.FreezeRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_ActivateRule(t *testing.T) {
	rm, mockStub, rules, _, chains, chainsData, _ := rulePrepare(t)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Error("get Appchain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Error("is available error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesFrozen := make([]*ruleMgr.Rule, 0)
	retRulesFrozen = append(retRulesFrozen, rules[2])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesFrozen).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()

	// check permission(get account error) error
	res := rm.ActivateRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// isAvailable error
	res = rm.ActivateRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// submit proposal error
	res = rm.ActivateRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// change status(get object) error
	res = rm.ActivateRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.ActivateRule(chains[0].ID, rules[0].Address)
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

	retRulesFrozen := make([]*ruleMgr.Rule, 0)
	retRulesFrozen = append(retRulesFrozen, rules[2])
	mockStub.EXPECT().GetObject(RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesFrozen).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()

	// check permission(get account error) error
	res := rm.LogoutRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// isAvailable error
	res = rm.LogoutRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// submit proposal error
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
	assert.False(t, res.Ok, string(res.Result))

	res = rm.CountRules(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.Rules(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.GetRuleByAddr(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.GetAvailableRuleAddr(chains[0].ID, chains[0].ChainType)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.IsAvailableRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))

	// true
	res = rm.CountAvailableRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "1", string(res.Result))

	res = rm.CountRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "4", string(res.Result))

	res = rm.Rules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetRuleByAddr(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetAvailableRuleAddr(chains[0].ID, chains[0].ChainType)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.IsAvailableRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func rulePrepare(t *testing.T) (*RuleManager, *mock_stub.MockStub, []*ruleMgr.Rule, [][]byte, []*appchainMgr.Appchain, [][]byte, *ledger.Account) {
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

	for i := 0; i < 4; i++ {
		rule := &ruleMgr.Rule{
			Address: types.NewAddress([]byte{2}).String(),
			ChainId: chains[0].ID,
			Status:  governance.GovernanceAvailable,
		}
		switch i {
		case 1:
			rule.Status = governance.GovernanceBinding
		case 2:
			rule.Status = governance.GovernanceFrozen
		case 3:
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

func mockAccount(t *testing.T) *ledger.Account {
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
