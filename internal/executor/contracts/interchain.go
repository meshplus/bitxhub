package contracts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	service_mgr "github.com/meshplus/bitxhub-core/service-mgr"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
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
	INTERCHAINSERVICE_PREFIX   = "service"
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

func (cs *ChainService) getFromChainID() string {
	if cs.IsLocal {
		return cs.ChainId
	}
	return cs.BxhId
}

func (x *InterchainManager) Register(chainServiceID string) *boltvm.Response {
	bxhID, err := x.getBitXHubID()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	fullServiceID := fmt.Sprintf("%s:%s", bxhID, chainServiceID)
	interchain, ok := x.getInterchain(fullServiceID)
	if !ok {
		x.setInterchain(fullServiceID, interchain)
	}
	body, err := interchain.Marshal()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(body)
}

func (x *InterchainManager) DeleteInterchain(id string) *boltvm.Response {
	x.Delete(serviceKey(id))
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
	interchain := &pb.Interchain{ID: id}
	ok, data := x.Get(serviceKey(id))

	if ok {
		if err := interchain.Unmarshal(data); err != nil {
			panic(err)
		}
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

	return interchain, ok
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) setInterchain(id string, interchain *pb.Interchain) {
	data, err := interchain.Marshal()
	if err != nil {
		panic(err)
	}

	x.Set(serviceKey(id), data)
}

// Interchain returns information of the interchain count, Receipt count and SourceReceipt count
func (x *InterchainManager) Interchain(id string) *boltvm.Response {
	ok, data := x.Get(serviceKey(id))
	if !ok {
		return boltvm.Error(fmt.Errorf("this service does not exist").Error())
	}
	return boltvm.Success(data)
}

// GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) GetInterchain(id string) *boltvm.Response {
	ok, data := x.Get(serviceKey(id))
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

	interchain, _ := x.getInterchain(srcChainService.getFullServiceId())

	if pb.IBTP_INTERCHAIN == ibtp.Type {
		// if src chain service is from appchain registered in current bitxhub, check service index
		fromChain := srcChainService.getFromChainID()

		srcAppchain, err := x.getAppchainInfo(fromChain)
		if err != nil {
			return nil, fmt.Errorf("%s: source appchain %s is not registered", CurAppchainNotAvailable, fromChain)
		}

		if !srcAppchain.IsAvailable() {
			return nil, fmt.Errorf("%s: source appchain status is %s, can not handle IBTP", CurAppchainNotAvailable, string(srcAppchain.Status))
		}

		// if dst chain service is from appchain registered in current bitxhub, get service info and check src service's permission
		if dstChainService.IsLocal {
			dstService, err := x.getServiceByID(dstChainService.getChainServiceId())
			if err != nil {
				return nil, fmt.Errorf(fmt.Sprintf("%s: cannot get service by %s", TargetServiceNotAvailable, dstChainService.getChainServiceId()))
			}

			if !dstService.CheckPermission(srcChainService.getFullServiceId()) {
				return nil, fmt.Errorf(fmt.Sprintf("%s: the service %s is not permitted to visit %s", TargetServiceNotAvailable, srcChainService.getFullServiceId(), dstChainService.getFullServiceId()))
			}

			if dstService.Ordered {
				if err := checkIndex(interchain.InterchainCounter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
					return nil, err
				}
			}
		} else {
			if err := checkIndex(interchain.InterchainCounter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
				return nil, err
			}
		}
	} else if ibtp.Type == pb.IBTP_RECEIPT_SUCCESS || ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
		if err := checkIndex(interchain.ReceiptCounter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
			return nil, err
		}
	}

	return interchain, nil
}

