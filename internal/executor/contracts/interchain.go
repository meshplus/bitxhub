package contracts

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meshplus/bitxhub-core/governance"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

const (
	BitXHubID                  = "bitxhub-id"
	CurAppchainNotAvailable    = "current appchain not available"
	TargetAppchainNotAvailable = "target appchain not available"
	AppchainNotAvailable       = "the appchain is not available"
	ServiceNotAvailable        = "the service is not available"
	InvalidIBTP                = "invalid ibtp"
	InvalidTargetService       = "invalid target service"
	TargetServiceNotAvailable  = "target service not available"
	internalError              = "internal server error"
	ibtpIndexExist             = "index already exists"
	ibtpIndexWrong             = "wrong index"
	DEFAULT_UNION_PIER_ID      = "default_union_pier_id"
)

type InterchainMeta struct {
	TargetChain string `json:"target_chain"`
	TxHash      string `json:"tx_hash"`
	Timestamp   int64  `json:"timestamp"`
}

type InterchainInfo struct {
	ChainId            string            `json:"chain_id"`
	InterchainCounter  uint64            `json:"interchain_counter"`
	ReceiptCounter     uint64            `json:"receipt_counter"`
	SendInterchains    []*InterchainMeta `json:"send_interchains"`
	ReceiptInterchains []*InterchainMeta `json:"receipt_interchains"`
}

type InterchainManager struct {
	boltvm.Stub
}

type BxhValidators struct {
	Addresses []string `json:"addresses"`
}

type ChainService struct {
	BxhId     string `json:"bxh_id"`
	ChainId   string `json:"chain_id"`
	ServiceId string `json:"service_id"`
	IsLocal   bool   `json:"is_local"`
}

func (cs *ChainService) getFullServiceId() string {
	return fmt.Sprintf("%s:%s:%s", cs.BxhId, cs.ChainId, cs.ServiceId)
}

func (cs *ChainService) getChainServiceId() string {
	return fmt.Sprintf("%s:%s", cs.ChainId, cs.ServiceId)
}

func (x *InterchainManager) Register(chainServiceID string) *boltvm.Response {
	bxhID, err := x.getBitXHubID()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	fullServiceID := fmt.Sprintf("%s:%s", bxhID, chainServiceID)
	interchain, ok := x.getInterchain(fullServiceID)
	if !ok {
		interchain = &pb.Interchain{
			ID:                      fullServiceID,
			InterchainCounter:       make(map[string]uint64),
			ReceiptCounter:          make(map[string]uint64),
			SourceInterchainCounter: make(map[string]uint64),
			SourceReceiptCounter:    make(map[string]uint64),
		}
		x.setInterchain(fullServiceID, interchain)
	}
	body, err := interchain.Marshal()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(body)
}

func (x *InterchainManager) DeleteInterchain(id string) *boltvm.Response {
	x.Delete(ServiceKey(id))
	return boltvm.Success(nil)
}

func (x *InterchainManager) GetInterchainInfo(chainId string) *boltvm.Response {
	interchain, ok := x.getInterchain(chainId)
	info := &InterchainInfo{
		ChainId:            chainId,
		SendInterchains:    []*InterchainMeta{},
		ReceiptInterchains: []*InterchainMeta{},
	}
	if !ok {
		interchain = &pb.Interchain{
			ID:                   chainId,
			InterchainCounter:    make(map[string]uint64),
			ReceiptCounter:       make(map[string]uint64),
			SourceReceiptCounter: make(map[string]uint64),
		}
	}
	for _, counter := range interchain.InterchainCounter {
		info.InterchainCounter += counter
	}

	for _, counter := range interchain.ReceiptCounter {
		info.ReceiptCounter += counter
	}
	x.GetObject(x.indexSendInterchainMeta(chainId), &info.SendInterchains)
	x.GetObject(x.indexReceiptInterchainMeta(chainId), &info.ReceiptInterchains)
	data, err := json.Marshal(&info)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) getInterchain(id string) (*pb.Interchain, bool) {
	interchain := &pb.Interchain{}
	ok, data := x.Get(ServiceKey(id))
	if !ok {
		return nil, false
	}

	if err := interchain.Unmarshal(data); err != nil {
		panic(err)
	}

	if interchain.InterchainCounter == nil {
		interchain.InterchainCounter = make(map[string]uint64)
	}

	if interchain.ReceiptCounter == nil {
		interchain.ReceiptCounter = make(map[string]uint64)
	}

	if interchain.SourceInterchainCounter == nil {
		interchain.SourceInterchainCounter = make(map[string]uint64)
	}

	if interchain.SourceReceiptCounter == nil {
		interchain.SourceReceiptCounter = make(map[string]uint64)
	}

	return interchain, true
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) setInterchain(id string, interchain *pb.Interchain) {
	data, err := interchain.Marshal()
	if err != nil {
		panic(err)
	}

	x.Set(ServiceKey(id), data)
}

