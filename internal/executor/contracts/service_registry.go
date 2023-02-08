package contracts

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/eth-kit/ledger"
)

const (
	RootDomain           = "hub"
	PriceLenLevel        = "priceLevel"
	BitxhubTokenPrice    = "tokenPrice"
	Level1Domain         = "level1Domain"
	ResolverMap          = "resolverMap"
	PreRegister          = "preRegister"
	PermissionController = "permissionController"
	GRACEPERIOD          = uint64(90 * 24 * 60 * 60)
	SecondTime           = int64(time.Second)
	LevelZero            = 0
	LevelOne             = 1
	LevelTwo             = 2
	RegisteredDomain     = 1
	ReviewedDomain       = 0
)

type ServiceRegistry struct {
	boltvm.Stub
}

type Price struct {
	base    uint64
	premium uint64
}

type PriceLevel struct {
	Price1Letter uint64 `json:"price1Letter"`
	Price2Letter uint64 `json:"price2Letter"`
	Price3Letter uint64 `json:"price3Letter"`
	Price4Letter uint64 `json:"price4Letter"`
	Price5Letter uint64 `json:"price5Letter"`
}

type ServDomainRec struct {
	Owner             string          `json:"owner"`
	Resolver          string          `json:"resolver"`
	SubDomain         map[string]bool `json:"subDomain"`
	SubDomainInReview map[string]bool `json:"subDomainInReview"` // 审核中子域名
	Parent            string          `json:"parent"`
}

type SubDomainProposalData struct {
	ParentName  string `json:"parent_name"`
	ParentOwner string `json:"parent_owner"`
	SonName     string `json:"son_name"`
	Onwer       string `json:"owner"`
	Resolver    string `json:"resolver"`
	ServiceName string `json:"service_name"`
}

type ServDomain struct {
	Name       string `json:"name"`
	Level      int    `json:"level"`
	Status     int    `json:"status"`
	ParentName string `json:"parent_name"`
}

// Register a first-level domain name
func (sr ServiceRegistry) Register(name string, duration uint64, resolver string) *boltvm.Response {
	if name == "" {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id can not be an empty string")
	}
	owner := sr.Caller()
	if !checkBxhAddress(owner) {
		return boltvm.Error(boltvm.BnsErrCode, "The address is not valid")
	}

	if resolver == "" || !sr.checkResolverAddress(resolver) {
		return boltvm.Error(boltvm.BnsErrCode, "The resolver is not in the list")
	}

	if duration == 0 {
		return boltvm.Error(boltvm.BnsErrCode, "The duration can not be zero")
	}

	account := sr.GetAccount(owner).(ledger.IAccount)
	balance := account.GetBalance().Uint64()

	level1Domain := sr.getLevel1Domain()
	registerName := generateSubDomain(RootDomain, name)
	if level1Domain[registerName]+GRACEPERIOD > uint64(sr.GetTxTimeStamp()/SecondTime) || uint64(sr.GetTxTimeStamp()/SecondTime)+duration+GRACEPERIOD < uint64(sr.GetTxTimeStamp()/SecondTime)+GRACEPERIOD {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id registered or in GRACEPERIOD")
	}

	price, err := sr.getPrice(name, level1Domain[name], duration)
	if err != nil {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("get register price: %v", err))
	}
	registerCost := price.premium + price.base
	if balance < registerCost {
		return boltvm.Error(boltvm.BnsErrCode, "Not enough Bitxhub Token provided")
	}
	sr.setSubDomainOwner(RootDomain, name, owner)
	sr.setRecord(registerName, owner, resolver, RootDomain)

	level1Domain[registerName] = uint64(sr.GetTxTimeStamp()/SecondTime) + duration
	sr.SetObject(Level1Domain, level1Domain)

	res := sr.CrossInvoke(resolver, "SetServDomainData",
		pb.String(registerName),
		pb.Uint64(1), pb.String(owner), pb.String(""), pb.String(""), pb.String(""))
	if !res.Ok {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("register servDomainData error: %v", err))
	}

	account.SubBalance(new(big.Int).SetUint64(registerCost))

	return boltvm.Success(nil)
}