func (x *InterchainManager) checkAppchain(id string) (*pb.Interchain, *appchainMgr.Appchain, error) {
	interchain, ok := x.getInterchain(id)
	if !ok {
		return nil, nil, fmt.Errorf("%s: this appchain does not exist", AppchainNotAvailable)
	}

	app := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(id))
	if !res.Ok {
		return nil, nil, fmt.Errorf("%s: get appchain info error: %s", AppchainNotAvailable, string(res.Result))
	}

	if err := json.Unmarshal(res.Result, app); err != nil {
		return nil, nil, fmt.Errorf("unmarshal error: " + err.Error())
	}

	if !app.IsAvailable() {
		//if !isAppchainAvailable(app.Status) {
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
			if dstAppchain.IsAvailable() {
				availableFlag = true
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
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(chainMethod))
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
				res := x.CrossInvoke(constant.InterBrokerContractAddr.Address().String(), "InvokeInterchain", &pb.Arg{Type: pb.Arg_Bytes, Value: data})
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
				x.CrossInvoke(constant.InterBrokerContractAddr.Address().String(), "InvokeInterchain", &pb.Arg{Type: pb.Arg_Bytes, Value: data})
			}
		} else {
			m[DEFAULT_UNION_PIER_ID] = x.GetTxIndex()
		}

		ic, _ := x.getInterchain(to)
		ic.SourceReceiptCounter[from] = ibtp.Index
		x.setInterchain(to, ic)
		x.SetObject(x.indexReceiptMapKey(getIBTPID(from, to, ibtp.Index)), x.GetTxHash())

		result := true
		if ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
			result = false
		}
		x.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "RecordInvokeService",
			pb.String(to),
			pb.String(from),
			pb.Bool(result))
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

	return x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "BeginMultiTXs", args...)
}

func (x *InterchainManager) beginTransaction(ibtp *pb.IBTP) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	bxhID0, _, _ := ibtp.ParseFrom()
	bxhID1, _, _ := ibtp.ParseTo()
	timeoutHeight := uint64(ibtp.TimeoutHeight)
	// TODO: disable transaction management for inter-bitxhub transaction temporarily
	if bxhID0 != bxhID1 {
		timeoutHeight = 0
	}
	return x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "Begin", pb.String(txId), pb.Uint64(timeoutHeight))
}

func (x *InterchainManager) reportTransaction(ibtp *pb.IBTP, interchain *pb.Interchain) *boltvm.Response {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	result := int32(0)
	if ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
		result = 1
	}
	ret := x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "Report", pb.String(txId), pb.Int32(result))
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
	if ok := x.Has(serviceKey(ibtp.To)); !ok {
		return boltvm.Error(fmt.Sprintf("target appchain does not exist: %s", ibtp.To))
	}

	interchain, ok := x.getInterchain(ibtp.From)
	if !ok {
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
	if nil == app.TrustRoot {
		return false, fmt.Errorf("empty validators in relay chain:%s", app.ID)
	}
	var validators BxhValidators
	if err := json.Unmarshal(app.TrustRoot, &validators); err != nil {
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
		ok, _ := asym.VerifyWithType(sign, hash[:], *addr)
		if ok {
			counter++
		}
		if counter > threshold {
			return true, nil
		}
	}
	return false, fmt.Errorf("multi signs verify fail, counter: %d", counter)
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

func (x *InterchainManager) GetBitXHubID() *boltvm.Response {
	id, err := x.getBitXHubID()
	if err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success([]byte(id))
	}
}

func (x *InterchainManager) getBitXHubID() (string, error) {
	ok, val := x.Get(BitXHubID)
	if !ok {
		return "", fmt.Errorf("cannot get bitxhub ID")
	}

	return string(val), nil
}

func (x *InterchainManager) getServiceByID(id string) (*service_mgr.Service, error) {
	service := &service_mgr.Service{}

	res := x.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", pb.String(id))
	if !res.Ok {
		return nil, fmt.Errorf("%s: get service info error: %s", ServiceNotAvailable, string(res.Result))
	}

	if err := json.Unmarshal(res.Result, service); err != nil {
		return nil, fmt.Errorf("unmarshal service of ID %s: %w", id, err)
	}

	return service, nil
}

func checkIndex(exp, cur uint64) error {
	if cur < exp {
		return fmt.Errorf("%s: required %d, but %d", ibtpIndexExist, exp, cur)
	}
	if cur > exp {
		return fmt.Errorf("%s: required %d, but %d", ibtpIndexWrong, exp, cur)
	}

	return nil
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

func serviceKey(id string) string {
	return fmt.Sprintf("%s-%s", INTERCHAINSERVICE_PREFIX, id)
}
