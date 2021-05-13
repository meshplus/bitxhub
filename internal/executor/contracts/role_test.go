package contracts

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

func TestRole_GetRole(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()
	mockStub.EXPECT().Caller().Return(admins[0].Address)

	im := &Role{mockStub}

	res := im.GetRole()
	assert.True(t, res.Ok)
	assert.Equal(t, "admin", string(res.Result))

	mockStub.EXPECT().Caller().Return(types.NewAddress([]byte{2}).String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error(""))

	res = im.GetRole()
	assert.True(t, res.Ok)
	assert.Equal(t, "none", string(res.Result))

	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))

	res = im.GetRole()
	assert.True(t, res.Ok)
	assert.Equal(t, "appchain_admin", string(res.Result))
}

func TestRole_IsAdmin(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.IsAdmin(admins[0].Address)
	assert.True(t, res.Ok)
	assert.Equal(t, "true", string(res.Result))

	res = im.IsAdmin(types.NewAddress([]byte{2}).String())
	assert.True(t, res.Ok)
	assert.Equal(t, "false", string(res.Result))
}

func TestRole_GetAdminRoles(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
		&repo.Admin{
			Address: "0x79a1215469FaB6f9c63c1816b45183AD3624bE34",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.GetAdminRoles()
	assert.True(t, res.Ok)

	var as []*repo.Admin
	err := json.Unmarshal(res.Result, &as)
	assert.Nil(t, err)
	assert.Equal(t, len(admins), len(as))
	for i, admin := range admins {
		assert.Equal(t, admin.Address, as[i].Address)
	}
}

func TestRole_SetAdminRoles(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addrs := []string{types.NewAddress([]byte{0}).String(), types.NewAddress([]byte{1}).String()}
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()

	im := &Role{mockStub}

	data, err := json.Marshal(addrs)
	assert.Nil(t, err)

	res := im.SetAdminRoles(string(data))
	assert.True(t, res.Ok)
}

func TestRole_GetRoleWeight(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.GetRoleWeight(admins[0].Address)
	assert.True(t, res.Ok)
	w, err := strconv.Atoi(string(res.Result))
	assert.Nil(t, err)
	assert.Equal(t, admins[0].Weight, uint64(w))

	res = im.GetRoleWeight("")
	assert.False(t, res.Ok)
}

func TestRole_CheckPermission(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.CheckPermission(string(PermissionAdmin), "", admins[0].Address, nil)
	assert.True(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionAdmin), "", types.NewAddress([]byte{2}).String(), nil)
	assert.False(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionSelfAdmin), "", admins[0].Address, nil)
	assert.True(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionSelfAdmin), "", types.NewAddress([]byte{2}).String(), nil)
	assert.False(t, res.Ok, string(res.Result))

	addrData, err := json.Marshal([]string{admins[0].Address})
	assert.Nil(t, err)
	res = im.CheckPermission(string(PermissionSpecific), "", admins[0].Address, addrData)
	assert.True(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionSpecific), "", "", addrData)
	assert.False(t, res.Ok, string(res.Result))
}
