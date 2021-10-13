package contracts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	service_mgr "github.com/meshplus/bitxhub-core/service-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

const (
	BitXHubID                  = "bitxhub-id"
	CurAppchainNotAvailable    = "current appchain not available"
	TargetAppchainNotAvailable = "target appchain not available"
	SrcBitXHubNotAvailable     = "source bitxhub not available"
	TargetBitXHubNotAvailable  = "target bitxhub not available"
	CurServiceNotAvailable     = "current service not available"
	TargetServiceNotAvailable  = "target service not available"
	InvalidIBTP                = "invalid ibtp"
	InvalidTargetService       = "invalid target service"
	internalError              = "internal server error"
	ibtpIndexExist             = "index already exists"
	ibtpIndexWrong             = "wrong index"
	DEFAULT_UNION_PIER_ID      = "default_union_pier_id"
	INTERCHAINSERVICE_PREFIX   = "service"
	MULTITX_PREFIX             = "multitx"
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

// todo: the parameter name should be fullServiceID
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

// getInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
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

// setInterchain set information of the interchain count, Receipt count and SourceReceipt count by id
func (x *InterchainManager) setInterchain(id string, interchain *pb.Interchain) {
	data, err := interchain.Marshal()
	if err != nil {
		panic(err)
	}

	x.Set(serviceKey(id), data)
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
	interchain, targetErr, err := x.checkIBTP(ibtp)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	var change *StatusChange
	if pb.IBTP_REQUEST == ibtp.Category() {
		change, err = x.beginTransaction(ibtp, targetErr != nil)
	} else if pb.IBTP_RESPONSE == ibtp.Category() {
		change, err = x.reportTransaction(ibtp)
	}
	if err != nil {
		return boltvm.Error(err.Error())
	}

	x.notifySrcDst(ibtp, change)

	ret := x.ProcessIBTP(ibtp, interchain)

	return boltvm.Success(ret)
}

func (x *InterchainManager) checkIBTP(ibtp *pb.IBTP) (*pb.Interchain, error, error) {
	var targetError error

	srcChainService, err := x.parseChainService(ibtp.From)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: parse source chain service id %w", InvalidIBTP, err)
	}

	dstChainService, err := x.parseChainService(ibtp.To)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: parsed dest chain service id %w", InvalidTargetService, err)
	}

	interchain, _ := x.getInterchain(srcChainService.getFullServiceId())

	if pb.IBTP_REQUEST == ibtp.Category() {
		// if src chain service is from appchain registered in current bitxhub, check service index
		if srcChainService.IsLocal {
			if err := x.checkSourceAvailability(srcChainService); err != nil {
				return nil, nil, err
			}

			targetError = x.checkTargetAvailability(srcChainService, dstChainService, ibtp.Type)

			// if dst chain service is from appchain registered in current bitxhub, get service info and check index
			if err := x.checkServiceIndex(ibtp, interchain.InterchainCounter, dstChainService); err != nil {
				return nil, nil, err
			}
		} else {
			if !dstChainService.IsLocal {
				return nil, nil, fmt.Errorf("%s: neither source service nor dest service of IBTP %s is in current bitxhub", InvalidIBTP, ibtp.ID())
			}

			if err := x.checkBitXHubAvailability(srcChainService.BxhId); err != nil {
				return nil, nil, fmt.Errorf("%s: source BitXHub %s is not registered", SrcBitXHubNotAvailable, srcChainService.BxhId)
			}

			if err := x.checkServiceIndex(ibtp, interchain.InterchainCounter, dstChainService); err != nil {
				return nil, nil, err
			}
		}
	} else if ibtp.Category() == pb.IBTP_RESPONSE {
		if err := x.checkServiceIndex(ibtp, interchain.ReceiptCounter, dstChainService); err != nil {
			return nil, nil, err
		}
	} else {
		return nil, nil, fmt.Errorf("%s: IBTP type %v is not expected", InvalidIBTP, ibtp.Type)
	}

	return interchain, targetError, nil
}