// Renew First-level domain name renewal
func (sr ServiceRegistry) Renew(name string, duration uint64) *boltvm.Response {
	if name == "" {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id can not be an empty string")
	}
	if duration == 0 {
		return boltvm.Error(boltvm.BnsErrCode, "The duration can not be zero")
	}
	if !sr.checkNameAvailable(name) {
		return boltvm.Error(boltvm.BnsErrCode, "The domain must register first")
	}
	account := sr.GetAccount(sr.Caller()).(ledger.IAccount)
	balance := account.GetBalance().Uint64()

	level1Domain := sr.getLevel1Domain()
	if level1Domain[name]+GRACEPERIOD < uint64(sr.GetTxTimeStamp()/SecondTime) || uint64(sr.GetTxTimeStamp()/SecondTime)+duration+GRACEPERIOD < uint64(sr.GetTxTimeStamp()/SecondTime)+GRACEPERIOD {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id must registered in GRACEPERIOD ")
	}
	price, err := sr.getPrice(name, level1Domain[name], duration)
	if err != nil {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("get register price: %v", err))
	}
	registerCost := price.premium + price.base
	if balance < registerCost {
		return boltvm.Error(boltvm.BnsErrCode, "Not enough Bitxhub Token provided")
	}
	level1Domain[name] = level1Domain[name] + duration
	sr.SetObject(Level1Domain, level1Domain)
	account.SubBalance(new(big.Int).SetUint64(registerCost))

	return boltvm.Success(nil)
}

func (sr ServiceRegistry) AllocateSubDomain(parentName string, sonName string, owner string, resolver string, serviceName string) *boltvm.Response {
	if parentName == "" {
		return boltvm.Error(boltvm.BnsErrCode, "The parentDomain name can not be an empty string")
	}
	if sonName == "" {
		return boltvm.Error(boltvm.BnsErrCode, "The sonDomain name can not be an empty string")
	}
	if !sr.checkNameAvailable(parentName) {
		return boltvm.Error(boltvm.BnsErrCode, "The domain must register first")
	}

	if !checkBxhAddress(owner) {
		return boltvm.Error(boltvm.BnsErrCode, "The address is not valid")
	}

	if resolver == "" || !sr.checkResolverAddress(resolver) {
		return boltvm.Error(boltvm.BnsErrCode, "The resolver is not in the list")
	}

	if !sr.authorised(parentName) {
		return boltvm.Error(boltvm.BnsErrCode, "The domain name does not belong to you")
	}

	sonDomain := generateSubDomain(parentName, sonName)
	parentDomainRec := ServDomainRec{}
	ok := sr.GetObject(parentName, &parentDomainRec)
	if !ok {
		return boltvm.Error(boltvm.BnsErrCode, "parentName not exist")
	}
	if parentDomainRec.SubDomainInReview == nil {
		parentDomainRec.SubDomainInReview = make(map[string]bool)
	}
	parentDomainRec.SubDomainInReview[sonDomain] = true
	sr.SetObject(parentName, parentDomainRec)

	preRegister := make(map[string]bool)
	sr.GetObject(PreRegister, &preRegister)
	if preRegister[parentName] {
		return boltvm.Error(boltvm.BnsErrCode, "current proposal is not end")
	}
	subDomainProposalData := SubDomainProposalData{
		ParentName:  parentName,
		ParentOwner: sr.Caller(),
		SonName:     sonName,
		Onwer:       owner,
		Resolver:    resolver,
		ServiceName: serviceName,
	}
	subDomainProposalDataBytes, err := json.Marshal(subDomainProposalData)
	if err != nil {
		return boltvm.Error(boltvm.BnsErrCode, "marshal err")
	}
	event := governance.EventRegister
	proposalRes := sr.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(sr.Caller()),
		pb.String(string(event)),
		pb.String(string(BnsMgr)),
		pb.String(generateSubDomain(parentName, sonName)),
		pb.String(""), // no last status
		pb.String(""),
		pb.Bytes(subDomainProposalDataBytes),
	)
	if !proposalRes.Ok {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("submit proposal error: %s", string(proposalRes.Result)))
	}

	preRegister[parentName] = true
	sr.SetObject(PreRegister, preRegister)

	return getGovernanceRet(string(proposalRes.Result), subDomainProposalDataBytes)
}