// Interchain returns information of the interchain count, Receipt count and SourceReceipt count
func (x *InterchainManager) Interchain(id string) *boltvm.Response {
	ok, data := x.Get(ServiceKey(id))
	if !ok {
		return boltvm.Error(fmt.Errorf("this service does not exist").Error())
	}
	return boltvm.Success(data)
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) GetInterchain(id string) *boltvm.Response {
	ok, data := x.Get(ServiceKey(id))
	if !ok {
		return boltvm.Error(fmt.Errorf("this service does not exist: %s", id).Error())
	}
	return boltvm.Success(data)
}

func (x *InterchainManager) HandleIBTPData(input []byte) *boltvm.Response {
	ibtp := &pb.IBTP{}
	err := ibtp.Unmarshal(input)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return x.HandleIBTP(ibtp)
}

func (x *InterchainManager) HandleIBTP(ibtp *pb.IBTP) *boltvm.Response {
	// Pier should retry if checkIBTP failed
	interchain, err := x.checkIBTP(ibtp)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	// ProcessIBTP should always executed even if checkTargetAppchainAvailability failed
	appchainErr := x.checkTargetAppchainAvailability(ibtp)

	res := boltvm.Success(nil)

	if pb.IBTP_INTERCHAIN == ibtp.Type {
		res = x.beginTransaction(ibtp)
	} else if pb.IBTP_RECEIPT_SUCCESS == ibtp.Type || pb.IBTP_RECEIPT_FAILURE == ibtp.Type {
		res = x.reportTransaction(ibtp, interchain)
	}

	if !res.Ok {
		return res
	}

	res.Result = x.ProcessIBTP(ibtp, interchain)

	if appchainErr != nil {
		return boltvm.Error(appchainErr.Error())
	}

	return res
}

