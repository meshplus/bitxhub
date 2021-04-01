package proof

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/validator/mock_validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/stretchr/testify/require"
)

const (
	from     = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	to       = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	contract = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
)

func TestVerifyPool_CheckProof(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)
	mockEngine := mock_validator.NewMockEngine(mockCtl)

	chain := &appchainMgr.Appchain{
		ID:            from,
		Name:          "appchain A",
		Validators:    "",
		ConsensusType: "rbft",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "11111",
	}

	chainData, err := json.Marshal(chain)
	require.Nil(t, err)

	rl := &contracts.Rule{
		Address: contract,
	}
	rlData, err := json.Marshal(rl)
	require.Nil(t, err)

	mockLedger.EXPECT().GetState(constant.AppchainMgrContractAddr.Address(), gomock.Any()).Return(true, chainData)
	mockLedger.EXPECT().GetState(constant.RuleManagerContractAddr.Address(), gomock.Any()).Return(false, rlData)
	mockEngine.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()

	vp := New(mockLedger, log.NewWithModule("test_verify"))
	vp = &VerifyPool{
		ledger: mockLedger,
		ve:     mockEngine,
		logger: log.NewWithModule("test_verify"),
	}

	engine := vp.ValidationEngine()
	require.NotNil(t, engine)

	txWithNoIBTP := &pb.Transaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  nil,
		Extra: nil,
	}
	txWithNoIBTP.TransactionHash = txWithNoIBTP.Hash()
	ok, err := vp.CheckProof(txWithNoIBTP)
	require.Nil(t, err)
	require.True(t, ok)

	txWithNoExtra := &pb.Transaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, []byte("1")),
		Extra: nil,
	}
	txWithNoExtra.TransactionHash = txWithNoIBTP.Hash()
	ok, err = vp.CheckProof(txWithNoExtra)
	require.NotNil(t, err)
	require.False(t, ok)

	txWithNotEqualProofHash := &pb.Transaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, []byte("222")),
		Extra: []byte("1"),
	}
	txWithNotEqualProofHash.TransactionHash = txWithNoIBTP.Hash()
	ok, err = vp.CheckProof(txWithNotEqualProofHash)
	require.NotNil(t, err)
	require.False(t, ok)

	proof := []byte("test_proof")
	proofHash := sha256.Sum256(proof)

	txWithIBTP := &pb.Transaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, proofHash[:]),
		Extra: proof,
	}
	txWithIBTP.TransactionHash = txWithIBTP.Hash()
	ok, err = vp.CheckProof(txWithIBTP)
	require.Nil(t, err)
	require.True(t, ok)

	proofData, ok := vp.GetProof(*txWithIBTP.Hash())
	require.Nil(t, proofData)
	require.False(t, ok)

	vp.DeleteProof(*txWithIBTP.Hash())

}

func TestVerifyPool_CheckProof2(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockEngine := mock_validator.NewMockEngine(mockCtl)

	chain := &appchainMgr.Appchain{
		Status:        appchainMgr.AppchainAvailable,
		ID:            from,
		Name:          "appchain" + from,
		Validators:    "",
		ConsensusType: "rbft",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "pubkey",
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

	mockEngine.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()

	ibtp := getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, nil)
	ibtpHash := ibtp.Hash()
	hash := sha256.Sum256([]byte(ibtpHash.String()))
	sign := &pb.SignResponse{Sign: make(map[string][]byte)}
	for _, key := range keys {
		signData, err := key.Sign(hash[:])
		require.Nil(t, err)

		address, err := key.PublicKey().Address()
		require.Nil(t, err)
		ok, err := asym.Verify(crypto.Secp256k1, signData[:], hash[:], *address)
		require.Nil(t, err)
		require.True(t, ok)
		sign.Sign[address.String()] = signData
	}
	signData, err := sign.Marshal()
	require.Nil(t, err)
	ibtp.Proof = signData

	ok, err := verifyMultiSign(chain, ibtp, nil)
	require.NotNil(t, err)
	require.False(t, ok)

	chain.Validators = string(addrsData)
	ok, err = verifyMultiSign(chain, ibtp, nil)
	require.Nil(t, err)
	require.True(t, ok)
}

func TestVerifyPool_CheckProof3(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)
	mockEngine := mock_validator.NewMockEngine(mockCtl)

	mockLedger.EXPECT().GetState(constant.AppchainMgrContractAddr.Address(), gomock.Any()).Return(true, []byte("123"))
	mockEngine.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()

	vp := VerifyPool{
		ledger: mockLedger,
		ve:     mockEngine,
		logger: log.NewWithModule("test_verify"),
	}

	proof := []byte("test_proof")
	proofHash := sha256.Sum256(proof)

	txWithIBTP := &pb.Transaction{
		From:  types.NewAddressByStr(from),
		To:    types.NewAddressByStr(to),
		IBTP:  getIBTP(t, 1, pb.IBTP_RECEIPT_SUCCESS, proofHash[:]),
		Extra: proof,
	}

	txWithIBTP.TransactionHash = txWithIBTP.Hash()
	ok, err := vp.CheckProof(txWithIBTP)
	require.NotNil(t, err)
	require.False(t, ok)
}

func getIBTP(t *testing.T, index uint64, typ pb.IBTP_Type, proof []byte) *pb.IBTP {
	ct := &pb.Content{
		SrcContractId: from,
		DstContractId: to,
		Func:          "set",
		Args:          [][]byte{[]byte("Alice")},
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
		From:      from,
		To:        to,
		Payload:   ibtppd,
		Index:     index,
		Type:      typ,
		Proof:     proof,
		Timestamp: time.Now().UnixNano(),
	}
}