func (sr ServiceRegistry) Manage(eventTyp, proposalResult, _, _ string, extra []byte) *boltvm.Response {
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			if err := sr.manageRegisterApprove(extra); err != nil {
				return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("manage register approve error: %v", err))
			}
		}
	} else {
		switch eventTyp {
		case string(governance.EventRegister):
			if err := sr.manageRegisterReject(extra); err != nil {
				return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("manage register approve error: %v", err))
			}
		}
	}

	return boltvm.Success(nil)

}

func (sr ServiceRegistry) manageRegisterApprove(extra []byte) error {
	domainData := SubDomainProposalData{}
	if err := json.Unmarshal(extra, &domainData); err != nil {
		return fmt.Errorf("unmarshal data error: %v", err)
	}
	parentName := domainData.ParentName
	sonName := domainData.SonName
	owner := domainData.Onwer
	resolver := domainData.Resolver

	sonDomain := sr.setSubRecord(parentName, sonName, owner, resolver)

	res := sr.CrossInvoke(resolver, "SetServDomainData",
		pb.String(sonDomain),
		pb.Uint64(1), pb.String(owner), pb.String(domainData.ServiceName), pb.String(""), pb.String(""))
	if !res.Ok {
		return fmt.Errorf("register servDomainData error: %v", res.Result)
	}

	preRegister := make(map[string]bool)
	sr.GetObject(PreRegister, &preRegister)
	preRegister[parentName] = false
	sr.SetObject(PreRegister, preRegister)

	return nil
}

func (sr ServiceRegistry) manageRegisterReject(extra []byte) error {
	domainData := SubDomainProposalData{}
	if err := json.Unmarshal(extra, &domainData); err != nil {
		return fmt.Errorf("unmarshal data error: %v", err)
	}

	preRegister := make(map[string]bool)
	parentName := domainData.ParentName

	parentDomainRec := ServDomainRec{}
	ok := sr.GetObject(parentName, &parentDomainRec)
	if !ok {
		return fmt.Errorf("parentName not exist")
	}
	sonDomain := generateSubDomain(parentName, domainData.SonName)
	if parentDomainRec.SubDomain == nil {
		parentDomainRec.SubDomain = make(map[string]bool)
	}
	if parentDomainRec.SubDomainInReview == nil {
		parentDomainRec.SubDomainInReview = make(map[string]bool)
	}
	parentDomainRec.SubDomainInReview[sonDomain] = false
	sr.SetObject(parentName, parentDomainRec)

	sr.GetObject(PreRegister, &preRegister)
	preRegister[parentName] = false
	sr.SetObject(PreRegister, preRegister)
	return nil
}

func (sr ServiceRegistry) DeleteSecondDomain(name string) *boltvm.Response {
	if name == "" {
		return boltvm.Error(boltvm.BnsErrCode, "The domain name can not be an empty string")
	}
	nameArr := strings.Split(name, ".")
	if len(nameArr) != 3 {
		return boltvm.Error(boltvm.BnsErrCode, "The domain name must be second")
	}
	if !sr.checkNameAvailable(name) {
		return boltvm.Error(boltvm.BnsErrCode, "The domain must be Allocate first")
	}
	if !sr.authorised(name) {
		return boltvm.Error(boltvm.BnsErrCode, "The domain name does not belong to you")
	}
	serviceDomainRec := ServDomainRec{}
	ok := sr.GetObject(name, &serviceDomainRec)
	if !ok {
		return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
	}
	parentServiceDomainRec := ServDomainRec{}
	ok = sr.GetObject(serviceDomainRec.Parent, &parentServiceDomainRec)
	if !ok {
		return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
	}
	parentServiceDomainRec.SubDomain[name] = false
	sr.SetObject(serviceDomainRec.Parent, parentServiceDomainRec)
	res := sr.CrossInvoke(serviceDomainRec.Resolver, "DeleteServDomainData",
		pb.String(name))
	if !res.Ok {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("delete servDomainData error: %v", res.Result))
	}
	sr.Delete(name)
	return boltvm.Success(nil)
}

