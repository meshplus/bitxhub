package contracts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meshplus/bitxid"

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
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), err.Error()))
	}

	fullServiceID := fmt.Sprintf("%s:%s", bxhID, chainServiceID)
	interchain, ok := x.getInterchain(fullServiceID)
	if !ok {
		x.setInterchain(fullServiceID, interchain)
	}
	body, err := interchain.Marshal()
	if err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), err.Error()))
	}

	if err := x.postAuditInterchainEvent(fullServiceID); err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), fmt.Sprintf("post audit interchain event error: %v", err)))
	}

	return boltvm.Success(body)
}

func (x *InterchainManager) DeleteInterchain(id string) *boltvm.Response {
	x.Delete(serviceKey(id))

	if err := x.postAuditInterchainEvent(id); err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), fmt.Sprintf("post audit interchain event error: %v", err)))
	}
	return boltvm.Success(nil)
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
		return boltvm.Error(boltvm.InterchainNonexistentInterchainCode, fmt.Sprintf(string(boltvm.InterchainNonexistentInterchainMsg), id))
	}
	return boltvm.Success(data)
}

func (x *InterchainManager) HandleIBTPData(input []byte) *boltvm.Response {
	ibtp := &pb.IBTP{}
	err := ibtp.Unmarshal(input)
	if err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), err.Error()))
	}

	return x.HandleIBTP(ibtp)
}

func (x *InterchainManager) HandleIBTP(ibtp *pb.IBTP) *boltvm.Response {
	// Pier should retry if checkIBTP failed
	interchain, targetErr, bxhErr := x.checkIBTP(ibtp)
	if bxhErr != nil {
		return boltvm.Error(bxhErr.Code, string(bxhErr.Msg))
	}

	var change *StatusChange
	var err error
	if pb.IBTP_REQUEST == ibtp.Category() {
		change, err = x.beginTransaction(ibtp, targetErr != nil)
	} else if pb.IBTP_RESPONSE == ibtp.Category() {
		change, err = x.reportTransaction(ibtp)
	}
	if err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), err.Error()))
	}

	x.notifySrcDst(ibtp, change)

	ret := x.ProcessIBTP(ibtp, interchain, targetErr != nil)

	if err := x.postAuditInterchainEvent(ibtp.From); err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), fmt.Sprintf("post audit interchain event error: %v", err)))
	}
	if err := x.postAuditInterchainEvent(ibtp.To); err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), fmt.Sprintf("post audit interchain event error: %v", err)))
	}

	if ibtp.Extra != nil {
		x.Logger().Infof(string(ibtp.Extra))
		resp := x.CrossInvoke(constant.SuperviseMgrContractAddr.String(), "Record", pb.String(ibtp.From), pb.String(ibtp.To), pb.Bytes(ibtp.Extra))
		if !resp.Ok {
			return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), string(resp.Result)))
		}
	}

	return boltvm.Success(ret)
}