func (x *InterchainManager) checkIBTP(ibtp *pb.IBTP) (*pb.Interchain, error) {
	srcChainService, err := x.parseChainService(ibtp.From)
	if err != nil {
		return nil, fmt.Errorf("%s: parse source chain service id %w", InvalidIBTP, err)
	}

	dstChainService, err := x.parseChainService(ibtp.To)
	if err != nil {
		return nil, fmt.Errorf("%s: parsed dest chain service id %w", InvalidTargetService, err)
	}

	interchain, ok := x.getInterchain(srcChainService.getFullServiceId())
	if !ok {
		interchain = &pb.Interchain{
			ID:                      srcChainService.getFullServiceId(),
			InterchainCounter:       make(map[string]uint64),
			ReceiptCounter:          make(map[string]uint64),
			SourceInterchainCounter: make(map[string]uint64),
			SourceReceiptCounter:    make(map[string]uint64),
		}
	}

	if pb.IBTP_INTERCHAIN == ibtp.Type {
		// if src chain service is from appchain registered in current bitxhub, check service index
		if srcChainService.IsLocal {
			srcAppchain, err := x.getAppchainInfo(srcChainService.ChainId)
			if err != nil {
				return nil, fmt.Errorf("%s: source appchain %s is not registered", CurAppchainNotAvailable, ibtp.From)
			}

			if !isAppchainAvailable(srcAppchain.Status) {
				return nil, fmt.Errorf("%s: source appchain status is %s, can not handle IBTP", CurAppchainNotAvailable, string(srcAppchain.Status))
			}

			if srcAppchain.ChainType != appchainMgr.RelaychainType {
				if err := x.checkPubKeyAndCaller(srcAppchain.PublicKey); err != nil {
					return nil, fmt.Errorf("%s: caller is not bind to ibtp from: %w", InvalidIBTP, err)
				}
			}

			if !dstChainService.IsLocal {
				idx := interchain.InterchainCounter[dstChainService.getFullServiceId()]
				if ibtp.Index <= idx {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexExist, idx+1, ibtp.Index))
				}
				if ibtp.Index > idx+1 {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexWrong, idx+1, ibtp.Index))
				}
			}
		}
		// if dst chain service is from appchain registered in current bitxhub, get service info and check src service's permission
		if dstChainService.IsLocal {
			dstService, err := x.getServiceByID(dstChainService.getChainServiceId())
			if err != nil {
				return nil, fmt.Errorf(fmt.Sprintf("%s: cannot get service by %s", TargetServiceNotAvailable, dstChainService.getChainServiceId()))
			}

			if !dstService.checkPermission(srcChainService.getFullServiceId()) {
				return nil, fmt.Errorf(fmt.Sprintf("%s: the service %s is not permitted to visit %s", TargetServiceNotAvailable, srcChainService.getFullServiceId(), dstChainService.getFullServiceId()))
			}

			if dstService.Ordered {
				idx := interchain.InterchainCounter[dstChainService.getFullServiceId()]
				if ibtp.Index <= idx {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexExist, idx+1, ibtp.Index))
				}
				if ibtp.Index > idx+1 {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexWrong, idx+1, ibtp.Index))
				}
			}
		}
	} else if ibtp.Type == pb.IBTP_RECEIPT_SUCCESS || ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
		if srcChainService.IsLocal {
			srcService, err := x.getServiceByID(srcChainService.getChainServiceId())
			if err != nil {
				return nil, err
			}
			if srcService.Ordered {
				idx := interchain.ReceiptCounter[dstChainService.getFullServiceId()]
				if ibtp.Index <= idx {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexExist, idx+1, ibtp.Index))
				}

				if ibtp.Index > idx+1 {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexWrong, idx+1, ibtp.Index))
				}
			}
		}

		if dstChainService.IsLocal {
			if !srcChainService.IsLocal {
				idx := interchain.ReceiptCounter[dstChainService.getFullServiceId()]
				if ibtp.Index <= idx {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexExist, idx+1, ibtp.Index))
				}

				if ibtp.Index > idx+1 {
					return nil, fmt.Errorf(fmt.Sprintf("%s: required %d, but %d", ibtpIndexWrong, idx+1, ibtp.Index))
				}
			}
		}
	}

	return interchain, nil
}

func (x *InterchainManager) checkPubKeyAndCaller(pub string) error {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		return fmt.Errorf("decode public key bytes: %w", err)
	}
	pubKey, err := ecdsa.UnmarshalPublicKey(pubKeyBytes, crypto.Secp256k1)
	if err != nil {
		return fmt.Errorf("decrypt registerd public key error: %w", err)
	}
	addr, err := pubKey.Address()
	if err != nil {
		return fmt.Errorf("decrypt registerd public key error: %w", err)
	}
	if addr.String() != x.Caller() {
		return fmt.Errorf("chain pub key derived address != caller")
	}
	return nil
}

func (x *InterchainManager) checkAppchain(id string) (*pb.Interchain, *appchainMgr.Appchain, error) {
	interchain, ok := x.getInterchain(id)
	if !ok {
		return nil, nil, fmt.Errorf("%s: this appchain does not exist", AppchainNotAvailable)
	}

	app := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(id))
	if !res.Ok {
		return nil, nil, fmt.Errorf("%s: get appchain info error: %s", AppchainNotAvailable, string(res.Result))
	}

	if err := json.Unmarshal(res.Result, app); err != nil {
		return nil, nil, fmt.Errorf("unmarshal error: " + err.Error())
	}

	if !isAppchainAvailable(app.Status) {
		return nil, nil, fmt.Errorf("%s: the appchain status is %s, can not handle IBTP", AppchainNotAvailable, string(app.Status))
	}

	return interchain, app, nil
}