// SetPriceLevel Update registration price
func (sr ServiceRegistry) SetPriceLevel(price1Letter uint64, price2Letter uint64, price3Letter uint64, price4Letter uint64, price5Letter uint64) *boltvm.Response {
	addr := sr.Caller()
	res := sr.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(addr), pb.String(string(GovernanceAdmin)))
	if !res.Ok || "false" == string(res.Result) {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, "you have no permission")
	}
	priceLevel := PriceLevel{
		Price1Letter: price1Letter,
		Price2Letter: price2Letter,
		Price3Letter: price3Letter,
		Price4Letter: price4Letter,
		Price5Letter: price5Letter,
	}
	sr.SetObject(PriceLenLevel, priceLevel)
	return boltvm.Success(nil)
}

func (sr ServiceRegistry) GetPriceLevel() *boltvm.Response {
	priceLevel := PriceLevel{}
	ok := sr.GetObject(PriceLenLevel, &priceLevel)
	if !ok {
		return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
	}
	servDomainBytes, err := json.Marshal(priceLevel)
	if err != nil {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("marshal servDomainData error: %v", err))
	}
	return boltvm.Success(servDomainBytes)
}

func (sr ServiceRegistry) SetTokenPrice(tokenPrice uint64) *boltvm.Response {
	if tokenPrice == 0 {
		return boltvm.Error(boltvm.BnsErrCode, "The Token Price can not be zero")
	}
	addr := sr.Caller()
	res := sr.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(addr), pb.String(string(GovernanceAdmin)))
	if !res.Ok || "false" == string(res.Result) {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, "you have no permission")
	}
	sr.SetObject(BitxhubTokenPrice, tokenPrice)
	return boltvm.Success(nil)
}

func (sr ServiceRegistry) GetTokenPrice() *boltvm.Response {
	var tokenPrice uint64
	ok := sr.GetObject(BitxhubTokenPrice, &tokenPrice)
	if !ok {
		return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
	}
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, tokenPrice)
	return boltvm.Success(res)
}

// GetDomainExpires Get the expiration time of the first-level domain name
func (sr ServiceRegistry) GetDomainExpires(name string) *boltvm.Response {
	if name == "" {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id can not be an empty string")
	}
	expires := sr.getLevel1Domain()
	if expires[name] == 0 {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id is not registered")
	}
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, expires[name])
	return boltvm.Success(res)
}

func (sr ServiceRegistry) RecordExists(name string) *boltvm.Response {
	if name == "" {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id can not be an empty string")
	}
	ok := sr.Has(name)
	return boltvm.Success([]byte(strconv.FormatBool(ok)))
}

func (sr ServiceRegistry) Owner(name string) *boltvm.Response {
	if name == "" || !sr.checkNameAvailable(name) {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id must be registered")
	}
	serviceDomainRec := ServDomainRec{}
	ok := sr.GetObject(name, &serviceDomainRec)
	if !ok {
		return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
	}
	return boltvm.Success([]byte(serviceDomainRec.Owner))
}

func (sr ServiceRegistry) Resolver(name string) *boltvm.Response {
	if name == "" || !sr.checkNameAvailable(name) {
		return boltvm.Error(boltvm.BnsErrCode, "The domain id must be registered")
	}
	serviceDomainRec := ServDomainRec{}
	ok := sr.GetObject(name, &serviceDomainRec)
	if !ok {
		return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
	}
	return boltvm.Success([]byte(serviceDomainRec.Resolver))
}