func (x *InterchainManager) checkSourceAvailability(srcChainService *ChainService) error {
	if err := x.checkAppchainAvailability(srcChainService.ChainId); err != nil {
		return fmt.Errorf("%s: source appchain id is %s", CurAppchainNotAvailable, srcChainService.ChainId)
	}

	if err := x.checkServiceAvailability(srcChainService.getChainServiceId()); err != nil {
		return fmt.Errorf(fmt.Sprintf("%s: cannot get service by %s", CurServiceNotAvailable, srcChainService.getChainServiceId()))
	}

	return nil
}

func (x *InterchainManager) checkTargetAvailability(srcChainService, dstChainService *ChainService, typ pb.IBTP_Type) error {
	if pb.IBTP_INTERCHAIN == typ {
		if dstChainService.IsLocal {
			if dstChainService.ChainId == dstChainService.BxhId {
				return nil
			}

			if err := x.checkAppchainAvailability(dstChainService.ChainId); err != nil {
				return fmt.Errorf("%s: target appchain id is %s", TargetAppchainNotAvailable, dstChainService.ChainId)
			}

			dstService, err := x.getServiceByID(dstChainService.getChainServiceId())
			if err != nil {
				return fmt.Errorf("%s: cannot get service by %s", TargetServiceNotAvailable, dstChainService.getChainServiceId())
			}

			if !dstService.IsAvailable() {
				return fmt.Errorf("%s: current status of service %s is %v", TargetServiceNotAvailable, dstChainService.getChainServiceId(), dstService.Status)
			}

			if !dstService.CheckPermission(srcChainService.getFullServiceId()) {
				return fmt.Errorf("%s: the service %s is not permitted to visit %s", TargetServiceNotAvailable, srcChainService.getFullServiceId(), dstChainService.getFullServiceId())
			}
		} else {
			if err := x.checkBitXHubAvailability(dstChainService.BxhId); err != nil {
				return fmt.Errorf("%s, error is %w", TargetBitXHubNotAvailable, err)
			}
		}
	}

	return nil
}

// getAppchainInfo returns the appchain info by chain ID
func (x *InterchainManager) getAppchainInfo(chainID string) (*appchainMgr.Appchain, error) {
	appchain := &appchainMgr.Appchain{}
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(chainID))
	if !res.Ok {
		return nil, fmt.Errorf("chain %s is not registered", chainID)
	}
	if err := json.Unmarshal(res.Result, appchain); err != nil {
		return nil, fmt.Errorf("%s: unmarshal appchain info error: %w", internalError, err)
	}
	return appchain, nil
}

func (x *InterchainManager) ProcessIBTP(ibtp *pb.IBTP, interchain *pb.Interchain) []byte {
	srcChainService, _ := x.parseChainService(ibtp.From)
	dstChainService, _ := x.parseChainService(ibtp.To)

	if pb.IBTP_REQUEST == ibtp.Category() {
		if interchain.InterchainCounter == nil {
			x.Logger().Info("interchain counter is nil, make one")
			interchain.InterchainCounter = make(map[string]uint64)
		}
		interchain.InterchainCounter[ibtp.To]++
		x.setInterchain(ibtp.From, interchain)
		x.AddObject(IndexMapKey(getIBTPID(ibtp.From, ibtp.To, ibtp.Index)), x.GetTxHash())
		if dstChainService.ChainId == dstChainService.BxhId {
			data, _ := ibtp.Marshal()
			res := x.CrossInvoke(constant.InterBrokerContractAddr.Address().String(), "InvokeInterchain", pb.Bytes(data))
			if res.Ok {
				return res.Result
			}
		}

		ic, _ := x.getInterchain(ibtp.To)
		ic.SourceInterchainCounter[ibtp.From] = ibtp.Index
		x.setInterchain(ibtp.To, ic)
		x.updateInterchainMeta(ibtp)
	} else {
		interchain.ReceiptCounter[ibtp.To] = ibtp.Index
		x.setInterchain(ibtp.From, interchain)
		if srcChainService.ChainId == srcChainService.BxhId {
			data, _ := ibtp.Marshal()
			x.CrossInvoke(constant.InterBrokerContractAddr.Address().String(), "InvokeReceipt", pb.Bytes(data))
		}

		ic, _ := x.getInterchain(ibtp.To)
		ic.SourceReceiptCounter[ibtp.From] = ibtp.Index
		x.setInterchain(ibtp.To, ic)
		x.SetObject(IndexReceiptMapKey(getIBTPID(ibtp.From, ibtp.To, ibtp.Index)), x.GetTxHash())

		result := true
		if ibtp.Type == pb.IBTP_RECEIPT_FAILURE {
			result = false
		}
		x.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "RecordInvokeService",
			pb.String(ibtp.To),
			pb.String(ibtp.From),
			pb.Bool(result))
	}

	return nil
}