func (x *InterchainManager) checkTargetAppchainAvailability(ibtp *pb.IBTP) error {
	if pb.IBTP_INTERCHAIN == ibtp.Type {
		dstChainService, err := x.parseChainService(ibtp.To)
		if err != nil {
			return fmt.Errorf("%s: parsed dest chain service id %w", InvalidTargetService, err)
		}

		if dstChainService.IsLocal {
			if dstChainService.ChainId == dstChainService.BxhId {
				return nil
			}
			dstAppchain, err := x.getAppchainInfo(dstChainService.ChainId)
			if err != nil {
				return fmt.Errorf("%s: dest appchain id %s is not registered", TargetAppchainNotAvailable, dstChainService.ChainId)
			}
			availableFlag := false
			for _, s := range appchainMgr.AppchainAvailableState {
				if dstAppchain.Status == s {
					availableFlag = true
					break
				}
			}
			if !availableFlag {
				return fmt.Errorf("%s: dest appchain status is %s, can not handle IBTP", TargetAppchainNotAvailable, string(dstAppchain.Status))
			}
		}
	}

	return nil
}

// isRelayIBTP returns whether ibtp.from is relaychain type
func (x *InterchainManager) getAppchainInfo(chainMethod string) (*appchainMgr.Appchain, error) {
	srcChain := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chainMethod))
	if err := json.Unmarshal(res.Result, srcChain); err != nil {
		return nil, fmt.Errorf("%s: unmarshal appchain info error: %w", internalError, err)
	}
	return srcChain, nil
}

func (x *InterchainManager) ProcessIBTP(ibtp *pb.IBTP, interchain *pb.Interchain) []byte {
	m := make(map[string]uint64)
	srcChainService, _ := x.parseChainService(ibtp.From)
	dstChainService, _ := x.parseChainService(ibtp.To)

	from := srcChainService.getFullServiceId()
	to := dstChainService.getFullServiceId()

	if pb.IBTP_INTERCHAIN == ibtp.Type || pb.IBTP_ROLLBACK == ibtp.Type {
		if interchain.InterchainCounter == nil {
			x.Logger().Info("interchain counter is nil, make one")
			interchain.InterchainCounter = make(map[string]uint64)
		}
		interchain.InterchainCounter[to]++
		x.setInterchain(from, interchain)
		x.AddObject(x.indexMapKey(getIBTPID(from, to, ibtp.Index)), x.GetTxHash())
		if dstChainService.IsLocal {
			m[dstChainService.ChainId] = x.GetTxIndex()
			if dstChainService.ChainId == dstChainService.BxhId {
				data, _ := ibtp.Marshal()
				res := x.CrossInvoke(constant.InterBrokerContractAddr.String(), "InvokeInterchain", &pb.Arg{Type: pb.Arg_Bytes, Value: data})
				if res.Ok {
					return res.Result
				}
			}
		} else {
			m[DEFAULT_UNION_PIER_ID] = x.GetTxIndex()
		}

		ic, _ := x.getInterchain(to)
		ic.SourceInterchainCounter[from] = ibtp.Index
		x.setInterchain(to, ic)

		meta := &InterchainMeta{
			TargetChain: ibtp.To,
			TxHash:      x.GetTxHash().String(),
			Timestamp:   x.GetTxTimeStamp(),
		}
		x.setInterchainMeta(x.indexSendInterchainMeta(ibtp.From), meta)

		meta.TargetChain = ibtp.From
		x.setInterchainMeta(x.indexReceiptInterchainMeta(ibtp.To), meta)
	} else {
		interchain.ReceiptCounter[to] = ibtp.Index
		x.setInterchain(from, interchain)
		if srcChainService.IsLocal {
			m[srcChainService.ChainId] = x.GetTxIndex()
			if srcChainService.ChainId == srcChainService.BxhId {
				data, _ := ibtp.Marshal()
				x.CrossInvoke(constant.InterBrokerContractAddr.String(), "InvokeInterchain", &pb.Arg{Type: pb.Arg_Bytes, Value: data})
			}
		} else {
			m[DEFAULT_UNION_PIER_ID] = x.GetTxIndex()
		}

		ic, _ := x.getInterchain(to)
		ic.SourceReceiptCounter[from] = ibtp.Index
		x.setInterchain(to, ic)
		x.SetObject(x.indexReceiptMapKey(getIBTPID(from, to, ibtp.Index)), x.GetTxHash())
	}

	if pb.IBTP_RECEIPT_ROLLBACK != ibtp.Type {
		x.PostInterchainEvent(m)
	}

	return nil
}

