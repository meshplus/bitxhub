package proof

import (
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-core/validator/mock_validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/pkg/utils"
	"github.com/stretchr/testify/require"
)

const (
	from          = "1356:chain0:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	to            = "1356:chain1:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	HappyRuleAddr = "0x00000000000000000000000000000000000000a2"
	wasmGasLimit  = 5000000000000000
)

func TestVerifyPool_CheckProof(t *testing.T) {
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	mockEngine := mock_validator.NewMockEngine(mockCtl)

	chain := &appchainMgr.Appchain{
		ID:      from,
		Desc:    "",
		Version: 0,
	}

	chainData, err := json.Marshal(chain)
	require.Nil(t, err)

	rules := make([]*ruleMgr.Rule, 0)
	rl := &ruleMgr.Rule{
		Address: HappyRuleAddr,
		Status:  governance.GovernanceAvailable,
	}
	rules = append(rules, rl)
	rlData, err := json.Marshal(rules)
	require.Nil(t, err)

	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
	stateLedger.EXPECT().GetState(constant.AppchainMgrContractAddr.Address(), gomock.Any()).Return(true, chainData)
	stateLedger.EXPECT().GetState(constant.RuleManagerContractAddr.Address(), gomock.Any()).Return(true, rlData)
	mockEngine.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, uint64(0), nil).AnyTimes()

	vp := &VerifyPool{
		ledger:    mockLedger,
		ve:        mockEngine,
		logger:    log.NewWithModule("test_verify"),
		bitxhubID: "1356",
	}

	engine := vp.ValidationEngine()
	require.NotNil(t, engine)

	proof := []byte("1")
	proofHash := sha256.Sum256(proof)

	normalTx := &pb.BxhTransaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, proofHash[:]),
		Extra: []byte("1"),
	}

	normalTx.TransactionHash = normalTx.Hash()
	ok, _, err := vp.CheckProof(normalTx)
	require.Nil(t, err)
	require.True(t, ok)

	txWithNoIBTP := &pb.BxhTransaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  nil,
		Extra: nil,
	}
	txWithNoIBTP.TransactionHash = txWithNoIBTP.Hash()
	ok, _, err = vp.CheckProof(txWithNoIBTP)
	require.Nil(t, err)
	require.True(t, ok)

	txWithNoExtra := &pb.BxhTransaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, []byte("1")),
		Extra: nil,
	}
	txWithNoExtra.TransactionHash = txWithNoIBTP.Hash()
	ok, _, err = vp.CheckProof(txWithNoExtra)
	require.NotNil(t, err)
	require.False(t, ok)

	txWithNotEqualProofHash := &pb.BxhTransaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, []byte("222")),
		Extra: []byte("1"),
	}
	txWithNotEqualProofHash.TransactionHash = txWithNoIBTP.Hash()
	ok, _, err = vp.CheckProof(txWithNotEqualProofHash)
	require.NotNil(t, err)
	require.False(t, ok)

	proof = []byte("test_proof")
	proofHash = sha256.Sum256(proof)

	stateLedger.EXPECT().GetState(constant.AppchainMgrContractAddr.Address(), gomock.Any()).Return(true, chainData)
	stateLedger.EXPECT().GetState(constant.RuleManagerContractAddr.Address(), gomock.Any()).Return(false, rlData).Times(1)

	txWithIBTP := &pb.BxhTransaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, proofHash[:]),
		Extra: proof,
	}
	txWithIBTP.TransactionHash = txWithIBTP.Hash()
	ok, _, err = vp.CheckProof(txWithIBTP)
	require.False(t, ok)
	require.Equal(t, true, strings.Contains(err.Error(), NoBindRule))

	proofData, ok := vp.GetProof(*txWithIBTP.Hash())
	require.Nil(t, proofData)
	require.False(t, ok)

	vp.DeleteProof(*txWithIBTP.Hash())

}

func TestVerifyPool_CheckProof2(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockEngine := mock_validator.NewMockEngine(mockCtl)

	chain := &appchainMgr.Appchain{
		Status:  governance.GovernanceAvailable,
		ID:      from,
		Desc:    "",
		Version: 0,
	}

	keys := make([]crypto.PrivateKey, 0, 4)
	var bv contracts.BxhValidators
	addrs := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		keyPair, err := asym.GenerateKeyPair(crypto.Secp256k1)
		require.Nil(t, err)
		keys = append(keys, keyPair)
		address, err := keyPair.PublicKey().Address()
		require.Nil(t, err)
		addrs = append(addrs, address.String())
	}

	bv.Addresses = addrs
	addrsData, err := json.Marshal(bv)
	require.Nil(t, err)

	mockEngine.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, uint64(0), nil).AnyTimes()

	ibtp := getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, nil)
	txStatus := pb.TransactionStatus_SUCCESS
	hash, err := utils.EncodePackedAndHash(ibtp, txStatus)
	require.Nil(t, err)
	bxhProof := &pb.BxhProof{TxStatus: txStatus}
	for _, key := range keys {
		signData, err := key.Sign(hash[:])
		require.Nil(t, err)

		address, err := key.PublicKey().Address()
		require.Nil(t, err)
		ok, err := asym.Verify(crypto.Secp256k1, signData[:], hash[:], *address)
		require.Nil(t, err)
		require.True(t, ok)
		bxhProof.MultiSign = append(bxhProof.MultiSign, signData)
	}
	signData, err := bxhProof.Marshal()
	require.Nil(t, err)
	proof := signData

	vp := &VerifyPool{
		ve:        mockEngine,
		logger:    log.NewWithModule("test_verify"),
		bitxhubID: "1356",
	}

	ok, _, err := vp.verifyMultiSign(chain, ibtp, nil)
	require.NotNil(t, err)
	require.False(t, ok)

	chain.TrustRoot = addrsData
	ok, _, err = vp.verifyMultiSign(chain, ibtp, proof)
	require.Nil(t, err)
	require.True(t, ok)
}

func TestVerifyPool_CheckProof3(t *testing.T) {
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	mockEngine := mock_validator.NewMockEngine(mockCtl)

	stateLedger.EXPECT().GetState(constant.AppchainMgrContractAddr.Address(), gomock.Any()).Return(true, []byte("123"))
	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
	mockEngine.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, uint64(0), nil).AnyTimes()

	vp := VerifyPool{
		ledger: mockLedger,
		ve:     mockEngine,
		logger: log.NewWithModule("test_verify"),
	}

	proof := []byte("test_proof")
	proofHash := sha256.Sum256(proof)

	txWithIBTP := &pb.BxhTransaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, proofHash[:]),
		Extra: proof,
	}

	txWithIBTP.TransactionHash = txWithIBTP.Hash()
	ok, _, err := vp.CheckProof(txWithIBTP)
	require.NotNil(t, err)
	require.False(t, ok)
}

func getIBTP(t *testing.T, index uint64, typ pb.IBTP_Type, proof []byte) *pb.IBTP {
	ct := &pb.Content{
		Func: "set",
		Args: [][]byte{[]byte("Alice")},
	}
	c, err := ct.Marshal()
	require.Nil(t, err)

	pd := pb.Payload{
		Encrypted: false,
		Content:   c,
	}
	ibtppd, err := pd.Marshal()
	require.Nil(t, err)

	return &pb.IBTP{
		From:    from,
		To:      to,
		Payload: ibtppd,
		Index:   index,
		Type:    typ,
		Proof:   proof,
	}
}