func (sr ServiceRegistry) GetSubDomain(name string) *boltvm.Response {
	var subDomains []string
	if name == RootDomain {
		var servDomain map[string]uint64
		ok := sr.GetObject(Level1Domain, &servDomain)
		if !ok {
			return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
		}
		for k := range servDomain {
			subDomains = append(subDomains, k)
		}
	} else {
		serviceDomainRec := ServDomainRec{}
		ok := sr.GetObject(name, &serviceDomainRec)
		if !ok {
			return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
		}
		for k := range serviceDomainRec.SubDomain {
			if serviceDomainRec.SubDomain[k] {
				subDomains = append(subDomains, k)
			}
		}

	}
	subDomainsBytes, err := json.Marshal(subDomains)
	if err != nil {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("marshal servDomainData error: %v", err))
	}
	return boltvm.Success(subDomainsBytes)
}

func (sr ServiceRegistry) IsApproved(account string, operator string) *boltvm.Response {
	permissionController := make(map[string]map[string]bool)
	ok := sr.GetObject(PermissionController, &permissionController)
	if !ok {
		return boltvm.Success([]byte(strconv.FormatBool(false)))
	}
	return boltvm.Success([]byte(strconv.FormatBool(permissionController[account][operator])))
}

func (sr ServiceRegistry) getPrice(name string, expires uint64, duration uint64) (Price, error) {
	price := Price{}

	priceLevel := PriceLevel{}
	ok := sr.GetObject(PriceLenLevel, &priceLevel)
	if !ok {
		return price, fmt.Errorf("there is not exist key")
	}
	var tokenPrice uint64
	ok = sr.GetObject(BitxhubTokenPrice, &tokenPrice)
	if !ok {
		return price, fmt.Errorf("there is not exist key")
	}

	nameLen := len(name)
	var basePrice uint64
	if nameLen >= 5 {
		basePrice = priceLevel.Price5Letter * duration
	} else if nameLen == 4 {
		basePrice = priceLevel.Price4Letter * duration
	} else if nameLen == 3 {
		basePrice = priceLevel.Price3Letter * duration
	} else if nameLen == 2 {
		basePrice = priceLevel.Price2Letter * duration
	} else {
		basePrice = priceLevel.Price1Letter * duration
	}
	price.base = sr.attoCYNToWei(basePrice, tokenPrice)
	price.premium = sr.attoCYNToWei(sr.premium(name, expires, duration), tokenPrice)
	return price, nil
}

func (sr ServiceRegistry) premium(_ string, _ uint64, _ uint64) uint64 {
	return 0
}

func (sr ServiceRegistry) attoCYNToWei(amount uint64, tokenPrice uint64) uint64 {
	return amount * 1e8 / tokenPrice
}

func (sr ServiceRegistry) getLevel1Domain() map[string]uint64 {
	servDomain := make(map[string]uint64)
	ok := sr.GetObject(Level1Domain, &servDomain)
	if !ok {
		return servDomain
	}
	return servDomain
}

func (sr ServiceRegistry) authorised(name string) bool {
	servDomainRec := ServDomainRec{}
	caller := sr.Caller()
	ok := sr.GetObject(name, &servDomainRec)
	if !ok {
		return false
	}
	permissionController := make(map[string]map[string]bool)
	ok = sr.GetObject(PermissionController, &permissionController)
	if !ok {
		return false
	}
	return servDomainRec.Owner == caller || permissionController[sr.CurrentCaller()][string(constant.ServiceRegistryContractAddr)]
}

func (sr ServiceRegistry) setRecord(name string, owner string, resolver string, parent string) {
	servDomainRec := ServDomainRec{}
	sr.GetObject(name, &servDomainRec)
	servDomainRec.Owner = owner
	servDomainRec.Resolver = resolver
	servDomainRec.Parent = parent
	sr.SetObject(name, servDomainRec)
}

