// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package appchain

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// InterchainSwapABI is the input ABI used to generate the binding from.
const InterchainSwapABI = "[{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_relayers\",\"type\":\"address[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"ethToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"relayToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"burner\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Burn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"ethToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"relayToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"txid\",\"type\":\"string\"}],\"name\":\"Mint\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RELAYER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ethTokenAddr\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"relayTokenAddr\",\"type\":\"address\"}],\"name\":\"addSupportToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"ethTokenAddrs\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"relayTokenAddrs\",\"type\":\"address[]\"}],\"name\":\"addSupportTokens\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"bxh2ethToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"eth2bxhToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getRoleMember\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleMemberCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ethToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"relayToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"_txid\",\"type\":\"string\"},{\"internalType\":\"bytes[]\",\"name\":\"signatures\",\"type\":\"bytes[]\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"mintAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ethTokenAddr\",\"type\":\"address\"}],\"name\":\"removeSupportToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"addrs\",\"type\":\"address[]\"}],\"name\":\"removeSupportTokens\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"name\":\"txUnlocked\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// InterchainSwapFuncSigs maps the 4-byte function signature to its string representation.
var InterchainSwapFuncSigs = map[string]string{
	"a217fddf": "DEFAULT_ADMIN_ROLE()",
	"926d7d7f": "RELAYER_ROLE()",
	"7010584c": "addSupportToken(address,address)",
	"ab1494de": "addSupportTokens(address[],address[])",
	"b8ce670d": "burn(address,uint256,address)",
	"520de93e": "bxh2ethToken(address)",
	"a1090028": "eth2bxhToken(address)",
	"248a9ca3": "getRoleAdmin(bytes32)",
	"9010d07c": "getRoleMember(bytes32,uint256)",
	"ca15c873": "getRoleMemberCount(bytes32)",
	"2f2ff15d": "grantRole(bytes32,address)",
	"91d14854": "hasRole(bytes32,address)",
	"fc3f6916": "mint(address,address,address,address,uint256,string,bytes[])",
	"3c0a25e2": "mintAmount(address,address)",
	"e2769cfa": "removeSupportToken(address)",
	"0daff621": "removeSupportTokens(address[])",
	"36568abe": "renounceRole(bytes32,address)",
	"d547741f": "revokeRole(bytes32,address)",
	"967145af": "txUnlocked(string)",
}