func (x *InterchainManager) updateInterchainMeta(ibtp *pb.IBTP) {
	meta := &InterchainMeta{
		TargetChain: ibtp.To,
		TxHash:      x.GetTxHash().String(),
		Timestamp:   x.GetTxTimeStamp(),
	}
	x.setInterchainMeta(x.indexSendInterchainMeta(ibtp.From), meta)

	meta.TargetChain = ibtp.From
	x.setInterchainMeta(x.indexReceiptInterchainMeta(ibtp.To), meta)
}

func (x *InterchainManager) notifySrcDst(ibtp *pb.IBTP, statusChange *StatusChange) {
	m := make(map[string]uint64)
	srcChainService, _ := x.parseChainService(ibtp.From)
	dstChainService, _ := x.parseChainService(ibtp.To)

	notifySrc, notifyDst := statusChange.NotifyFlags()
	if notifySrc {
		if srcChainService.IsLocal {
			m[srcChainService.ChainId] = x.GetTxIndex()
			x.addToMultiTxNotifyMap(x.GetCurrentHeight(), statusChange.OtherIBTPIDs, true)
		} else {
			m[DEFAULT_UNION_PIER_ID] = x.GetTxIndex()
		}
	}
	if notifyDst {
		if dstChainService.IsLocal {
			m[dstChainService.ChainId] = x.GetTxIndex()
			x.addToMultiTxNotifyMap(x.GetCurrentHeight(), statusChange.OtherIBTPIDs, false)
		} else {
			m[DEFAULT_UNION_PIER_ID] = x.GetTxIndex()
		}
	}

	x.PostInterchainEvent(m)
}

func (x *InterchainManager) beginTransaction(ibtp *pb.IBTP, isFailed bool) (*StatusChange, error) {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	bxhID0, _, _ := ibtp.ParseFrom()
	bxhID1, _, _ := ibtp.ParseTo()
	timeoutHeight := uint64(ibtp.TimeoutHeight)
	// TODO: disable transaction management for inter-bitxhub transaction temporarily
	if bxhID0 != bxhID1 {
		timeoutHeight = 0
	}

	res := boltvm.Success(nil)
	if ibtp.Group == nil {
		res = x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "Begin", pb.String(txId), pb.Uint64(timeoutHeight), pb.Bool(isFailed))
		if !res.Ok {
			return nil, fmt.Errorf(string(res.Result))
		}
	} else {
		globalID, err := genGlobalTxID(ibtp)
		if err != nil {
			return nil, err
		}
		res = x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "BeginMultiTXs", pb.String(globalID), pb.Uint64(timeoutHeight), pb.Bool(isFailed), pb.String(ibtp.To))
		if !res.Ok {
			return nil, fmt.Errorf(string(res.Result))
		}
	}

	change := StatusChange{}
	if err := json.Unmarshal(res.Result, &change); err != nil {
		return nil, err
	}

	return &change, nil
}

func (x *InterchainManager) reportTransaction(ibtp *pb.IBTP) (*StatusChange, error) {
	txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
	res := x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "Report", pb.String(txId), pb.Int32(int32(ibtp.Type)))
	if !res.Ok {
		return nil, fmt.Errorf(string(res.Result))
	}

	change := StatusChange{}
	if err := json.Unmarshal(res.Result, &change); err != nil {
		return nil, err
	}

	return &change, nil
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
		key = IndexMapKey(id)
	} else {
		key = IndexReceiptMapKey(id)
	}
	exist := x.GetObject(key, &hash)
	if !exist {
		return boltvm.Error("this ibtp id does not exist")
	}

	return boltvm.Success(hash.Bytes())
}