func (x *InterchainManager) beginMultiTargetsTransaction(srcChainMethod string, ibtps *pb.IBTPs) *boltvm.Response {
	args := make([]*pb.Arg, 0)
	globalId := fmt.Sprintf("%s-%s", srcChainMethod, x.GetTxHash())
	args = append(args, pb.String(globalId))

	for _, ibtp := range ibtps.Ibtps {
		if ibtp.Type != pb.IBTP_INTERCHAIN {
			return boltvm.Error("ibtp type != IBTP_INTERCHAIN")
		}

		childTxId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
		args = append(args, pb.String(childTxId))
	}

	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "BeginMultiTXs", args...)
}

func (x *InterchainManager) beginTransaction(ibtp *pb.IBTP) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	return x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "Begin", pb.String(txId), pb.Uint64(uint64(ibtp.TimeoutHeight)))
}

func (x *InterchainManager) reportTransaction(ibtp *pb.IBTP, interchain *pb.Interchain) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	result := int32(0)
	if ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
		result = 1
	}
	ret := x.CrossInvoke(constant.TransactionMgrContractAddr.String(), "Report", pb.String(txId), pb.Int32(result))
	if strings.Contains(string(ret.Result), fmt.Sprintf("transaction with Id %s has been rollback", txId)) {
		interchain.ReceiptCounter[ibtp.To] = ibtp.Index
		x.setInterchain(ibtp.From, interchain)

		ic, _ := x.getInterchain(ibtp.To)
		ic.SourceReceiptCounter[ibtp.From] = ibtp.Index
		x.setInterchain(ibtp.To, ic)
		x.SetObject(x.indexReceiptMapKey(ibtp.ID()), x.GetTxHash())
	}

	return ret
}

func (x *InterchainManager) GetIBTPByID(id string, isReq bool) *boltvm.Response {
	arr := strings.Split(id, "-")
	if len(arr) != 3 {
		return boltvm.Error("wrong ibtp id")
	}

	var (
		hash types.Hash
		key  string
	)

	if isReq {
		key = x.indexMapKey(id)
	} else {
		key = x.indexReceiptMapKey(id)
	}
	exist := x.GetObject(key, &hash)
	if !exist {
		return boltvm.Error("this ibtp id does not exist")
	}

	return boltvm.Success(hash.Bytes())
}

func (x *InterchainManager) handleUnionIBTP(ibtp *pb.IBTP) *boltvm.Response {
	srcRelayChainID := strings.Split(ibtp.From, "-")[0]

	_, app, err := x.checkAppchain(srcRelayChainID)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	if ibtp.To == "" {
		return boltvm.Error("empty destination chain id")
	}
	if ok := x.Has(ServiceKey(ibtp.To)); !ok {
		return boltvm.Error(fmt.Sprintf("target appchain does not exist: %s", ibtp.To))
	}

	interchain, ok := x.getInterchain(ibtp.From)
	if !ok {
		interchain = &pb.Interchain{
			ID: ibtp.From,
		}
		x.setInterchain(ibtp.From, interchain)
	}

	if err := x.checkUnionIBTP(app, ibtp, interchain); err != nil {
		return boltvm.Error(err.Error())
	}

	x.ProcessIBTP(ibtp, interchain)
	return boltvm.Success(nil)
}

func (x *InterchainManager) checkUnionIBTP(app *appchainMgr.Appchain, ibtp *pb.IBTP, interchain *pb.Interchain) error {
	if pb.IBTP_INTERCHAIN == ibtp.Type {

		idx := interchain.InterchainCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			return fmt.Errorf(fmt.Sprintf("wrong index, required %d, but %d", idx+1, ibtp.Index))
		}
	} else {
		idx := interchain.ReceiptCounter[ibtp.To]
		if idx+1 != ibtp.Index {
			if interchain.SourceReceiptCounter[ibtp.To]+1 != ibtp.Index {
				return fmt.Errorf("wrong receipt index, required %d, but %d", idx+1, ibtp.Index)
			}
		}
	}

	_, err := verifyMultiSign(app, ibtp, ibtp.Proof)
	return err
}