// InterchainSwapBin is the compiled bytecode used for deploying new contracts.
var InterchainSwapBin = "0x60806040523480156200001157600080fd5b5060405162001b9c38038062001b9c8339810160408190526200003491620001c0565b6200004160003362000094565b60005b81518110156200008c57620000836b52454c415945525f524f4c4560a01b8383815181106200006f57fe5b60200260200101516200009460201b60201c565b60010162000044565b50506200029c565b620000a08282620000a4565b5050565b600082815260208181526040909120620000c991839062000b436200011d821b17901c565b15620000a057620000d96200013d565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b600062000134836001600160a01b03841662000141565b90505b92915050565b3390565b60006200014f838362000190565b620001875750815460018181018455600084815260208082209093018490558454848252828601909352604090209190915562000137565b50600062000137565b60009081526001919091016020526040902054151590565b80516001600160a01b03811681146200013757600080fd5b60006020808385031215620001d3578182fd5b82516001600160401b0380821115620001ea578384fd5b818501915085601f830112620001fe578384fd5b8151818111156200020d578485fd5b83810291506200021f84830162000275565b8181528481019084860184860187018a10156200023a578788fd5b8795505b838610156200026857620002538a82620001a8565b8352600195909501949186019186016200023e565b5098975050505050505050565b6040518181016001600160401b03811182821017156200029457600080fd5b604052919050565b6118f080620002ac6000396000f3fe608060405234801561001057600080fd5b50600436106101215760003560e01c8063926d7d7f116100ad578063b8ce670d11610071578063b8ce670d1461024c578063ca15c8731461025f578063d547741f14610272578063e2769cfa14610285578063fc3f69161461029857610121565b8063926d7d7f14610203578063967145af1461020b578063a10900281461021e578063a217fddf14610231578063ab1494de1461023957610121565b80633c0a25e2116100f45780633c0a25e21461018a578063520de93e1461019d5780637010584c146101bd5780639010d07c146101d057806391d14854146101e357610121565b80630daff62114610126578063248a9ca31461013b5780632f2ff15d1461016457806336568abe14610177575b600080fd5b61013961013436600461120c565b6102ab565b005b61014e6101493660046112a8565b6102df565b60405161015b91906114db565b60405180910390f35b6101396101723660046112c0565b6102f4565b6101396101853660046112c0565b610341565b61014e610198366004611093565b610383565b6101b06101ab366004611078565b6103a0565b60405161015b9190611421565b6101396101cb366004611093565b6103bb565b6101b06101de3660046112ef565b61049a565b6101f66101f13660046112c0565b6104bb565b60405161015b91906114d0565b61014e6104d3565b6101f6610219366004611310565b6104e6565b6101b061022c366004611078565b610506565b61014e610521565b610139610247366004611247565b610526565b61013961025a3660046111cb565b610590565b61014e61026d3660046112a8565b6106d5565b6101396102803660046112c0565b6106ec565b610139610293366004611078565b610726565b6101396102a63660046110c7565b6107cb565b60005b81518110156102db576102d38282815181106102c657fe5b6020026020010151610726565b6001016102ae565b5050565b60009081526020819052604090206002015490565b600082815260208190526040902060020154610312906101f1610b58565b6103375760405162461bcd60e51b815260040161032e90611584565b60405180910390fd5b6102db8282610b5c565b610349610b58565b6001600160a01b0316816001600160a01b0316146103795760405162461bcd60e51b815260040161032e906117dc565b6102db8282610bc5565b600360209081526000928352604080842090915290825290205481565b6002602052600090815260409020546001600160a01b031681565b6103c66000336104bb565b6103e25760405162461bcd60e51b815260040161032e90611515565b6001600160a01b03828116600090815260016020526040902054161561041a5760405162461bcd60e51b815260040161032e906117a5565b6001600160a01b0381811660009081526002602052604090205416156104525760405162461bcd60e51b815260040161032e906117a5565b6001600160a01b0391821660008181526001602090815260408083208054969095166001600160a01b0319968716811790955593825260029052919091208054909216179055565b60008281526020819052604081206104b29083610c2e565b90505b92915050565b60008281526020819052604081206104b29083610c3a565b6b52454c415945525f524f4c4560a01b81565b805160208183018101805160048252928201919093012091525460ff1681565b6001602052600090815260409020546001600160a01b031681565b600081565b80518251146105475760405162461bcd60e51b815260040161032e906115d3565b60005b825181101561058b5761058383828151811061056257fe5b602002602001015183838151811061057657fe5b60200260200101516103bb565b60010161054a565b505050565b6001600160a01b038084166000908152600160205260409020548491166105c95760405162461bcd60e51b815260040161032e906116c2565b6001600160a01b03841660009081526003602090815260408083203384529091529020546105f79084610c4f565b6001600160a01b038516600081815260036020908152604080832033845290915290819020929092559051632770a7eb60e21b8152639dc29fac9061064290859087906004016114b7565b600060405180830381600087803b15801561065c57600080fd5b505af1158015610670573d6000803e3d6000fd5b505050506001600160a01b03848116600090815260026020526040908190205490517f5f3dad08c1dd97826b9b06465a3fa5e9695444695f7b522b13a3cae63d20d505926106c79216908790339087908990611435565b60405180910390a150505050565b60008181526020819052604081206104b590610c91565b60008281526020819052604090206002015461070a906101f1610b58565b6103795760405162461bcd60e51b815260040161032e90611726565b6107316000336104bb565b61074d5760405162461bcd60e51b815260040161032e90611515565b6001600160a01b03818116600090815260016020526040902054166107845760405162461bcd60e51b815260040161032e906116f9565b6001600160a01b0390811660008181526001602081815260408084208054909616845260028252832080546001600160a01b03199081169091559390925290528154169055565b6001600160a01b038088166000908152600160205260409020548891166108045760405162461bcd60e51b815260040161032e906116c2565b61081d6b52454c415945525f524f4c4560a01b336104bb565b6108395760405162461bcd60e51b815260040161032e90611776565b8260048160405161084a91906113d4565b9081526040519081900360200190205460ff161561087a5760405162461bcd60e51b815260040161032e9061169d565b6001600160a01b038089166000908152600260209081526040808320548d8516845260019092529091205482169116146108c65760405162461bcd60e51b815260040161032e90611603565b60006108e06b52454c415945525f524f4c4560a01b6106d5565b845190915060036000198301049060028284018101049081111561090657505050610b38565b60008c8c8c8c8c8c6040516020016109239695949392919061136f565b60405160208183030381529060405280519060200120905060005b87518110156109b157600061096e61095584610c9c565b8a848151811061096157fe5b6020026020010151610ccc565b90506109896b52454c415945525f524f4c4560a01b826104bb565b156109a85760008381526005602052604090206109a69082610b43565b505b5060010161093e565b50600081815260056020526040902082906109cb90610c91565b10156109e95760405162461bcd60e51b815260040161032e9061163a565b60016004896040516109fb91906113d4565b9081526040805160209281900383019020805460ff1916931515939093179092556001600160a01b038e8116600090815260038352838120918e16815291522054610a46908a610dab565b600360008e6001600160a01b03166001600160a01b0316815260200190815260200160002060008c6001600160a01b03166001600160a01b03168152602001908152602001600020819055508b6001600160a01b03166340c10f198b8b6040518363ffffffff1660e01b8152600401610ac09291906114b7565b600060405180830381600087803b158015610ada57600080fd5b505af1158015610aee573d6000803e3d6000fd5b505050507f4feaa67f2bfe27f3f037662df125ebe5bbba60fe4cbab27b0fc61b13c44789f88d8d8d8d8d8d604051610b2b96959493929190611468565b60405180910390a1505050505b505050505050505050565b60006104b2836001600160a01b038416610dd0565b3390565b6000828152602081905260409020610b749082610b43565b156102db57610b81610b58565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b6000828152602081905260409020610bdd9082610e1a565b156102db57610bea610b58565b6001600160a01b0316816001600160a01b0316837ff6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b60405160405180910390a45050565b60006104b28383610e2f565b60006104b2836001600160a01b038416610e74565b60006104b283836040518060400160405280601e81526020017f536166654d6174683a207375627472616374696f6e206f766572666c6f770000815250610e8c565b60006104b582610eb8565b600081604051602001610caf91906113f0565b604051602081830303815290604052805190602001209050919050565b60008151604114610cdf575060006104b5565b60208201516040830151606084015160001a7f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0821115610d2557600093505050506104b5565b8060ff16601b14158015610d3d57508060ff16601c14155b15610d4e57600093505050506104b5565b600060018783868660405160008152602001604052604051610d7394939291906114e4565b6020604051602081039080840390855afa158015610d95573d6000803e3d6000fd5b5050604051601f19015198975050505050505050565b6000828201838110156104b25760405162461bcd60e51b815260040161032e90611666565b6000610ddc8383610e74565b610e12575081546001818101845560008481526020808220909301849055845484825282860190935260409020919091556104b5565b5060006104b5565b60006104b2836001600160a01b038416610ebc565b81546000908210610e525760405162461bcd60e51b815260040161032e90611542565b826000018281548110610e6157fe5b9060005260206000200154905092915050565b60009081526001919091016020526040902054151590565b60008184841115610eb05760405162461bcd60e51b815260040161032e9190611502565b505050900390565b5490565b60008181526001830160205260408120548015610f785783546000198083019190810190600090879083908110610eef57fe5b9060005260206000200154905080876000018481548110610f0c57fe5b600091825260208083209091019290925582815260018981019092526040902090840190558654879080610f3c57fe5b600190038181906000526020600020016000905590558660010160008781526020019081526020016000206000905560019450505050506104b5565b60009150506104b5565b80356001600160a01b03811681146104b557600080fd5b600082601f830112610fa9578081fd5b8135610fbc610fb782611852565b61182b565b818152915060208083019084810181840286018201871015610fdd57600080fd5b60005b8481101561100457610ff28883610f82565b84529282019290820190600101610fe0565b505050505092915050565b600082601f83011261101f578081fd5b813567ffffffffffffffff811115611035578182fd5b611048601f8201601f191660200161182b565b915080825283602082850101111561105f57600080fd5b8060208401602084013760009082016020015292915050565b600060208284031215611089578081fd5b6104b28383610f82565b600080604083850312156110a5578081fd5b6110af8484610f82565b91506110be8460208501610f82565b90509250929050565b600080600080600080600060e0888a0312156110e1578283fd5b6110eb8989610f82565b96506110fa8960208a01610f82565b95506111098960408a01610f82565b94506111188960608a01610f82565b93506080880135925060a088013567ffffffffffffffff8082111561113b578384fd5b6111478b838c0161100f565b935060c08a013591508082111561115c578283fd5b508801601f81018a1361116d578182fd5b803561117b610fb782611852565b818152602080820191908401855b848110156111b6576111a18f6020843589010161100f565b84526020938401939190910190600101611189565b50508094505050505092959891949750929550565b6000806000606084860312156111df578283fd5b83356111ea816118a2565b9250602084013591506040840135611201816118a2565b809150509250925092565b60006020828403121561121d578081fd5b813567ffffffffffffffff811115611233578182fd5b61123f84828501610f99565b949350505050565b60008060408385031215611259578182fd5b823567ffffffffffffffff80821115611270578384fd5b61127c86838701610f99565b93506020850135915080821115611291578283fd5b5061129e85828601610f99565b9150509250929050565b6000602082840312156112b9578081fd5b5035919050565b600080604083850312156112d2578182fd5b8235915060208301356112e4816118a2565b809150509250929050565b60008060408385031215611301578182fd5b50508035926020909101359150565b600060208284031215611321578081fd5b813567ffffffffffffffff811115611337578182fd5b61123f8482850161100f565b6000815180845261135b816020860160208601611872565b601f01601f19169290920160200192915050565b60006bffffffffffffffffffffffff19808960601b168352808860601b166014840152808760601b166028840152808660601b16603c8401525083605083015282516113c2816070850160208701611872565b91909101607001979650505050505050565b600082516113e6818460208701611872565b9190910192915050565b7f19457468657265756d205369676e6564204d6573736167653a0a3332000000008152601c810191909152603c0190565b6001600160a01b0391909116815260200190565b6001600160a01b039586168152938516602085015291841660408401529092166060820152608081019190915260a00190565b6001600160a01b03878116825286811660208301528581166040830152841660608201526080810183905260c060a082018190526000906114ab90830184611343565b98975050505050505050565b6001600160a01b03929092168252602082015260400190565b901515815260200190565b90815260200190565b93845260ff9290921660208401526040830152606082015260800190565b6000602082526104b26020830184611343565b60208082526013908201527231b0b63632b91034b9903737ba1030b236b4b760691b604082015260600190565b60208082526022908201527f456e756d657261626c655365743a20696e646578206f7574206f6620626f756e604082015261647360f01b606082015260800190565b6020808252602f908201527f416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e60408201526e0818591b5a5b881d1bc819dc985b9d608a1b606082015260800190565b6020808252601690820152750a8ded6cadc40d8cadccee8d040dcdee840dac2e8c6d60531b604082015260600190565b60208082526017908201527f4275726e3a3a4e6f7420537570706f727420546f6b656e000000000000000000604082015260600190565b6020808252601290820152711cda59db985d1d5c995cc81a5b9d985b1a5960721b604082015260600190565b6020808252601b908201527f536166654d6174683a206164646974696f6e206f766572666c6f770000000000604082015260600190565b6020808252600b908201526a1d1e081d5b9b1bd8dad95960aa1b604082015260600190565b6020808252601f908201527f4d696e74206f72204275726e3a3a4e6f7420537570706f727420546f6b656e00604082015260600190565b602080825260139082015272151bdad95b881b9bdd0814dd5c1c1bdc9d1959606a1b604082015260600190565b60208082526030908201527f416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e60408201526f2061646d696e20746f207265766f6b6560801b606082015260800190565b60208082526015908201527431b0b63632b91034b9903737ba1031b937b9b9b2b960591b604082015260600190565b60208082526017908201527f546f6b656e20616c726561647920537570706f72746564000000000000000000604082015260600190565b6020808252602f908201527f416363657373436f6e74726f6c3a2063616e206f6e6c792072656e6f756e636560408201526e103937b632b9903337b91039b2b63360891b606082015260800190565b60405181810167ffffffffffffffff8111828210171561184a57600080fd5b604052919050565b600067ffffffffffffffff821115611868578081fd5b5060209081020190565b60005b8381101561188d578181015183820152602001611875565b8381111561189c576000848401525b50505050565b6001600160a01b03811681146118b757600080fd5b5056fea2646970667358221220d628ede061498b32a3ab2e8a145c65a4a899f7f95887beb5681963f9b45fc3b264736f6c634300060c0033"

