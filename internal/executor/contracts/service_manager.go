package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

const (
	ServicePreKey = "service-"

	REGISTERED = 0
	APPROVED   = 1
)

type ServiceManager struct {
	boltvm.Stub
}

type Service struct {
	Id         string          `json:"id"`         // service id
	Name       string          `json:"name"`       // service name
	Type       string          `json:"type"`       // service type
	Desc       string          `json:"desc"`       // service description
	Ordered    bool            `json:"ordered"`    // service should be in order or not
	Permission []string        `json:"permission"` // counter party services which are allowed to call the service
	Items      map[string]Item `json:"items"`      // service entities
	Status     int32           `json:"status"`     // 0 => registered, 1 => approved, -1 => rejected
}

type Item struct {
	Method     string   `json:"method"`      // method desc
	ArgType    []string `json:"arg_type"`    // method arg type
	ReturnType []string `json:"return_type"` // method return type
	Status     int32    `json:"status"`      // -1 => rejected, 1 => approved
}

type auditRecord struct {
	Service    *Service `json:"service"`
	IsApproved bool     `json:"is_approved"`
	Desc       string   `json:"desc"`
}

type auditItemRecord struct {
	ServiceId  string `json:"service_id"`
	Item       *Item  `json:"item"`
	IsApproved bool   `json:"is_approved"`
	Desc       string `json:"desc"`
}

func NewServiceMng() agency.Contract {
	return &ServiceManager{}
}

func (sm *ServiceManager) Register(id, name, desc, typ string, ordered bool, permit string, itemData []byte) *boltvm.Response {
	res := sm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(id))
	if !res.Ok {
		return res
	}

	sm.Logger().Info("get appchain success")
	var items map[string]Item
	if err := json.Unmarshal(itemData, &items); err != nil {
		return boltvm.Error(err.Error())
	}

	permission := strings.Split(permit, ",")

	service := &Service{
		Id:         id,
		Name:       name,
		Desc:       desc,
		Type:       typ,
		Ordered:    ordered,
		Permission: permission,
		Items:      items,
		Status:     REGISTERED,
	}

	ok := sm.Has(sm.serviceKey(id))
	if ok {
		sm.Logger().WithFields(logrus.Fields{
			"id": id,
		}).Debug("Service has registered")
		sm.GetObject(sm.serviceKey(id), service)
	} else {
		sm.SetObject(sm.serviceKey(id), service)
		sm.Logger().WithFields(logrus.Fields{
			"id": id,
		}).Info("Service register successfully")
	}
	body, err := json.Marshal(service)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(body)
}

func (sm *ServiceManager) Call(data []byte) *boltvm.Response {
	var ibtp pb.IBTP
	err := ibtp.Unmarshal(data)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	res := sm.CrossInvoke(constant.InterchainContractAddr.Address().String(), "HandleIBTP", pb.Bytes(data))
	return res
}

func (sm *ServiceManager) Update(name string, desc string, itemData []byte) *boltvm.Response {
	id := sm.Caller()
	ok := sm.Has(sm.serviceKey(id))
	if !ok {
		return boltvm.Error("register service firstly")
	}

	service := sm.getServiceInfo(id)

	if service.Status == REGISTERED {
		return boltvm.Error("this service is being audited")
	}

	var items map[string]Item
	if err := json.Unmarshal(itemData, &items); err != nil {
		return boltvm.Error(err.Error())
	}
	service = &Service{
		Name:   name,
		Desc:   desc,
		Items:  items,
		Status: service.Status,
	}

	sm.SetObject(sm.serviceKey(id), service)
	return boltvm.Success(nil)
}

func (sm *ServiceManager) GetServiceInfo(id string) *boltvm.Response {
	service, err := json.Marshal(sm.getServiceInfo(id))
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(service)
}

func (sm *ServiceManager) getServiceInfo(id string) *Service {
	service := &Service{}
	sm.GetObject(sm.serviceKey(id), service)
	return service
}