func (x *InterchainManager) checkIBTP(ibtp *pb.IBTP) (*pb.Interchain, *boltvm.BxhError, *boltvm.BxhError) {
	var targetError *boltvm.BxhError

	// In the full crossChain, the pier ensures that the format is correct,
	// if a format problem occurs, the pier is evil, this situation is not credible.
	srcChainService, err := x.parseChainService(ibtp.From)
	if err != nil {
		return nil, nil, boltvm.BError(boltvm.InterchainInvalidIBTPParseSourceErrorCode, fmt.Sprintf(string(boltvm.InterchainInvalidIBTPParseSourceErrorMsg), err.Error()))
	}

	dstChainService, err := x.parseChainService(ibtp.To)
	if err != nil {
		return nil, nil, boltvm.BError(boltvm.InterchainInvalidIBTPParseDestErrorCode, fmt.Sprintf(string(boltvm.InterchainInvalidIBTPParseDestErrorMsg), err.Error()))
	}

	var interchain *pb.Interchain
	if isDID, _ := ibtp.CheckFormat(); isDID {
		interchain, _ = x.getInterchain(ibtp.From)
	} else {
		interchain, _ = x.getInterchain(srcChainService.getFullServiceId())
	}

	if pb.IBTP_REQUEST == ibtp.Category() {
		// if src chain service is from appchain registered in current bitxhub, check service index
		if srcChainService.IsLocal {
			if err := x.checkSourceAvailability(srcChainService); err != nil {
				return nil, nil, err
			}

			// Ordered identifies whether the dst service needs to be invoked sequentially, that is, check index
			// - Large-scale cross-chain services: not local
			// - Services that are registered on the bitxhub and need to be called in order: is local && ordered == true
			// - Bitxhub service: is local && ChainId == BxhId
			var ordered bool
			ordered, targetError = x.checkTargetAvailability(srcChainService, dstChainService, ibtp.Type)

			if ordered {
				if err := checkIndex(interchain.InterchainCounter[ibtp.To]+1, ibtp.Index); err != nil {
					return nil, nil, err
				}
			}
		} else {
			if !dstChainService.IsLocal {
				return nil, nil, boltvm.BError(boltvm.InterchainInvalidIBTPNotInCurBXHCode, fmt.Sprintf(string(boltvm.InterchainInvalidIBTPNotInCurBXHMsg), ibtp.ID()))
			}

			if err := x.checkBitXHubAvailability(srcChainService.BxhId); err != nil {
				return nil, nil, boltvm.BError(boltvm.InterchainSourceBitXHubNotAvailableCode, fmt.Sprintf(string(boltvm.InterchainSourceBitXHubNotAvailableMsg), srcChainService.BxhId, err))
			}

			var ordered bool
			ordered, targetError = x.checkTargetAvailability(srcChainService, dstChainService, ibtp.Type)

			if ordered {
				if err := checkIndex(interchain.InterchainCounter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
					return nil, nil, err
				}
			}
		}
	} else if ibtp.Category() == pb.IBTP_RESPONSE {
		// Situation which need to check the index
		// - Bitxhub service：dstService == nil
		// - The dst service needs to be invoked sequentially：dstService.Ordered
		dstService, _ := x.getServiceByID(dstChainService.getChainServiceId())
		if dstService == nil || dstService.Ordered {
			if err := checkIndex(interchain.ReceiptCounter[ibtp.To]+1, ibtp.Index); err != nil {
				return nil, nil, err
			}
		}
	} else {
		return nil, nil, boltvm.BError(boltvm.InterchainInvalidIBTPIllegalTypeCode, fmt.Sprintf(string(boltvm.InterchainInvalidIBTPIllegalTypeMsg), ibtp.Type))
	}

	return interchain, targetError, nil
}

func (x *InterchainManager) checkSourceAvailability(srcChainService *ChainService) *boltvm.BxhError {
	//if err := x.checkAppchainAvailability(srcChainService.ChainId); err != nil {
	//	return boltvm.BError(boltvm.InterchainSourceAppchainNotAvailableCode, fmt.Sprintf(string(boltvm.InterchainSourceAppchainNotAvailableMsg), srcChainService.ChainId, err.Error()))
	//}

	if err := x.checkServiceAvailability(srcChainService.getChainServiceId()); err != nil {
		return boltvm.BError(boltvm.InterchainSourceServiceNotAvailableCode, fmt.Sprintf(string(boltvm.InterchainSourceServiceNotAvailableMsg), srcChainService.getChainServiceId(), err.Error()))
	}

	return nil
}

// The first return value indicates whether the destination service needs to be invoked in order, that is, whether index needs to be checked
func (x *InterchainManager) checkTargetAvailability(srcChainService, dstChainService *ChainService, typ pb.IBTP_Type) (bool, *boltvm.BxhError) {
	ordered := true
	if pb.IBTP_INTERCHAIN == typ {
		if dstChainService.IsLocal {
			if dstChainService.ChainId == dstChainService.BxhId {
				return true, nil
			}

			//if err := x.checkAppchainAvailability(dstChainService.ChainId); err != nil {
			//	return boltvm.BError(boltvm.InterchainTargetAppchainNotAvailableCode, fmt.Sprintf(string(boltvm.InterchainTargetAppchainNotAvailableMsg), dstChainService.ChainId, err.Error()))
			//}

			dstService, err := x.getServiceByID(dstChainService.getChainServiceId())
			if err != nil {
				return true, boltvm.BError(boltvm.InterchainTargetServiceNotAvailableCode, fmt.Sprintf(string(boltvm.InterchainTargetServiceNotAvailableMsg), dstChainService.getChainServiceId(), err.Error()))
			}
			ordered = dstService.Ordered

			if !dstService.IsAvailable() {
				return ordered, boltvm.BError(boltvm.InterchainTargetServiceNotAvailableCode, fmt.Sprintf(string(boltvm.InterchainTargetServiceNotAvailableMsg), dstChainService.getChainServiceId(), fmt.Sprintf("current status of service %s is %v", dstChainService.getChainServiceId(), dstService.Status)))
			}

			if !dstService.CheckPermission(srcChainService.getFullServiceId()) {
				return ordered, boltvm.BError(boltvm.InterchainTargetServiceNoPermissionCode, fmt.Sprintf(string(boltvm.InterchainTargetServiceNoPermissionMsg), srcChainService.getFullServiceId(), dstChainService.getFullServiceId()))
			}
		} else {
			if err := x.checkBitXHubAvailability(dstChainService.BxhId); err != nil {
				return ordered, boltvm.BError(boltvm.InterchainTargetBitXHubNotAvailableCode, fmt.Sprintf(string(boltvm.InterchainTargetBitXHubNotAvailableMsg), dstChainService.BxhId, err))
			}
		}
	}

	return ordered, nil
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

func (x *InterchainManager) ProcessIBTP(ibtp *pb.IBTP, interchain *pb.Interchain, isTargetFail bool) []byte {
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
			return res.Result
		}

		ic, _ := x.getInterchain(ibtp.To)
		ic.SourceInterchainCounter[ibtp.From] = ibtp.Index
		x.setInterchain(ibtp.To, ic)
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
			pb.String(dstChainService.getFullServiceId()),
			pb.String(srcChainService.getFullServiceId()),
			pb.Bool(result))
	}

	if isTargetFail {
		res := &boltvm.Response{}
		res.Result = []byte("begin_failure")
		return res.Result
	}
	return nil
}