// verifyMultiSign .
func verifyMultiSign(app *appchainMgr.Appchain, ibtp *pb.IBTP, proof []byte) (bool, error) {
	if "" == app.Validators {
		return false, fmt.Errorf("empty validators in relay chain:%s", app.ID)
	}
	var validators BxhValidators
	if err := json.Unmarshal([]byte(app.Validators), &validators); err != nil {
		return false, err
	}

	m := make(map[string]struct{}, 0)
	for _, validator := range validators.Addresses {
		m[validator] = struct{}{}
	}

	var signs pb.SignResponse
	if err := signs.Unmarshal(ibtp.Proof); err != nil {
		return false, err
	}

	threshold := (len(validators.Addresses) - 1) / 3 // TODO be dynamic
	counter := 0

	ibtpHash := ibtp.Hash()
	hash := sha256.Sum256([]byte(ibtpHash.String()))
	for v, sign := range signs.Sign {
		if _, ok := m[v]; !ok {
			return false, fmt.Errorf("wrong validator: %s", v)
		}
		delete(m, v)
		addr := types.NewAddressByStr(v)
		ok, _ := asym.Verify(crypto.Secp256k1, sign, hash[:], *addr)
		if ok {
			counter++
		}
		if counter > threshold {
			return true, nil
		}
	}
	return false, fmt.Errorf("multi signs verify fail, counter: %d", counter)
}

func ServiceKey(id string) string {
	return ServicePreKey + id
}

func (x *InterchainManager) indexMapKey(id string) string {
	return fmt.Sprintf("index-tx-%s", id)
}

func (x *InterchainManager) indexReceiptMapKey(id string) string {
	return fmt.Sprintf("index-receipt-tx-%s", id)
}

func (x *InterchainManager) ParseChainService(id string) *boltvm.Response {
	chainService, err := x.parseChainService(id)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	data, _ := json.Marshal(chainService)
	return boltvm.Success(data)
}

func (x *InterchainManager) parseChainService(id string) (*ChainService, error) {
	splits := strings.Split(id, ":")

	size := len(splits)

	if size != 2 && size != 3 {
		return nil, fmt.Errorf("invalid chain service id %s", id)
	}

	bxhId, err := x.getBitXHubID()
	if err != nil {
		return nil, err
	}

	if len(splits) == 2 {
		return &ChainService{
			BxhId:     bxhId,
			ChainId:   splits[0],
			ServiceId: splits[1],
			IsLocal:   true,
		}, nil
	}

	return &ChainService{
		BxhId:     splits[0],
		ChainId:   splits[1],
		ServiceId: splits[2],
		IsLocal:   splits[0] == bxhId,
	}, nil
}

func (x *InterchainManager) getBitXHubID() (string, error) {
	ok, val := x.Get(BitXHubID)
	if !ok {
		return "", fmt.Errorf("cannot get bitxhub ID")
	}

	return string(val), nil
}

func (x *InterchainManager) getServiceByID(id string) (*Service, error) {
	service := &Service{}

	res := x.CrossInvoke(constant.ServiceMgrContractAddr.String(), "GetServiceInfo", pb.String(id))
	if !res.Ok {
		return nil, fmt.Errorf("%s: get service info error: %s", ServiceNotAvailable, string(res.Result))
	}

	if err := json.Unmarshal(res.Result, service); err != nil {
		return nil, fmt.Errorf("unmarshal service of ID %s: %w", id, err)
	}

	return service, nil
}

func getIBTPID(from, to string, index uint64) string {
	return fmt.Sprintf("%s-%s-%d", from, to, index)
}

func (x *InterchainManager) setInterchainMeta(indexKey string, meta *InterchainMeta) {
	var metas []*InterchainMeta
	x.GetObject(indexKey, &metas)
	if len(metas) >= 5 {
		metas = metas[1:]
	}
	metas = append(metas, meta)
	x.SetObject(indexKey, &metas)
}

func (x *InterchainManager) indexSendInterchainMeta(id string) string {
	return fmt.Sprintf("index-send-interchain-%s", id)
}

func (x *InterchainManager) indexReceiptInterchainMeta(id string) string {
	return fmt.Sprintf("index-receipt-interchain-%s", id)
}

func isAppchainAvailable(state governance.GovernanceStatus) bool {
	for _, s := range appchainMgr.AppchainAvailableState {
		if state == s {
			return true
		}
	}

	return false
}
