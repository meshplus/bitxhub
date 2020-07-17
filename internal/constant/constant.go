package constant

import "github.com/meshplus/bitxhub-kit/types"

type BoltContractAddress string

const (
	InterchainContractAddr     BoltContractAddress = "0x000000000000000000000000000000000000000a"
	StoreContractAddr          BoltContractAddress = "0x000000000000000000000000000000000000000b"
	RuleManagerContractAddr    BoltContractAddress = "0x000000000000000000000000000000000000000c"
	RoleContractAddr           BoltContractAddress = "0x000000000000000000000000000000000000000d"
	AppchainMgrContractAddr    BoltContractAddress = "0x000000000000000000000000000000000000000e"
	TransactionMgrContractAddr BoltContractAddress = "0x000000000000000000000000000000000000000f"
)

func (addr BoltContractAddress) Address() types.Address {
	return types.String2Address(string(addr))
}

func (addr BoltContractAddress) String() string {
	return string(addr)
}