func (x *InterchainManager) notifySrcDst(ibtp *pb.IBTP, statusChange *StatusChange) {
	m := make(map[string]uint64)
	srcChainService, _ := x.parseChainService(ibtp.From)
	dstChainService, _ := x.parseChainService(ibtp.To)

	notifySrc, notifyDst := statusChange.NotifyFlags()
	if notifySrc {
		if srcChainService.IsLocal {
			if isDID, _ := ibtp.CheckFormat(); isDID {
				methodID, _ := ibtp.ParseDIDFrom()
				m[methodID] = x.GetTxIndex()
			} else {
				m[srcChainService.ChainId] = x.GetTxIndex()
			}
			x.addToMultiTxNotifyMap(x.GetCurrentHeight(), statusChange.OtherIBTPIDs, true)
		} else {
			m[DEFAULT_UNION_PIER_ID] = x.GetTxIndex()
		}
	}
	if notifyDst {
		if dstChainService.IsLocal {
			if isDID, _ := ibtp.CheckFormat(); isDID {
				methodID, _ := ibtp.ParseDIDTo()
				m[methodID] = x.GetTxIndex()
			} else {
				m[dstChainService.ChainId] = x.GetTxIndex()
			}
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
		if !isFailed {
			sourceFailed, err := x.checkTxStatusForTargetBxh(ibtp)
			if err != nil {
				return nil, err
			}
			isFailed = sourceFailed
		}
	}

	res := boltvm.Success(nil)
	if ibtp.Group == nil {
		res = x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "Begin", pb.String(txId), pb.Uint64(timeoutHeight), pb.Bool(isFailed))
		if !res.Ok {
			return nil, fmt.Errorf(string(res.Result))
		}
	} else {
		count := uint64(len(ibtp.Group.Keys) + 1)
		globalID, err := genGlobalTxID(ibtp)
		if err != nil {
			return nil, err
		}
		res = x.CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), "BeginMultiTXs", pb.String(globalID), pb.String(ibtp.ID()), pb.Uint64(timeoutHeight), pb.Bool(isFailed), pb.Uint64(count))
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
		return boltvm.Error(boltvm.InterchainWrongIBTPIDCode, fmt.Sprintf(string(boltvm.InterchainWrongIBTPIDMsg), id))
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
		return boltvm.Error(boltvm.InterchainNonexistentIBTPCode, fmt.Sprintf(string(boltvm.InterchainNonexistentIBTPMsg), id))
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

	if size != 2 && size != 3 && size != 4 {
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
	if len(splits) == 3 {
		return &ChainService{
			BxhId:     splits[0],
			ChainId:   splits[1],
			ServiceId: splits[2],
			IsLocal:   splits[0] == bxhId,
		}, nil
	}

	methodDID := "did:bitxhub:" + splits[2] + ":."
	resp := x.CrossInvoke(constant.MethodRegistryContractAddr.String(), "ResolveWithDoc", pb.String(methodDID))
	if !resp.Ok {
		return nil, fmt.Errorf("get method doc from %s failed: %s", methodDID, string(resp.Result))
	}
	var methodDoc *bitxid.MethodDoc
	if err := json.Unmarshal(resp.Result, &methodDoc); err != nil {
		return nil, fmt.Errorf("unmarshal method doc error: %w", err)
	}
	addr, err := pb.GetAddrFromDoc(methodDoc)
	if err != nil {
		return nil, err
	}
	resp = x.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetChainIdByAdmin", pb.String(addr))
	if !resp.Ok {
		return nil, fmt.Errorf(string(resp.Result))
	}
	return &ChainService{
		BxhId:     bxhId,
		ChainId:   string(resp.Result),
		ServiceId: splits[3],
		IsLocal:   true,
	}, nil
}

