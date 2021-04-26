package contracts

//func TestRuleManager_BindRule(t *testing.T) {
//
//	mockCtl, mockStub, rules, rulesData, chains, chainsData := rulePrepare(t)
//
//
//	//
//	//mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(id1)).Return(boltvm.Error("")).AnyTimes()
//	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chains[0].ID)).Return(boltvm.Success(chainsData[0])).AnyTimes()
//	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
//	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chains[0].ID)).Return(boltvm.Success(nil)).AnyTimes()
//	mockStub.EXPECT().GetAccount(rules[0]).Return(false, nil).AnyTimes()
//	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
//
//	im := &RuleManager{
//		Stub: mockStub,
//	}
//
//	addr := types.NewAddress([]byte{2}).String()
//	res := im.BindRule(id0, addr)
//	assert.True(t, res.Ok)
//
//	res = im.RegisterRule(id1, addr)
//	assert.False(t, res.Ok)
//}
//
//func TestRuleManager_GetRuleAddress(t *testing.T) {
//	mockCtl := gomock.NewController(t)
//	mockStub := mock_stub.NewMockStub(mockCtl)
//
//	id0 := types.NewAddress([]byte{0}).String()
//	id1 := types.NewAddress([]byte{1}).String()
//	rule := Rule{
//		Address: "123",
//		Status:  1,
//	}
//
//	mockStub.EXPECT().GetObject(RuleKey(id0), gomock.Any()).SetArg(1, rule).Return(true)
//	mockStub.EXPECT().GetObject(RuleKey(id1), gomock.Any()).Return(false).MaxTimes(5)
//
//	im := &RuleManager{mockStub}
//
//	res := im.GetRuleAddress(id0, "")
//	assert.True(t, res.Ok)
//	assert.Equal(t, rule.Address, string(res.Result))
//
//	res = im.GetRuleAddress(id1, "fabric")
//	assert.True(t, res.Ok)
//	assert.Equal(t, validator.FabricRuleAddr, string(res.Result))
//
//	res = im.GetRuleAddress(id1, "hyperchain")
//	assert.False(t, res.Ok)
//	assert.Equal(t, "", string(res.Result))
//}
//
//func TestRuleManager_Audit(t *testing.T) {
//	mockCtl := gomock.NewController(t)
//	mockStub := mock_stub.NewMockStub(mockCtl)
//
//	id0 := types.NewAddress([]byte{0}).String()
//	id1 := types.NewAddress([]byte{1}).String()
//	rule := Rule{
//		Address: "123",
//		Status:  1,
//	}
//
//	mockStub.EXPECT().GetObject(RuleKey(id0), gomock.Any()).SetArg(1, rule).Return(true)
//	mockStub.EXPECT().GetObject(RuleKey(id1), gomock.Any()).Return(false)
//	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
//	mockStub.EXPECT().GetObject("audit-record-"+id0, gomock.Any()).SetArg(1, []*ruleRecord{}).Return(true).AnyTimes()
//
//	im := &RuleManager{mockStub}
//
//	res := im.Audit(id1, 0, "")
//	assert.False(t, res.Ok)
//	assert.Equal(t, "this rule does not exist", string(res.Result))
//
//	res = im.Audit(id0, 1, "approve")
//	assert.True(t, res.Ok)
//}
//
//func rulePrepare(t *testing.T) (*RuleManager, *mock_stub.MockStub, []*ruleMgr.Rule, [][]byte, []*appchainMgr.Appchain, [][]byte) {
//	// 1. prepare stub
//	mockCtl := gomock.NewController(t)
//	mockStub := mock_stub.NewMockStub(mockCtl)
//	rm := &RuleManager{
//		Stub: mockStub,
//	}
//
//	// 2. prepare chain
//	var chains []*appchainMgr.Appchain
//	var chainsData [][]byte
//	chainType := []string{string(governance.GovernanceAvailable), string(governance.GovernanceFrozen)}
//
//	chainAdminKeyPath, err := repo.PathRootWithDefault("../../../tester/test_data/admin.json")
//	assert.Nil(t, err)
//	pubKey, err := getPubKey(chainAdminKeyPath)
//	assert.Nil(t, err)
//
//	for i := 0; i < 2; i++ {
//		addr := appchainMethod + types.NewAddress([]byte{byte(i)}).String()
//
//		chain := &appchainMgr.Appchain{
//			Status:        governance.GovernanceStatus(chainType[i]),
//			ID:            addr,
//			Name:          "appchain" + addr,
//			Validators:    "",
//			ConsensusType: "",
//			ChainType:     "fabric",
//			Desc:          "",
//			Version:       "",
//			PublicKey:     pubKey,
//		}
//
//		data, err := json.Marshal(chain)
//		assert.Nil(t, err)
//
//		chainsData = append(chainsData, data)
//		chains = append(chains, chain)
//	}
//
//	// 3. prepare rule
//	var rules []*ruleMgr.Rule
//	var rulesData [][]byte
//
//	for i := 0; i < 2; i++ {
//		rule := &ruleMgr.Rule{
//			Address: types.NewAddress([]byte{2}).String(),
//			ChainId: chains[0].ID,
//			Status: governance.GovernanceAvailable,
//		}
//
//		data, err := json.Marshal(rule)
//		assert.Nil(t, err)
//
//		rulesData = append(rulesData, data)
//		rules = append(rules, rule)
//	}
//
//
//	return rm, mockStub, rules, rulesData, chains, chainsData
//}

//func createMockRepo(t *testing.T) *repo.Repo {
//	key := `-----BEGIN EC PRIVATE KEY-----
//BcNwjTDCxyxLNjFKQfMAc6sY6iJs+Ma59WZyC/4uhjE=
//-----END EC PRIVATE KEY-----`
//
//	privKey, err := libp2pcert.ParsePrivateKey([]byte(key), crypto.Secp256k1)
//	require.Nil(t, err)
//
//	address, err := privKey.PublicKey().Address()
//	require.Nil(t, err)
//
//	return &repo.Repo{
//		Key: &repo.Key{
//			PrivKey: privKey,
//			Address: address.String(),
//		},
//	}
//}
