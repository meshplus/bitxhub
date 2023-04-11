package utils

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/crypto"
	crypto3 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/meshplus/bitxhub-core/tss/conversion"
	"github.com/meshplus/bitxhub-core/tss/keysign"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/pkg/tssmgr"
	"github.com/sirupsen/logrus"
)

var AuditEventStrHash = types.NewHash([]byte(fmt.Sprintf("event%d%d%d%d%d%d%d%d",
	pb.Event_AUDIT_PROPOSAL,
	pb.Event_AUDIT_APPCHAIN,
	pb.Event_AUDIT_RULE,
	pb.Event_AUDIT_SERVICE,
	pb.Event_AUDIT_NODE,
	pb.Event_AUDIT_ROLE,
	pb.Event_AUDIT_INTERCHAIN,
	pb.Event_AUDIT_DAPP)))

func GetIBTPSign(ledger *ledger.Ledger, id string, isReq bool, privKey crypto2.PrivateKey) (string, []byte, error) {
	ibtp, err := GetIBTP(ledger, id, isReq)
	if err != nil {
		return "", nil, fmt.Errorf("get ibtp %s isReq %v: %w", id, isReq, err)
	}

	txStatus, err := GetTxStatus(ledger, id)
	if err != nil {
		return "", nil, fmt.Errorf("get tx status of ibtp %s isReq %v: %w", id, isReq, err)
	}

	hash, err := EncodePackedAndHash(ibtp, txStatus)
	if err != nil {
		return "", nil, fmt.Errorf("encode packed and hash for ibtp %s isReq %v: %w", id, isReq, err)
	}

	sign, err := privKey.Sign(hash)
	if err != nil {
		return "", nil, fmt.Errorf("bitxhub sign ibtp %s isReq %v: %w", id, isReq, err)
	}

	addr, err := privKey.PublicKey().Address()
	if err != nil {
		return "", nil, fmt.Errorf("get address from public key failed: %w", err)
	}

	return addr.String(), sign, nil
}

// GetIBTPTssSign return:
// - signature data
// - blame nodes id list
// - error
func GetIBTPTssSign(tssMgr *tssmgr.TssMgr, ledger *ledger.Ledger, content string, isReq bool, signers []string, randomN string) ([]byte, []string, error) {
	msgs, err := getMsgToSign(ledger, content, isReq)
	if err != nil {
		return nil, nil, fmt.Errorf("get msg to sign err: %w", err)
	}

	return tssMgr.KeySign(signers, msgs, randomN)
}

func NotifyNotTssParties(tssMgr *tssmgr.TssMgr, ledger *ledger.Ledger, content string, isReq bool, signers []string, randomN string) error {
	msgID, err := getTssMsgID(tssMgr, ledger, content, isReq, signers, randomN)
	if err != nil {
		return fmt.Errorf("get tss msg id err: %w", err)
	}
	err = tssMgr.SetTssRoundDone(msgID, true)
	return err
}

func getTssMsgID(tssMgr *tssmgr.TssMgr, ledger *ledger.Ledger, content string, isReq bool, signers []string, randomN string) (string, error) {
	msgs, err := getMsgToSign(ledger, content, isReq)
	if err != nil {
		return "", fmt.Errorf("get msg to sign err: %w", err)
	}

	// 1. get pool pubKey
	_, pk, err := tssMgr.GetTssPubkey()
	if err != nil {
		return "", fmt.Errorf("get tss pubkey error: %w", err)
	}

	// 2. get signers pk
	tssInfo, err := tssMgr.GetTssInfo()
	if err != nil {
		return "", fmt.Errorf("fail to get keygen parties pk map error: %w", err)
	}
	signersPk := make([]crypto3.PubKey, 0)
	for _, id := range signers {
		data, ok := tssInfo.PartiesPkMap[id]
		if !ok {
			return "", fmt.Errorf("party %s is not keygen party", id)
		}
		pk, err := conversion.GetPubKeyFromPubKeyData(data)
		if err != nil {
			return "", fmt.Errorf("fail to conversion pubkeydata to pubkey: %w", err)
		}
		signersPk = append(signersPk, pk)
	}

	// 3, new req to sign
	keysignReq := keysign.NewRequest(pk, msgs, signersPk, randomN)
	msgID, err := keysignReq.RequestToMsgId()
	if err != nil {
		return "", err
	}
	return msgID, nil
}

