package contracts

import (
	"encoding/json"
	"strconv"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub/internal/repo"
)

const (
	adminRolesKey = "admin-roles"
)

type Role struct {
	boltvm.Stub
}

func (r *Role) GetRole() *boltvm.Response {
	var admins []*repo.Admin
	r.GetObject(adminRolesKey, &admins)

	for _, admin := range admins {
		if admin.Address == r.Caller() {
			return boltvm.Success([]byte("admin"))
		}
	}

	res := r.CrossInvoke(constant.AppchainMgrContractAddr.String(), "Appchain")
	if res.Code != boltvm.Normal {
		return boltvm.Success([]byte("none"))
	}

	return boltvm.Success([]byte("appchain_admin"))
}

func (r *Role) IsAdmin(address string) *boltvm.Response {
	return boltvm.Success([]byte(strconv.FormatBool(r.isAdmin(address))))
}

func (r *Role) isAdmin(address string) bool {
	var admins []*repo.Admin
	r.GetObject(adminRolesKey, &admins)

	for _, admin := range admins {
		if admin.Address == address {
			return true
		}
	}

	return false
}

func (r *Role) GetAdminRoles() *boltvm.Response {
	var admins []*repo.Admin
	r.GetObject(adminRolesKey, &admins)

	ret, err := json.Marshal(admins)
	if err != nil {
		return boltvm.Error(err.Error(), boltvm.Internal)
	}

	return boltvm.Success(ret)
}

func (r *Role) SetAdminRoles(addrs string) *boltvm.Response {
	as := make([]string, 0)
	if err := json.Unmarshal([]byte(addrs), &as); err != nil {
		return boltvm.Error(err.Error(), boltvm.Internal)
	}

	admins := make([]*repo.Admin, 0)
	for _, addr := range as {
		admins = append(admins, &repo.Admin{
			Address: addr,
			Weight:  1,
		})
	}

	r.SetObject(adminRolesKey, admins)
	return boltvm.Success(nil)
}

func (r *Role) GetRoleWeight(address string) *boltvm.Response {
	var admins []*repo.Admin
	r.GetObject(adminRolesKey, &admins)

	for _, admin := range admins {
		if admin.Address == address {
			return boltvm.Success([]byte(strconv.Itoa(int(admin.Weight))))
		}
	}

	return boltvm.Error("the account at this address is not an administrator: "+address, boltvm.BadPermission)
}

// Permission manager
type Permission string

const (
	PermissionAdmin     Permission = "PermissionAdmin"
	PermissionSelfAdmin Permission = "PermissionSelfAdmin"
	PermissionSpecific  Permission = "PermissionSpecific"
)

func (r *Role) CheckPermission(permission string, regulatedId string, regulatorAddr string, specificAddrsData []byte) *boltvm.Response {
	switch permission {
	case string(PermissionAdmin):
		if !r.isAdmin(regulatorAddr) {
			return boltvm.Error("caller is not an admin account: "+regulatorAddr, boltvm.BadPermission)
		}
	case string(PermissionSelfAdmin):
		if regulatorAddr != regulatedId && !r.isAdmin(regulatorAddr) {
			return boltvm.Error("caller is not an admin account or appchain self: "+regulatorAddr, boltvm.BadPermission)
		}
	case string(PermissionSpecific):
		specificAddrs := []string{}
		err := json.Unmarshal(specificAddrsData, &specificAddrs)
		if err != nil {
			return boltvm.Error(err.Error(), boltvm.Internal)
		}
		for _, addr := range specificAddrs {
			if addr == regulatorAddr {
				return boltvm.Success(nil)
			}
		}
		return boltvm.Error("caller is not specific account: "+regulatorAddr, boltvm.BadPermission)
	default:
		return boltvm.Error("unsupport permission: "+permission, boltvm.BadPermission)
	}

	return boltvm.Success(nil)
}
