package contracts

import (
	"encoding/json"
	"strconv"

	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
)

const (
	adminRolesKey = "admin-roles"
)

type Role struct {
	boltvm.Stub
}

func (r *Role) GetRole() *boltvm.Response {
	var addrs []string
	r.GetObject(adminRolesKey, &addrs)

	for _, addr := range addrs {
		if addr == r.Caller() {
			return boltvm.Success([]byte("admin"))
		}
	}

	res := r.CrossInvoke(constant.AppchainMgrContractAddr.String(), "Appchain")
	if !res.Ok {
		return boltvm.Success([]byte("none"))
	}

	return boltvm.Success([]byte("appchain_admin"))
}

func (r *Role) IsAdmin(address string) *boltvm.Response {
	var addrs []string
	r.GetObject(adminRolesKey, &addrs)

	for _, addr := range addrs {
		if addr == address {
			return boltvm.Success([]byte(strconv.FormatBool(true)))
		}
	}

	return boltvm.Success([]byte(strconv.FormatBool(false)))
}

func (r *Role) GetAdminRoles() *boltvm.Response {
	var addrs []string
	r.GetObject(adminRolesKey, &addrs)

	ret, err := json.Marshal(addrs)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(ret)
}

func (r *Role) SetAdminRoles(addrs string) *boltvm.Response {
	as := make([]string, 0)
	if err := json.Unmarshal([]byte(addrs), &as); err != nil {
		return boltvm.Error(err.Error())
	}

	r.SetObject(adminRolesKey, as)
	return boltvm.Success(nil)
}