func getMsgToSign(ledger *ledger.Ledger, content string, isReq bool) ([]string, error) {
	ids := strings.Split(strings.Replace(content, " ", "", -1), ",")
	msgs := make([]string, 0)
	for _, id := range ids {
		ibtp, err := GetIBTP(ledger, id, isReq)
		if err != nil {
			return nil, fmt.Errorf("get ibtp %s isReq %v: %w", id, isReq, err)
		}
		txStatus, err := GetTxStatus(ledger, id)
		if err != nil {
			return nil, fmt.Errorf("get tx status of ibtp %s isReq %v: %w", id, isReq, err)
		}

		hash, err := EncodePackedAndHash(ibtp, txStatus)
		if err != nil {
			return nil, fmt.Errorf("encode packed and hash for ibtp %s isReq %v: %w", id, isReq, err)
		}

		msgs = append(msgs, base64.StdEncoding.EncodeToString(hash))
	}

	return msgs, nil
}

func VerifyTssSigns(signData []byte, pub *ecdsa.PublicKey, l logrus.FieldLogger) error {
	signs := make([]conversion.Signature, 0)
	if err := json.Unmarshal(signData, &signs); err != nil {
		return fmt.Errorf("unmarshal signData err: %w", err)
	}

	for _, sign := range signs {
		msgData, err := base64.StdEncoding.DecodeString(sign.Msg)
		if err != nil {
			return fmt.Errorf("sign convert Msg error: %s", err)
		}
		rData, err := base64.StdEncoding.DecodeString(sign.R)
		if err != nil {
			return fmt.Errorf("sign convert R error: %s", err)
		}
		sData, err := base64.StdEncoding.DecodeString(sign.S)
		if err != nil {
			return fmt.Errorf("sign convert S error: %s", err)
		}
		if !ecdsa.Verify(pub, msgData, new(big.Int).SetBytes(rData), new(big.Int).SetBytes(sData)) {
			return fmt.Errorf("fail to verify sign")
		}
	}

	return nil
}

func GetIBTP(ledger *ledger.Ledger, id string, isReq bool) (*pb.IBTP, error) {
	key := contracts.IndexMapKey(id)
	if !isReq {
		key = contracts.IndexReceiptMapKey(id)
	}

	var val []byte
	var ok bool
	var tx pb.Transaction
	var err error
	if err := retry.Retry(func(attempt uint) error {
		ok, val = ledger.Copy().GetState(constant.InterchainContractAddr.Address(), []byte(key))
		if !ok {
			return fmt.Errorf("cannot get the tx hash which contains the IBTP, id:%s, key:%s", id, key)
		}

		var hash types.Hash
		if err := json.Unmarshal(val, &hash); err != nil {
			return fmt.Errorf("unmarshal hash error: %w", err)
		}

		tx, err = ledger.GetTransaction(&hash)
		if err != nil {
			return fmt.Errorf("get transaction %s from ledger failed: %w", hash.String(), err)
		}

		return nil
	}, strategy.Wait(100*time.Millisecond), strategy.Limit(5),
	); err != nil {
		return nil, fmt.Errorf("retry error: %v", err)
	}

	return tx.GetIBTP(), nil
}