func (x *InterchainManager) GetBitXHubID() *boltvm.Response {
	id, err := x.getBitXHubID()
	if err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), err.Error()))
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
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", pb.String(id))
	if !res.Ok || string(res.Result) == FALSE {
		return fmt.Errorf("appchain %s is not available", id)
	}

	return nil
}

func (x *InterchainManager) checkBitXHubAvailability(id string) error {
	res := x.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailableBitxhub", pb.String(id))
	if !res.Ok || string(res.Result) == FALSE {
		return fmt.Errorf("chain %s is not available bitxhub", id)
	}

	return nil
}

func (x *InterchainManager) checkServiceAvailability(chainServiceID string) error {
	res := x.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "IsAvailable", pb.String(chainServiceID))
	if !res.Ok || string(res.Result) == FALSE {
		return fmt.Errorf("service %s is not available", chainServiceID)
	}

	return nil
}

func (x *InterchainManager) checkServiceIndex(ibtp *pb.IBTP, counter map[string]uint64, dstChainService *ChainService) *boltvm.BxhError {
	if dstChainService.IsLocal {
		dstService, _ := x.getServiceByID(dstChainService.getChainServiceId())
		if dstService == nil || dstService.Ordered {
			if isDID, _ := ibtp.CheckFormat(); isDID {
				if err := checkIndex(counter[ibtp.To]+1, ibtp.Index); err != nil {
					return err
				}
			} else {
				if err := checkIndex(counter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
					return err
				}
			}
		}
	} else {
		if err := checkIndex(counter[dstChainService.getFullServiceId()]+1, ibtp.Index); err != nil {
			return err
		}
	}

	return nil
}

func checkIndex(exp, cur uint64) *boltvm.BxhError {
	if cur < exp {
		return boltvm.BError(boltvm.InterchainIbtpIndexExistCode, fmt.Sprintf(string(boltvm.InterchainIbtpIndexExistMsg), exp, cur))
	}
	if cur > exp {
		return boltvm.BError(boltvm.InterchainIbtpIndexWrongCode, fmt.Sprintf(string(boltvm.InterchainIbtpIndexWrongMsg), exp, cur))
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

func serviceKey(id string) string {
	return fmt.Sprintf("%s-%s", INTERCHAINSERVICE_PREFIX, id)
}

func (x *InterchainManager) GetAllServiceIDs() *boltvm.Response {
	ret := make([]string, 0)
	ok, value := x.Query(INTERCHAINSERVICE_PREFIX)
	if !ok {
		return boltvm.Success(nil)
	}
	for _, data := range value {
		interchain := &pb.Interchain{}
		if err := interchain.Unmarshal(data); err != nil {
			return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), err.Error()))
		}
		ret = append(ret, interchain.ID)
	}

	if result, err := json.Marshal(ret); err != nil {
		return boltvm.Error(boltvm.InterchainInternalErrCode, fmt.Sprintf(string(boltvm.InterchainInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(result)
	}
}

func (x *InterchainManager) checkTxStatusForTargetBxh(ibtp *pb.IBTP) (bool, error) {
	targetBxhID, _, _ := ibtp.ParseTo()
	curBxhID, err := x.getBitXHubID()
	if err != nil {
		return false, err
	}

	if targetBxhID == curBxhID {
		proof := ibtp.GetExtra()
		if len(proof) == 0 {
			return false, fmt.Errorf("get empty proof from source bitxhub for IBTP %s", ibtp.ID())
		}

		bxhProof := &pb.BxhProof{}
		if err := bxhProof.Unmarshal(proof); err != nil {
			return false, err
		}
		if bxhProof.TxStatus == pb.TransactionStatus_BEGIN_ROLLBACK || bxhProof.TxStatus == pb.TransactionStatus_BEGIN_FAILURE {
			return true, nil
		}
	}

	return false, nil
}

func getIBTPID(from, to string, index uint64) string {
	return fmt.Sprintf("%s-%s-%d", from, to, index)
}

func MultiTxNotifyKey(height uint64) string {
	return fmt.Sprintf("%s-%d", MULTITX_PREFIX, height)
}

func (x *InterchainManager) postAuditInterchainEvent(fullServiceID string) error {
	ok, interchainData := x.Get(serviceKey(fullServiceID))
	if !ok {
		return fmt.Errorf("not found interchain %s", fullServiceID)
	}

	chainService, err := x.parseChainService(fullServiceID)
	if err != nil {
		return err
	}
	auditInfo := &pb.AuditRelatedObjInfo{
		AuditObj: interchainData,
		RelatedChainIDList: map[string][]byte{
			chainService.ChainId: {},
		},
		RelatedNodeIDList: map[string][]byte{},
	}
	x.PostEvent(pb.Event_AUDIT_INTERCHAIN, auditInfo)

	return nil
}