// DeployInterchainSwap deploys a new Ethereum contract, binding an instance of InterchainSwap to it.
func DeployInterchainSwap(auth *bind.TransactOpts, backend bind.ContractBackend, _relayers []common.Address) (common.Address, *types.Transaction, *InterchainSwap, error) {
	parsed, err := abi.JSON(strings.NewReader(InterchainSwapABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(InterchainSwapBin), backend, _relayers)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &InterchainSwap{InterchainSwapCaller: InterchainSwapCaller{contract: contract}, InterchainSwapTransactor: InterchainSwapTransactor{contract: contract}, InterchainSwapFilterer: InterchainSwapFilterer{contract: contract}}, nil
}

// InterchainSwap is an auto generated Go binding around an Ethereum contract.
type InterchainSwap struct {
	InterchainSwapCaller     // Read-only binding to the contract
	InterchainSwapTransactor // Write-only binding to the contract
	InterchainSwapFilterer   // Log filterer for contract events
}

// InterchainSwapCaller is an auto generated read-only Go binding around an Ethereum contract.
type InterchainSwapCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InterchainSwapTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InterchainSwapTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InterchainSwapFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InterchainSwapFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InterchainSwapSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InterchainSwapSession struct {
	Contract     *InterchainSwap   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InterchainSwapCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InterchainSwapCallerSession struct {
	Contract *InterchainSwapCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// InterchainSwapTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InterchainSwapTransactorSession struct {
	Contract     *InterchainSwapTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// InterchainSwapRaw is an auto generated low-level Go binding around an Ethereum contract.
type InterchainSwapRaw struct {
	Contract *InterchainSwap // Generic contract binding to access the raw methods on
}

// InterchainSwapCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InterchainSwapCallerRaw struct {
	Contract *InterchainSwapCaller // Generic read-only contract binding to access the raw methods on
}

// InterchainSwapTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InterchainSwapTransactorRaw struct {
	Contract *InterchainSwapTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInterchainSwap creates a new instance of InterchainSwap, bound to a specific deployed contract.
func NewInterchainSwap(address common.Address, backend bind.ContractBackend) (*InterchainSwap, error) {
	contract, err := bindInterchainSwap(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InterchainSwap{InterchainSwapCaller: InterchainSwapCaller{contract: contract}, InterchainSwapTransactor: InterchainSwapTransactor{contract: contract}, InterchainSwapFilterer: InterchainSwapFilterer{contract: contract}}, nil
}

// NewInterchainSwapCaller creates a new read-only instance of InterchainSwap, bound to a specific deployed contract.
func NewInterchainSwapCaller(address common.Address, caller bind.ContractCaller) (*InterchainSwapCaller, error) {
	contract, err := bindInterchainSwap(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InterchainSwapCaller{contract: contract}, nil
}

// NewInterchainSwapTransactor creates a new write-only instance of InterchainSwap, bound to a specific deployed contract.
func NewInterchainSwapTransactor(address common.Address, transactor bind.ContractTransactor) (*InterchainSwapTransactor, error) {
	contract, err := bindInterchainSwap(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InterchainSwapTransactor{contract: contract}, nil
}

// NewInterchainSwapFilterer creates a new log filterer instance of InterchainSwap, bound to a specific deployed contract.
func NewInterchainSwapFilterer(address common.Address, filterer bind.ContractFilterer) (*InterchainSwapFilterer, error) {
	contract, err := bindInterchainSwap(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InterchainSwapFilterer{contract: contract}, nil
}

// bindInterchainSwap binds a generic wrapper to an already deployed contract.
func bindInterchainSwap(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(InterchainSwapABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InterchainSwap *InterchainSwapRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InterchainSwap.Contract.InterchainSwapCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InterchainSwap *InterchainSwapRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InterchainSwap.Contract.InterchainSwapTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InterchainSwap *InterchainSwapRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InterchainSwap.Contract.InterchainSwapTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InterchainSwap *InterchainSwapCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InterchainSwap.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InterchainSwap *InterchainSwapTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InterchainSwap.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InterchainSwap *InterchainSwapTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InterchainSwap.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_InterchainSwap *InterchainSwapCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_InterchainSwap *InterchainSwapSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _InterchainSwap.Contract.DEFAULTADMINROLE(&_InterchainSwap.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_InterchainSwap *InterchainSwapCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _InterchainSwap.Contract.DEFAULTADMINROLE(&_InterchainSwap.CallOpts)
}

// RELAYERROLE is a free data retrieval call binding the contract method 0x926d7d7f.
//
// Solidity: function RELAYER_ROLE() view returns(bytes32)
func (_InterchainSwap *InterchainSwapCaller) RELAYERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "RELAYER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RELAYERROLE is a free data retrieval call binding the contract method 0x926d7d7f.
//
// Solidity: function RELAYER_ROLE() view returns(bytes32)
func (_InterchainSwap *InterchainSwapSession) RELAYERROLE() ([32]byte, error) {
	return _InterchainSwap.Contract.RELAYERROLE(&_InterchainSwap.CallOpts)
}

// RELAYERROLE is a free data retrieval call binding the contract method 0x926d7d7f.
//
// Solidity: function RELAYER_ROLE() view returns(bytes32)
func (_InterchainSwap *InterchainSwapCallerSession) RELAYERROLE() ([32]byte, error) {
	return _InterchainSwap.Contract.RELAYERROLE(&_InterchainSwap.CallOpts)
}

// Bxh2ethToken is a free data retrieval call binding the contract method 0x520de93e.
//
// Solidity: function bxh2ethToken(address ) view returns(address)
func (_InterchainSwap *InterchainSwapCaller) Bxh2ethToken(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "bxh2ethToken", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bxh2ethToken is a free data retrieval call binding the contract method 0x520de93e.
//
// Solidity: function bxh2ethToken(address ) view returns(address)
func (_InterchainSwap *InterchainSwapSession) Bxh2ethToken(arg0 common.Address) (common.Address, error) {
	return _InterchainSwap.Contract.Bxh2ethToken(&_InterchainSwap.CallOpts, arg0)
}

// Bxh2ethToken is a free data retrieval call binding the contract method 0x520de93e.
//
// Solidity: function bxh2ethToken(address ) view returns(address)
func (_InterchainSwap *InterchainSwapCallerSession) Bxh2ethToken(arg0 common.Address) (common.Address, error) {
	return _InterchainSwap.Contract.Bxh2ethToken(&_InterchainSwap.CallOpts, arg0)
}

// Eth2bxhToken is a free data retrieval call binding the contract method 0xa1090028.
//
// Solidity: function eth2bxhToken(address ) view returns(address)
func (_InterchainSwap *InterchainSwapCaller) Eth2bxhToken(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "eth2bxhToken", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Eth2bxhToken is a free data retrieval call binding the contract method 0xa1090028.
//
// Solidity: function eth2bxhToken(address ) view returns(address)
func (_InterchainSwap *InterchainSwapSession) Eth2bxhToken(arg0 common.Address) (common.Address, error) {
	return _InterchainSwap.Contract.Eth2bxhToken(&_InterchainSwap.CallOpts, arg0)
}

// Eth2bxhToken is a free data retrieval call binding the contract method 0xa1090028.
//
// Solidity: function eth2bxhToken(address ) view returns(address)
func (_InterchainSwap *InterchainSwapCallerSession) Eth2bxhToken(arg0 common.Address) (common.Address, error) {
	return _InterchainSwap.Contract.Eth2bxhToken(&_InterchainSwap.CallOpts, arg0)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_InterchainSwap *InterchainSwapCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_InterchainSwap *InterchainSwapSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _InterchainSwap.Contract.GetRoleAdmin(&_InterchainSwap.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_InterchainSwap *InterchainSwapCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _InterchainSwap.Contract.GetRoleAdmin(&_InterchainSwap.CallOpts, role)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_InterchainSwap *InterchainSwapCaller) GetRoleMember(opts *bind.CallOpts, role [32]byte, index *big.Int) (common.Address, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "getRoleMember", role, index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_InterchainSwap *InterchainSwapSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _InterchainSwap.Contract.GetRoleMember(&_InterchainSwap.CallOpts, role, index)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_InterchainSwap *InterchainSwapCallerSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _InterchainSwap.Contract.GetRoleMember(&_InterchainSwap.CallOpts, role, index)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_InterchainSwap *InterchainSwapCaller) GetRoleMemberCount(opts *bind.CallOpts, role [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "getRoleMemberCount", role)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_InterchainSwap *InterchainSwapSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _InterchainSwap.Contract.GetRoleMemberCount(&_InterchainSwap.CallOpts, role)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_InterchainSwap *InterchainSwapCallerSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _InterchainSwap.Contract.GetRoleMemberCount(&_InterchainSwap.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_InterchainSwap *InterchainSwapCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_InterchainSwap *InterchainSwapSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _InterchainSwap.Contract.HasRole(&_InterchainSwap.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_InterchainSwap *InterchainSwapCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _InterchainSwap.Contract.HasRole(&_InterchainSwap.CallOpts, role, account)
}

// MintAmount is a free data retrieval call binding the contract method 0x3c0a25e2.
//
// Solidity: function mintAmount(address , address ) view returns(uint256)
func (_InterchainSwap *InterchainSwapCaller) MintAmount(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "mintAmount", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MintAmount is a free data retrieval call binding the contract method 0x3c0a25e2.
//
// Solidity: function mintAmount(address , address ) view returns(uint256)
func (_InterchainSwap *InterchainSwapSession) MintAmount(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _InterchainSwap.Contract.MintAmount(&_InterchainSwap.CallOpts, arg0, arg1)
}

// MintAmount is a free data retrieval call binding the contract method 0x3c0a25e2.
//
// Solidity: function mintAmount(address , address ) view returns(uint256)
func (_InterchainSwap *InterchainSwapCallerSession) MintAmount(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _InterchainSwap.Contract.MintAmount(&_InterchainSwap.CallOpts, arg0, arg1)
}

// TxUnlocked is a free data retrieval call binding the contract method 0x967145af.
//
// Solidity: function txUnlocked(string ) view returns(bool)
func (_InterchainSwap *InterchainSwapCaller) TxUnlocked(opts *bind.CallOpts, arg0 string) (bool, error) {
	var out []interface{}
	err := _InterchainSwap.contract.Call(opts, &out, "txUnlocked", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// TxUnlocked is a free data retrieval call binding the contract method 0x967145af.
//
// Solidity: function txUnlocked(string ) view returns(bool)
func (_InterchainSwap *InterchainSwapSession) TxUnlocked(arg0 string) (bool, error) {
	return _InterchainSwap.Contract.TxUnlocked(&_InterchainSwap.CallOpts, arg0)
}

// TxUnlocked is a free data retrieval call binding the contract method 0x967145af.
//
// Solidity: function txUnlocked(string ) view returns(bool)
func (_InterchainSwap *InterchainSwapCallerSession) TxUnlocked(arg0 string) (bool, error) {
	return _InterchainSwap.Contract.TxUnlocked(&_InterchainSwap.CallOpts, arg0)
}

// AddSupportToken is a paid mutator transaction binding the contract method 0x7010584c.
//
// Solidity: function addSupportToken(address ethTokenAddr, address relayTokenAddr) returns()
func (_InterchainSwap *InterchainSwapTransactor) AddSupportToken(opts *bind.TransactOpts, ethTokenAddr common.Address, relayTokenAddr common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "addSupportToken", ethTokenAddr, relayTokenAddr)
}

// AddSupportToken is a paid mutator transaction binding the contract method 0x7010584c.
//
// Solidity: function addSupportToken(address ethTokenAddr, address relayTokenAddr) returns()
func (_InterchainSwap *InterchainSwapSession) AddSupportToken(ethTokenAddr common.Address, relayTokenAddr common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.AddSupportToken(&_InterchainSwap.TransactOpts, ethTokenAddr, relayTokenAddr)
}

// AddSupportToken is a paid mutator transaction binding the contract method 0x7010584c.
//
// Solidity: function addSupportToken(address ethTokenAddr, address relayTokenAddr) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) AddSupportToken(ethTokenAddr common.Address, relayTokenAddr common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.AddSupportToken(&_InterchainSwap.TransactOpts, ethTokenAddr, relayTokenAddr)
}

// AddSupportTokens is a paid mutator transaction binding the contract method 0xab1494de.
//
// Solidity: function addSupportTokens(address[] ethTokenAddrs, address[] relayTokenAddrs) returns()
func (_InterchainSwap *InterchainSwapTransactor) AddSupportTokens(opts *bind.TransactOpts, ethTokenAddrs []common.Address, relayTokenAddrs []common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "addSupportTokens", ethTokenAddrs, relayTokenAddrs)
}

// AddSupportTokens is a paid mutator transaction binding the contract method 0xab1494de.
//
// Solidity: function addSupportTokens(address[] ethTokenAddrs, address[] relayTokenAddrs) returns()
func (_InterchainSwap *InterchainSwapSession) AddSupportTokens(ethTokenAddrs []common.Address, relayTokenAddrs []common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.AddSupportTokens(&_InterchainSwap.TransactOpts, ethTokenAddrs, relayTokenAddrs)
}

// AddSupportTokens is a paid mutator transaction binding the contract method 0xab1494de.
//
// Solidity: function addSupportTokens(address[] ethTokenAddrs, address[] relayTokenAddrs) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) AddSupportTokens(ethTokenAddrs []common.Address, relayTokenAddrs []common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.AddSupportTokens(&_InterchainSwap.TransactOpts, ethTokenAddrs, relayTokenAddrs)
}

// Burn is a paid mutator transaction binding the contract method 0xb8ce670d.
//
// Solidity: function burn(address token, uint256 amount, address recipient) returns()
func (_InterchainSwap *InterchainSwapTransactor) Burn(opts *bind.TransactOpts, token common.Address, amount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "burn", token, amount, recipient)
}

// Burn is a paid mutator transaction binding the contract method 0xb8ce670d.
//
// Solidity: function burn(address token, uint256 amount, address recipient) returns()
func (_InterchainSwap *InterchainSwapSession) Burn(token common.Address, amount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.Burn(&_InterchainSwap.TransactOpts, token, amount, recipient)
}

// Burn is a paid mutator transaction binding the contract method 0xb8ce670d.
//
// Solidity: function burn(address token, uint256 amount, address recipient) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) Burn(token common.Address, amount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.Burn(&_InterchainSwap.TransactOpts, token, amount, recipient)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.GrantRole(&_InterchainSwap.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.GrantRole(&_InterchainSwap.TransactOpts, role, account)
}

// Mint is a paid mutator transaction binding the contract method 0xfc3f6916.
//
// Solidity: function mint(address ethToken, address relayToken, address from, address recipient, uint256 amount, string _txid, bytes[] signatures) returns()
func (_InterchainSwap *InterchainSwapTransactor) Mint(opts *bind.TransactOpts, ethToken common.Address, relayToken common.Address, from common.Address, recipient common.Address, amount *big.Int, _txid string, signatures [][]byte) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "mint", ethToken, relayToken, from, recipient, amount, _txid, signatures)
}

// Mint is a paid mutator transaction binding the contract method 0xfc3f6916.
//
// Solidity: function mint(address ethToken, address relayToken, address from, address recipient, uint256 amount, string _txid, bytes[] signatures) returns()
func (_InterchainSwap *InterchainSwapSession) Mint(ethToken common.Address, relayToken common.Address, from common.Address, recipient common.Address, amount *big.Int, _txid string, signatures [][]byte) (*types.Transaction, error) {
	return _InterchainSwap.Contract.Mint(&_InterchainSwap.TransactOpts, ethToken, relayToken, from, recipient, amount, _txid, signatures)
}

// Mint is a paid mutator transaction binding the contract method 0xfc3f6916.
//
// Solidity: function mint(address ethToken, address relayToken, address from, address recipient, uint256 amount, string _txid, bytes[] signatures) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) Mint(ethToken common.Address, relayToken common.Address, from common.Address, recipient common.Address, amount *big.Int, _txid string, signatures [][]byte) (*types.Transaction, error) {
	return _InterchainSwap.Contract.Mint(&_InterchainSwap.TransactOpts, ethToken, relayToken, from, recipient, amount, _txid, signatures)
}

// RemoveSupportToken is a paid mutator transaction binding the contract method 0xe2769cfa.
//
// Solidity: function removeSupportToken(address ethTokenAddr) returns()
func (_InterchainSwap *InterchainSwapTransactor) RemoveSupportToken(opts *bind.TransactOpts, ethTokenAddr common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "removeSupportToken", ethTokenAddr)
}

// RemoveSupportToken is a paid mutator transaction binding the contract method 0xe2769cfa.
//
// Solidity: function removeSupportToken(address ethTokenAddr) returns()
func (_InterchainSwap *InterchainSwapSession) RemoveSupportToken(ethTokenAddr common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RemoveSupportToken(&_InterchainSwap.TransactOpts, ethTokenAddr)
}

// RemoveSupportToken is a paid mutator transaction binding the contract method 0xe2769cfa.
//
// Solidity: function removeSupportToken(address ethTokenAddr) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) RemoveSupportToken(ethTokenAddr common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RemoveSupportToken(&_InterchainSwap.TransactOpts, ethTokenAddr)
}

// RemoveSupportTokens is a paid mutator transaction binding the contract method 0x0daff621.
//
// Solidity: function removeSupportTokens(address[] addrs) returns()
func (_InterchainSwap *InterchainSwapTransactor) RemoveSupportTokens(opts *bind.TransactOpts, addrs []common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "removeSupportTokens", addrs)
}

// RemoveSupportTokens is a paid mutator transaction binding the contract method 0x0daff621.
//
// Solidity: function removeSupportTokens(address[] addrs) returns()
func (_InterchainSwap *InterchainSwapSession) RemoveSupportTokens(addrs []common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RemoveSupportTokens(&_InterchainSwap.TransactOpts, addrs)
}

// RemoveSupportTokens is a paid mutator transaction binding the contract method 0x0daff621.
//
// Solidity: function removeSupportTokens(address[] addrs) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) RemoveSupportTokens(addrs []common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RemoveSupportTokens(&_InterchainSwap.TransactOpts, addrs)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RenounceRole(&_InterchainSwap.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RenounceRole(&_InterchainSwap.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RevokeRole(&_InterchainSwap.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_InterchainSwap *InterchainSwapTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _InterchainSwap.Contract.RevokeRole(&_InterchainSwap.TransactOpts, role, account)
}

// InterchainSwapBurnIterator is returned from FilterBurn and is used to iterate over the raw logs and unpacked data for Burn events raised by the InterchainSwap contract.
type InterchainSwapBurnIterator struct {
	Event *InterchainSwapBurn // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InterchainSwapBurnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InterchainSwapBurn)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InterchainSwapBurn)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InterchainSwapBurnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InterchainSwapBurnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InterchainSwapBurn represents a Burn event raised by the InterchainSwap contract.
type InterchainSwapBurn struct {
	EthToken   common.Address
	RelayToken common.Address
	Burner     common.Address
	Recipient  common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterBurn is a free log retrieval operation binding the contract event 0x5f3dad08c1dd97826b9b06465a3fa5e9695444695f7b522b13a3cae63d20d505.
//
// Solidity: event Burn(address ethToken, address relayToken, address burner, address recipient, uint256 amount)
func (_InterchainSwap *InterchainSwapFilterer) FilterBurn(opts *bind.FilterOpts) (*InterchainSwapBurnIterator, error) {

	logs, sub, err := _InterchainSwap.contract.FilterLogs(opts, "Burn")
	if err != nil {
		return nil, err
	}
	return &InterchainSwapBurnIterator{contract: _InterchainSwap.contract, event: "Burn", logs: logs, sub: sub}, nil
}

// WatchBurn is a free log subscription operation binding the contract event 0x5f3dad08c1dd97826b9b06465a3fa5e9695444695f7b522b13a3cae63d20d505.
//
// Solidity: event Burn(address ethToken, address relayToken, address burner, address recipient, uint256 amount)
func (_InterchainSwap *InterchainSwapFilterer) WatchBurn(opts *bind.WatchOpts, sink chan<- *InterchainSwapBurn) (event.Subscription, error) {

	logs, sub, err := _InterchainSwap.contract.WatchLogs(opts, "Burn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InterchainSwapBurn)
				if err := _InterchainSwap.contract.UnpackLog(event, "Burn", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBurn is a log parse operation binding the contract event 0x5f3dad08c1dd97826b9b06465a3fa5e9695444695f7b522b13a3cae63d20d505.
//
// Solidity: event Burn(address ethToken, address relayToken, address burner, address recipient, uint256 amount)
func (_InterchainSwap *InterchainSwapFilterer) ParseBurn(log types.Log) (*InterchainSwapBurn, error) {
	event := new(InterchainSwapBurn)
	if err := _InterchainSwap.contract.UnpackLog(event, "Burn", log); err != nil {
		return nil, err
	}
	return event, nil
}

// InterchainSwapMintIterator is returned from FilterMint and is used to iterate over the raw logs and unpacked data for Mint events raised by the InterchainSwap contract.
type InterchainSwapMintIterator struct {
	Event *InterchainSwapMint // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InterchainSwapMintIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InterchainSwapMint)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InterchainSwapMint)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InterchainSwapMintIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InterchainSwapMintIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InterchainSwapMint represents a Mint event raised by the InterchainSwap contract.
type InterchainSwapMint struct {
	EthToken   common.Address
	RelayToken common.Address
	From       common.Address
	Recipient  common.Address
	Amount     *big.Int
	Txid       string
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterMint is a free log retrieval operation binding the contract event 0x4feaa67f2bfe27f3f037662df125ebe5bbba60fe4cbab27b0fc61b13c44789f8.
//
// Solidity: event Mint(address ethToken, address relayToken, address from, address recipient, uint256 amount, string txid)
func (_InterchainSwap *InterchainSwapFilterer) FilterMint(opts *bind.FilterOpts) (*InterchainSwapMintIterator, error) {

	logs, sub, err := _InterchainSwap.contract.FilterLogs(opts, "Mint")
	if err != nil {
		return nil, err
	}
	return &InterchainSwapMintIterator{contract: _InterchainSwap.contract, event: "Mint", logs: logs, sub: sub}, nil
}

// WatchMint is a free log subscription operation binding the contract event 0x4feaa67f2bfe27f3f037662df125ebe5bbba60fe4cbab27b0fc61b13c44789f8.
//
// Solidity: event Mint(address ethToken, address relayToken, address from, address recipient, uint256 amount, string txid)
func (_InterchainSwap *InterchainSwapFilterer) WatchMint(opts *bind.WatchOpts, sink chan<- *InterchainSwapMint) (event.Subscription, error) {

	logs, sub, err := _InterchainSwap.contract.WatchLogs(opts, "Mint")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InterchainSwapMint)
				if err := _InterchainSwap.contract.UnpackLog(event, "Mint", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMint is a log parse operation binding the contract event 0x4feaa67f2bfe27f3f037662df125ebe5bbba60fe4cbab27b0fc61b13c44789f8.
//
// Solidity: event Mint(address ethToken, address relayToken, address from, address recipient, uint256 amount, string txid)
func (_InterchainSwap *InterchainSwapFilterer) ParseMint(log types.Log) (*InterchainSwapMint, error) {
	event := new(InterchainSwapMint)
	if err := _InterchainSwap.contract.UnpackLog(event, "Mint", log); err != nil {
		return nil, err
	}
	return event, nil
}

// InterchainSwapRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the InterchainSwap contract.
type InterchainSwapRoleAdminChangedIterator struct {
	Event *InterchainSwapRoleAdminChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InterchainSwapRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InterchainSwapRoleAdminChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InterchainSwapRoleAdminChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InterchainSwapRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InterchainSwapRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InterchainSwapRoleAdminChanged represents a RoleAdminChanged event raised by the InterchainSwap contract.
type InterchainSwapRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_InterchainSwap *InterchainSwapFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*InterchainSwapRoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _InterchainSwap.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &InterchainSwapRoleAdminChangedIterator{contract: _InterchainSwap.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_InterchainSwap *InterchainSwapFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *InterchainSwapRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _InterchainSwap.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InterchainSwapRoleAdminChanged)
				if err := _InterchainSwap.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_InterchainSwap *InterchainSwapFilterer) ParseRoleAdminChanged(log types.Log) (*InterchainSwapRoleAdminChanged, error) {
	event := new(InterchainSwapRoleAdminChanged)
	if err := _InterchainSwap.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	return event, nil
}

// InterchainSwapRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the InterchainSwap contract.
type InterchainSwapRoleGrantedIterator struct {
	Event *InterchainSwapRoleGranted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InterchainSwapRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InterchainSwapRoleGranted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InterchainSwapRoleGranted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InterchainSwapRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InterchainSwapRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InterchainSwapRoleGranted represents a RoleGranted event raised by the InterchainSwap contract.
type InterchainSwapRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_InterchainSwap *InterchainSwapFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*InterchainSwapRoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _InterchainSwap.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &InterchainSwapRoleGrantedIterator{contract: _InterchainSwap.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_InterchainSwap *InterchainSwapFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *InterchainSwapRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _InterchainSwap.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InterchainSwapRoleGranted)
				if err := _InterchainSwap.contract.UnpackLog(event, "RoleGranted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_InterchainSwap *InterchainSwapFilterer) ParseRoleGranted(log types.Log) (*InterchainSwapRoleGranted, error) {
	event := new(InterchainSwapRoleGranted)
	if err := _InterchainSwap.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	return event, nil
}

// InterchainSwapRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the InterchainSwap contract.
type InterchainSwapRoleRevokedIterator struct {
	Event *InterchainSwapRoleRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InterchainSwapRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InterchainSwapRoleRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InterchainSwapRoleRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InterchainSwapRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InterchainSwapRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InterchainSwapRoleRevoked represents a RoleRevoked event raised by the InterchainSwap contract.
type InterchainSwapRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_InterchainSwap *InterchainSwapFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*InterchainSwapRoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _InterchainSwap.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &InterchainSwapRoleRevokedIterator{contract: _InterchainSwap.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_InterchainSwap *InterchainSwapFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *InterchainSwapRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _InterchainSwap.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InterchainSwapRoleRevoked)
				if err := _InterchainSwap.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_InterchainSwap *InterchainSwapFilterer) ParseRoleRevoked(log types.Log) (*InterchainSwapRoleRevoked, error) {
	event := new(InterchainSwapRoleRevoked)
	if err := _InterchainSwap.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	return event, nil
}