func (sr ServiceRegistry) setResolver(name string, resolver string) {
	servDomainRec := ServDomainRec{}
	sr.GetObject(name, &servDomainRec)
	servDomainRec.Resolver = resolver
	sr.SetObject(name, servDomainRec)
}

func (sr ServiceRegistry) setSubRecord(parentName string, sonName string, owner string, resolver string) string {
	sonDomain := sr.setSubDomainOwner(parentName, sonName, owner)
	sr.setResolver(sonDomain, resolver)
	return sonDomain
}

func (sr ServiceRegistry) setSubDomainOwner(parentName string, sonName string, owner string) string {
	sonDomain := generateSubDomain(parentName, sonName)
	parentDomainRec := ServDomainRec{}
	sr.GetObject(parentName, &parentDomainRec)
	if parentDomainRec.SubDomain == nil {
		parentDomainRec.SubDomain = make(map[string]bool)
	}
	if parentDomainRec.SubDomainInReview == nil {
		parentDomainRec.SubDomainInReview = make(map[string]bool)
	}
	parentDomainRec.SubDomainInReview[sonDomain] = false
	parentDomainRec.SubDomain[sonDomain] = true
	sr.SetObject(parentName, parentDomainRec)
	sonDomainRec := ServDomainRec{}
	sr.GetObject(sonDomain, &sonDomainRec)
	sonDomainRec.Owner = owner
	sonDomainRec.Parent = parentName
	sr.SetObject(sonDomain, sonDomainRec)
	return sonDomain
}

func generateSubDomain(parentName string, sonName string) string {
	return fmt.Sprintf("%s.%s", sonName, parentName)
}

func checkBxhAddress(address string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	res := re.MatchString(address)
	return res
}

func (sr ServiceRegistry) checkResolverAddress(resolver string) bool {
	resolverMap := make(map[string]bool)
	ok := sr.GetObject(ResolverMap, &resolverMap)
	if !ok {
		return false
	}
	return resolverMap[resolver]
}

func (sr ServiceRegistry) checkNameAvailable(name string) bool {
	res := sr.Has(name)
	return res
}

func (sr ServiceRegistry) GetAllDomains() *boltvm.Response {
	root := ServDomain{
		Name:   RootDomain,
		Level:  LevelZero,
		Status: RegisteredDomain,
	}
	var res []*ServDomain
	res = append(res, &root)

	// 处理一级域名
	var servDomain map[string]uint64
	ok := sr.GetObject(Level1Domain, &servDomain)
	if ok {
		for firstDomain := range servDomain {
			first := ServDomain{
				Name:       firstDomain,
				Level:      LevelOne,
				Status:     RegisteredDomain,
				ParentName: root.Name,
			}
			res = append(res, &first)

			serviceDomainFirst := ServDomainRec{}
			ok := sr.GetObject(firstDomain, &serviceDomainFirst)
			if !ok {
				return boltvm.Error(boltvm.BnsErrCode, "there is not exist key")
			}
			for secondDomain := range serviceDomainFirst.SubDomain {
				if serviceDomainFirst.SubDomain[secondDomain] {
					second := ServDomain{
						Name:       secondDomain,
						Level:      LevelTwo,
						Status:     RegisteredDomain,
						ParentName: first.Name,
					}
					res = append(res, &second)
				}
			}
			for secondDomain := range serviceDomainFirst.SubDomainInReview {
				if serviceDomainFirst.SubDomainInReview[secondDomain] {
					second := ServDomain{
						Name:       secondDomain,
						Level:      LevelTwo,
						Status:     ReviewedDomain,
						ParentName: first.Name,
					}
					res = append(res, &second)
				}
			}

		}
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		return boltvm.Error(boltvm.BnsErrCode, fmt.Sprintf("marshal servDomainData error: %v", err))
	}
	return boltvm.Success(resBytes)

}