func genGlobalTxID(ibtp *pb.IBTP) (string, error) {
	m := make(map[string]uint64)

	m[ibtp.To] = ibtp.Index
	for i, key := range ibtp.Group.Keys {
		m[key] = ibtp.Group.Vals[i]
	}

	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(append([]byte(ibtp.From), data...))

	return types.NewHash(hash[:]).String(), nil
}

func IndexMapKey(id string) string {
	return fmt.Sprintf("index-tx-%s", id)
}

func IndexReceiptMapKey(id string) string {
	return fmt.Sprintf("index-receipt-tx-%s", id)
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
		return nil, fmt.Errorf("can not get service %s info: %s", id, string(res.Result))
	}

	if err := json.Unmarshal(res.Result, service); err != nil {
		return nil, fmt.Errorf("unmarshal service of ID %s: %w", id, err)
	}

	return service, nil
}

func (x *InterchainManager) checkAppchainAvailability(id string) error {
	chain, err := x.getAppchainInfo(id)
	if err != nil {
		return err
	}

	if !chain.IsAvailable() {
		return fmt.Errorf("appchain %s is not available", id)
	}

	return nil
}

func (x *InterchainManager) checkBitXHubAvailability(id string) error {
	chain, err := x.getAppchainInfo(id)
	if err != nil {
		return err
	}

	if !chain.IsBitXHub() {
		return fmt.Errorf("chain %s is not BitXHub", id)
	}

	if !chain.IsAvailable() {
		return fmt.Errorf("bitxhub %s is not available", id)
	}

	return nil
}

func (x *InterchainManager) checkServiceAvailability(chainServiceID string) error {
	service, err := x.getServiceByID(chainServiceID)
	if err != nil {
		return fmt.Errorf("cannot get service by %s", chainServiceID)
	}

	if !service.IsAvailable() {
		return fmt.Errorf("service %s is not available", chainServiceID)
	}

	return nil
}

func (x *InterchainManager) checkServiceIndex(ibtp *pb.IBTP, counter map[string]uint64, dstChainService *ChainService) error {
	if dstChainService.IsLocal {
		dstService, _ := x.getServiceByID(dstChainService.getChainServiceId())
		if dstService == nil || dstService.Ordered {
			if err := checkIndex(counter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
				return err
			}
		}
	} else {
		if err := checkIndex(counter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
			return err
		}
	}

	return nil
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

func (x *InterchainManager) addToMultiTxNotifyMap(height uint64, ibtpIDs []string, toSrc bool) {
	if len(ibtpIDs) == 0 {
		return
	}

	var multiTxNotifyMap map[string][]string
	ok := x.GetObject(MultiTxNotifyKey(height), &multiTxNotifyMap)
	if !ok {
		multiTxNotifyMap = make(map[string][]string)
	}

	if toSrc {
		from, _, _, _ := pb.ParseIBTPID(ibtpIDs[0])
		_, chainID, _, _ := pb.ParseFullServiceID(from)
		multiTxNotifyMap[chainID] = append(multiTxNotifyMap[chainID], ibtpIDs...)
	} else {
		for _, ibtpID := range ibtpIDs {
			_, to, _, _ := pb.ParseIBTPID(ibtpIDs[0])
			_, chainID, _, _ := pb.ParseFullServiceID(to)
			multiTxNotifyMap[chainID] = append(multiTxNotifyMap[chainID], ibtpID)
		}
	}

	x.SetObject(MultiTxNotifyKey(height), multiTxNotifyMap)
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

func getIBTPID(from, to string, index uint64) string {
	return fmt.Sprintf("%s-%s-%d", from, to, index)
}

func MultiTxNotifyKey(height uint64) string {
	return fmt.Sprintf("%s-%d", MULTITX_PREFIX, height)
}