// TODO: support global status
func GetTxStatus(ledger *ledger.Ledger, id string) (pb.TransactionStatus, error) {
	var (
		ok  bool
		val []byte
	)
	ok, val = ledger.Copy().GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TxInfoKey(id)))
	if !ok {
		// try to find globalID
		ok, val = ledger.Copy().GetState(constant.TransactionMgrContractAddr.Address(), []byte(id))
		if !ok {
			return 0, fmt.Errorf("no tx status found for ibtp %s", id)
		}
		globalId := string(val)
		ok, val = ledger.Copy().GetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.GlobalTxInfoKey(globalId)))
		if !ok {
			return 0, fmt.Errorf("no tx status found for multi ibtp %s", id)
		}

		// every child one2multi IBTP follow the global state
		txInfo := contracts.TransactionInfo{}
		if err := json.Unmarshal(val, &txInfo); err != nil {
			return 0, fmt.Errorf("unmarshal global transaction info error: %w", err)
		}
		return txInfo.GlobalState, nil
	}
	var record pb.TransactionRecord
	if err := record.Unmarshal(val); err != nil {
		return 0, fmt.Errorf("unmarshal transaction record error: %w", err)
	}

	return record.Status, nil
}

func EncodePackedAndHash(ibtp *pb.IBTP, txStatus pb.TransactionStatus) ([]byte, error) {
	var (
		data []byte
		pd   pb.Payload
	)

	data = append(data, []byte(ibtp.From)...)
	data = append(data, []byte(ibtp.To)...)
	data = append(data, uint64ToBytesInBigEndian(ibtp.Index)...)
	data = append(data, uint64ToBytesInBigEndian(uint64(ibtp.Type))...)

	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal ibtp payload error: %w", err)
	}

	data = append(data, pd.Hash...)
	data = append(data, uint64ToBytesInBigEndian(uint64(txStatus))...)

	hash := crypto.Keccak256(data)

	return hash[:], nil
}

func uint64ToBytesInBigEndian(i uint64) []byte {
	bytes := make([]byte, 8)

	binary.BigEndian.PutUint64(bytes, i)

	return bytes
}

func AddAuditPermitBloom(bloom *types.Bloom, relatedChains, relatedNodes map[string][]byte) {
	if bloom == nil {
		bloom = &types.Bloom{}
	}
	bloom.Add(AuditEventStrHash.Bytes())
	for k := range relatedChains {
		bloom.Add(types.NewHash([]byte(k)).Bytes())
	}
	for k := range relatedNodes {
		bloom.Add(types.NewHash([]byte(k)).Bytes())
	}
}

func TestAuditPermitBloom(bloom *types.Bloom, relatedChains, relatedNodes map[string]struct{}) bool {
	if !bloom.Test(AuditEventStrHash.Bytes()) {
		return false
	}
	for k := range relatedChains {
		if bloom.Test(types.NewHash([]byte(k)).Bytes()) {
			return true
		}
	}
	for k := range relatedNodes {
		if bloom.Test(types.NewHash([]byte(k)).Bytes()) {
			return true
		}
	}
	return false
}

func IsTssReq(req *pb.GetSignsRequest) bool {
	switch req.Type {
	case pb.GetSignsRequest_TSS_IBTP_REQUEST, pb.GetSignsRequest_TSS_IBTP_RESPONSE:
		return true
	default:
		return false
	}
}

type Pool struct {
	queue chan struct{}
	wg    *sync.WaitGroup
}

func NewGoPool(size int) *Pool {
	if size <= 0 {
		size = 1
	}
	return &Pool{
		queue: make(chan struct{}, size),
		wg:    &sync.WaitGroup{},
	}
}

func (p *Pool) Add() {
	p.queue <- struct{}{}
	p.wg.Add(1)
}

func (p *Pool) Done() {
	<-p.queue
	p.wg.Done()
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func PrettyPrint(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Println(v)
		return
	}

	var out bytes.Buffer
	err = json.Indent(&out, b, "", "  ")
	if err != nil {
		fmt.Println(v)
		return
	}
	fmt.Println(out.String())
}

func FindUnmatched(child []string, all []string) []string {
	childMap := make(map[string]bool)
	for _, val := range child {
		childMap[val] = true
	}

	var unmatched []string
	for _, val := range all {
		if _, ok := childMap[val]; !ok {
			unmatched = append(unmatched, val)
		}
	}
	return unmatched
}

func IsContained(child string, all []string) bool {
	for _, val := range all {
		if val == child {
			return true
		}
	}
	return false
}