func (sm *ServiceManager) AddItems(itemData []byte) *boltvm.Response {
	id := sm.Caller()
	ok := sm.Has(sm.serviceKey(id))
	if !ok {
		return boltvm.Error("register service firstly")
	}

	service := sm.getServiceInfo(id)

	if service.Status == REGISTERED {
		return boltvm.Error("this service is being audited")
	}

	var items map[string]Item
	if err := json.Unmarshal(itemData, &items); err != nil {
		return boltvm.Error(err.Error())
	}

	for method, item := range items {
		service.Items[method] = item
	}
	sm.SetObject(sm.serviceKey(id), service)
	return boltvm.Success(nil)
}

func (sm *ServiceManager) Audit(proposer string, isApproved int32, desc string) *boltvm.Response {
	if res := sm.IsAdmin(); !res.Ok {
		return res
	}

	service := &Service{}
	ok := sm.GetObject(sm.serviceKey(proposer), service)
	if !ok {
		return boltvm.Error("this service does not exist")
	}

	service.Status = isApproved

	for _, item := range service.Items {
		item.Status = APPROVED
	}

	record := &auditRecord{
		Service:    service,
		IsApproved: isApproved == APPROVED,
		Desc:       desc,
	}

	var records []*auditRecord
	sm.GetObject(sm.auditRecordKey(proposer), &records)
	records = append(records, record)

	sm.SetObject(sm.auditRecordKey(proposer), records)
	sm.SetObject(sm.serviceKey(proposer), service)

	return boltvm.Success([]byte(fmt.Sprintf("audit %s successfully", proposer)))
}

func (sm *ServiceManager) AuditItem(proposer string, isApproved int32, method string, desc string) *boltvm.Response {
	if res := sm.IsAdmin(); !res.Ok {
		return res
	}

	service := &Service{}
	ok := sm.GetObject(sm.serviceKey(proposer), service)
	if !ok {
		return boltvm.Error("this service does not exist")
	}

	item, ok := service.Items[method]
	if !ok {
		return boltvm.Error(fmt.Sprintf("this method:%s does not exist", method))
	}
	item.Status = isApproved
	record := &auditItemRecord{
		ServiceId:  proposer,
		Item:       &item,
		IsApproved: isApproved == APPROVED,
		Desc:       desc,
	}
	var records []*auditItemRecord
	sm.GetObject(sm.auditItemRecordKey(proposer), &records)
	records = append(records, record)

	sm.SetObject(sm.auditItemRecordKey(proposer), records)
	sm.SetObject(sm.serviceKey(proposer), service)

	return boltvm.Success([]byte(fmt.Sprintf("audit %s successfully", method)))

}

func (sm *ServiceManager) ListService() *boltvm.Response {
	ok, value := sm.Query(ServicePreKey)
	if !ok {
		return boltvm.Success(nil)
	}

	ret := make([]*Service, 0)
	for _, data := range value {
		service := &Service{}
		if err := json.Unmarshal(data, service); err != nil {
			return boltvm.Error(err.Error())
		}
		ret = append(ret, service)
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

func (sm *ServiceManager) DeleteService(id string) *boltvm.Response {
	if res := sm.IsAdmin(); !res.Ok {
		return res
	}
	sm.Delete(sm.serviceKey(id))
	return boltvm.Success([]byte(fmt.Sprintf("delete service:%s", id)))
}

func (sm *ServiceManager) IsAdmin() *boltvm.Response {
	ret := sm.CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", pb.String(sm.Caller()))
	is, err := strconv.ParseBool(string(ret.Result))
	if err != nil {
		return boltvm.Error(fmt.Errorf("judge caller type: %w", err).Error())
	}

	if !is {
		return boltvm.Error("caller is not an admin account")
	}
	return boltvm.Success([]byte("1"))
}

func (sm *ServiceManager) serviceKey(id string) string {
	return ServicePreKey + id
}

func (sm *ServiceManager) auditRecordKey(id string) string {
	return "audit-record-" + id
}

func (sm *ServiceManager) auditItemRecordKey(id string) string {
	return "audit-item-record-" + id
}

func (service *Service) checkPermission(serviceId string) bool {
	for _, id := range service.Permission {
		if id == serviceId {
			return true
		}
	}

	return false
}
