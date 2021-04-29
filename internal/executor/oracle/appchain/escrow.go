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

// EscrowsABI is the input ABI used to generate the binding from.
const EscrowsABI = "[{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"_relayers\",\"type\":\"address[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"ethToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"relayToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Lock\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"ethToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"relayToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"txid\",\"type\":\"string\"}],\"name\":\"Unlock\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RELAYER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ethTokenAddr\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"relayTokenAddr\",\"type\":\"address\"}],\"name\":\"addSupportToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"ethTokenAddrs\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"relayTokenAddrs\",\"type\":\"address[]\"}],\"name\":\"addSupportTokens\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getRoleMember\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleMemberCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"lock\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"lockAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"ethTokenAddr\",\"type\":\"address\"}],\"name\":\"removeSupportToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"addrs\",\"type\":\"address[]\"}],\"name\":\"removeSupportTokens\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"supportToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"name\":\"txUnlocked\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"_txid\",\"type\":\"string\"},{\"internalType\":\"bytes[]\",\"name\":\"signatures\",\"type\":\"bytes[]\"}],\"name\":\"unlock\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// EscrowsFuncSigs maps the 4-byte function signature to its string representation.
var EscrowsFuncSigs = map[string]string{
	"a217fddf": "DEFAULT_ADMIN_ROLE()",
	"926d7d7f": "RELAYER_ROLE()",
	"7010584c": "addSupportToken(address,address)",
	"ab1494de": "addSupportTokens(address[],address[])",
	"248a9ca3": "getRoleAdmin(bytes32)",
	"9010d07c": "getRoleMember(bytes32,uint256)",
	"ca15c873": "getRoleMemberCount(bytes32)",
	"2f2ff15d": "grantRole(bytes32,address)",
	"91d14854": "hasRole(bytes32,address)",
	"4bbc170a": "lock(address,uint256,address)",
	"e2095ab4": "lockAmount(address,address)",
	"e2769cfa": "removeSupportToken(address)",
	"0daff621": "removeSupportTokens(address[])",
	"36568abe": "renounceRole(bytes32,address)",
	"d547741f": "revokeRole(bytes32,address)",
	"2a4f1621": "supportToken(address)",
	"967145af": "txUnlocked(string)",
	"e4242dd5": "unlock(address,address,address,uint256,string,bytes[])",
}

// EscrowsBin is the compiled bytecode used for deploying new contracts.
var EscrowsBin = "0x60806040523480156200001157600080fd5b5060405162001cce38038062001cce8339810160408190526200003491620001c0565b6200004160003362000094565b60005b81518110156200008c57620000836b52454c415945525f524f4c4560a01b8383815181106200006f57fe5b60200260200101516200009460201b60201c565b60010162000044565b50506200029c565b620000a08282620000a4565b5050565b600082815260208181526040909120620000c9918390620009f66200011d821b17901c565b15620000a057620000d96200013d565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b600062000134836001600160a01b03841662000141565b90505b92915050565b3390565b60006200014f838362000190565b620001875750815460018181018455600084815260208082209093018490558454848252828601909352604090209190915562000137565b50600062000137565b60009081526001919091016020526040902054151590565b80516001600160a01b03811681146200013757600080fd5b60006020808385031215620001d3578182fd5b82516001600160401b0380821115620001ea578384fd5b818501915085601f830112620001fe578384fd5b8151818111156200020d578485fd5b83810291506200021f84830162000275565b8181528481019084860184860187018a10156200023a578788fd5b8795505b838610156200026857620002538a82620001a8565b8352600195909501949186019186016200023e565b5098975050505050505050565b6040518181016001600160401b03811182821017156200029457600080fd5b604052919050565b611a2280620002ac6000396000f3fe608060405234801561001057600080fd5b50600436106101165760003560e01c8063926d7d7f116100a2578063ca15c87311610071578063ca15c8731461022e578063d547741f14610241578063e2095ab414610254578063e2769cfa14610267578063e4242dd51461027a57610116565b8063926d7d7f146101f8578063967145af14610200578063a217fddf14610213578063ab1494de1461021b57610116565b806336568abe116100e957806336568abe1461018c5780634bbc170a1461019f5780637010584c146101b25780639010d07c146101c557806391d14854146101d857610116565b80630daff6211461011b578063248a9ca3146101305780632a4f1621146101595780632f2ff15d14610179575b600080fd5b61012e6101293660046112c8565b61028d565b005b61014361013e36600461137c565b6102c1565b60405161015091906115c7565b60405180910390f35b61016c61016736600461114b565b6102d6565b60405161015091906114e9565b61012e610187366004611394565b6102f1565b61012e61019a366004611394565b61033e565b61012e6101ad366004611287565b610380565b61012e6101c0366004611166565b61047c565b61016c6101d33660046113c3565b610509565b6101eb6101e6366004611394565b61052a565b60405161015091906115bc565b610143610542565b6101eb61020e3660046113e4565b610555565b610143610575565b61012e6102293660046112fb565b61057a565b61014361023c36600461137c565b6105e4565b61012e61024f366004611394565b6105fb565b610143610262366004611166565b610635565b61012e61027536600461114b565b610652565b61012e61028836600461119a565b6106d7565b60005b81518110156102bd576102b58282815181106102a857fe5b6020026020010151610652565b600101610290565b5050565b60009081526020819052604090206002015490565b6001602052600090815260409020546001600160a01b031681565b60008281526020819052604090206002015461030f906101e6610a0b565b6103345760405162461bcd60e51b815260040161032b90611670565b60405180910390fd5b6102bd8282610a0f565b610346610a0b565b6001600160a01b0316816001600160a01b0316146103765760405162461bcd60e51b815260040161032b90611912565b6102bd8282610a78565b6001600160a01b038084166000908152600160205260409020548491166103b95760405162461bcd60e51b815260040161032b9061185a565b6001600160a01b03841660009081526002602090815260408083203384529091529020546103e79084610ae1565b6001600160a01b03851660008181526002602090815260408083203380855292529091209290925561041a913086610b06565b6001600160a01b03808516600090815260016020526040908190205490517f4e9dc37847123badcca2493e1da785f59908e20645214cf795e6914bfdbaada29261046e9288929116903390879089906114fd565b60405180910390a150505050565b61048760003361052a565b6104a35760405162461bcd60e51b815260040161032b90611601565b6001600160a01b0382811660009081526001602052604090205416156104db5760405162461bcd60e51b815260040161032b90611823565b6001600160a01b03918216600090815260016020526040902080546001600160a01b03191691909216179055565b60008281526020819052604081206105219083610b64565b90505b92915050565b60008281526020819052604081206105219083610b70565b6b52454c415945525f524f4c4560a01b81565b805160208183018101805160038252928201919093012091525460ff1681565b600081565b805182511461059b5760405162461bcd60e51b815260040161032b906116bf565b60005b82518110156105df576105d78382815181106105b657fe5b60200260200101518383815181106105ca57fe5b602002602001015161047c565b60010161059e565b505050565b600081815260208190526040812061052490610b85565b600082815260208190526040902060020154610619906101e6610a0b565b6103765760405162461bcd60e51b815260040161032b906117a4565b600260209081526000928352604080842090915290825290205481565b61065d60003361052a565b6106795760405162461bcd60e51b815260040161032b90611601565b6001600160a01b03818116600090815260016020526040902054166106b05760405162461bcd60e51b815260040161032b90611777565b6001600160a01b0316600090815260016020526040902080546001600160a01b0319169055565b6001600160a01b038087166000908152600160205260409020548791166107105760405162461bcd60e51b815260040161032b9061185a565b6107296b52454c415945525f524f4c4560a01b3361052a565b6107455760405162461bcd60e51b815260040161032b906117f4565b82600381604051610756919061149c565b9081526040519081900360200190205460ff16156107865760405162461bcd60e51b815260040161032b90611726565b60006107a06b52454c415945525f524f4c4560a01b6105e4565b84519091506003600019830104906002828401810104908111156107c6575050506109ec565b60008b8b8b8b8b6040516020016107e1959493929190611443565b60405160208183030381529060405280519060200120905060005b875181101561086f57600061082c61081384610b90565b8a848151811061081f57fe5b6020026020010151610bc0565b90506108476b52454c415945525f524f4c4560a01b8261052a565b1561086657600083815260046020526040902061086490826109f6565b505b506001016107fc565b506000818152600460205260409020829061088990610b85565b10156108a75760405162461bcd60e51b815260040161032b9061174b565b60016003896040516108b9919061149c565b9081526040805160209281900383019020805460ff1916931515939093179092556001600160a01b038e8116600090815260028352838120918e16815291522054610904908a610c9f565b600260008e6001600160a01b03166001600160a01b0316815260200190815260200160002060008c6001600160a01b03166001600160a01b031681526020019081526020016000208190555061096e8a8a8e6001600160a01b0316610ce19092919063ffffffff16565b7f5800596ed55e41c52e6e763640e29ff0d4c09c47408a512c4fb0974a452c1f0c8c600160008f6001600160a01b03166001600160a01b0316815260200190815260200160002060009054906101000a90046001600160a01b03168d8d8d8d6040516109df96959493929190611530565b60405180910390a1505050505b5050505050505050565b6000610521836001600160a01b038416610d00565b3390565b6000828152602081905260409020610a2790826109f6565b156102bd57610a34610a0b565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b6000828152602081905260409020610a909082610d4a565b156102bd57610a9d610a0b565b6001600160a01b0316816001600160a01b0316837ff6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b60405160405180910390a45050565b6000828201838110156105215760405162461bcd60e51b815260040161032b906116ef565b610b5e846323b872dd60e01b858585604051602401610b279392919061157f565b60408051601f198184030181529190526020810180516001600160e01b03166001600160e01b031990931692909217909152610d5f565b50505050565b60006105218383610dee565b6000610521836001600160a01b038416610e33565b600061052482610e4b565b600081604051602001610ba391906114b8565b604051602081830303815290604052805190602001209050919050565b60008151604114610bd357506000610524565b60208201516040830151606084015160001a7f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0821115610c195760009350505050610524565b8060ff16601b14158015610c3157508060ff16601c14155b15610c425760009350505050610524565b600060018783868660405160008152602001604052604051610c6794939291906115d0565b6020604051602081039080840390855afa158015610c89573d6000803e3d6000fd5b5050604051601f19015198975050505050505050565b600061052183836040518060400160405280601e81526020017f536166654d6174683a207375627472616374696f6e206f766572666c6f770000815250610e4f565b6105df8363a9059cbb60e01b8484604051602401610b279291906115a3565b6000610d0c8383610e33565b610d4257508154600181810184556000848152602080822090930184905584548482528286019093526040902091909155610524565b506000610524565b6000610521836001600160a01b038416610e7b565b6060610db4826040518060400160405280602081526020017f5361666545524332303a206c6f772d6c6576656c2063616c6c206661696c6564815250856001600160a01b0316610f419092919063ffffffff16565b8051909150156105df5780806020019051810190610dd2919061135c565b6105df5760405162461bcd60e51b815260040161032b906118c8565b81546000908210610e115760405162461bcd60e51b815260040161032b9061162e565b826000018281548110610e2057fe5b9060005260206000200154905092915050565b60009081526001919091016020526040902054151590565b5490565b60008184841115610e735760405162461bcd60e51b815260040161032b91906115ee565b505050900390565b60008181526001830160205260408120548015610f375783546000198083019190810190600090879083908110610eae57fe5b9060005260206000200154905080876000018481548110610ecb57fe5b600091825260208083209091019290925582815260018981019092526040902090840190558654879080610efb57fe5b60019003818190600052602060002001600090559055866001016000878152602001908152602001600020600090556001945050505050610524565b6000915050610524565b6060610f508484600085610f58565b949350505050565b6060610f638561101c565b610f7f5760405162461bcd60e51b815260040161032b90611891565b60006060866001600160a01b03168587604051610f9c919061149c565b60006040518083038185875af1925050503d8060008114610fd9576040519150601f19603f3d011682016040523d82523d6000602084013e610fde565b606091505b50915091508115610ff2579150610f509050565b8051156110025780518082602001fd5b8360405162461bcd60e51b815260040161032b91906115ee565b6000813f7fc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470818114801590610f50575050151592915050565b80356001600160a01b038116811461052457600080fd5b600082601f83011261107c578081fd5b813561108f61108a82611988565b611961565b8181529150602080830190848101818402860182018710156110b057600080fd5b60005b848110156110d7576110c58883611055565b845292820192908201906001016110b3565b505050505092915050565b600082601f8301126110f2578081fd5b813567ffffffffffffffff811115611108578182fd5b61111b601f8201601f1916602001611961565b915080825283602082850101111561113257600080fd5b8060208401602084013760009082016020015292915050565b60006020828403121561115c578081fd5b6105218383611055565b60008060408385031215611178578081fd5b6111828484611055565b91506111918460208501611055565b90509250929050565b60008060008060008060c087890312156111b2578182fd5b6111bc8888611055565b955060206111cc89828a01611055565b95506111db8960408a01611055565b945060608801359350608088013567ffffffffffffffff808211156111fe578485fd5b61120a8b838c016110e2565b945060a08a013591508082111561121f578384fd5b508801601f81018a13611230578283fd5b803561123e61108a82611988565b81815283810190838501865b84811015611273576112618f8884358901016110e2565b8452928601929086019060010161124a565b505080955050505050509295509295509295565b60008060006060848603121561129b578283fd5b83356112a6816119d4565b92506020840135915060408401356112bd816119d4565b809150509250925092565b6000602082840312156112d9578081fd5b813567ffffffffffffffff8111156112ef578182fd5b610f508482850161106c565b6000806040838503121561130d578182fd5b823567ffffffffffffffff80821115611324578384fd5b6113308683870161106c565b93506020850135915080821115611345578283fd5b506113528582860161106c565b9150509250929050565b60006020828403121561136d578081fd5b81518015158114610521578182fd5b60006020828403121561138d578081fd5b5035919050565b600080604083850312156113a6578182fd5b8235915060208301356113b8816119d4565b809150509250929050565b600080604083850312156113d5578182fd5b50508035926020909101359150565b6000602082840312156113f5578081fd5b813567ffffffffffffffff81111561140b578182fd5b610f50848285016110e2565b6000815180845261142f8160208601602086016119a8565b601f01601f19169290920160200192915050565b60006bffffffffffffffffffffffff19808860601b168352808760601b166014840152808660601b1660288401525083603c830152825161148b81605c8501602087016119a8565b91909101605c019695505050505050565b600082516114ae8184602087016119a8565b9190910192915050565b7f19457468657265756d205369676e6564204d6573736167653a0a3332000000008152601c810191909152603c0190565b6001600160a01b0391909116815260200190565b6001600160a01b039586168152938516602085015291841660408401529092166060820152608081019190915260a00190565b6001600160a01b03878116825286811660208301528581166040830152841660608201526080810183905260c060a0820181905260009061157390830184611417565b98975050505050505050565b6001600160a01b039384168152919092166020820152604081019190915260600190565b6001600160a01b03929092168252602082015260400190565b901515815260200190565b90815260200190565b93845260ff9290921660208401526040830152606082015260800190565b6000602082526105216020830184611417565b60208082526013908201527231b0b63632b91034b9903737ba1030b236b4b760691b604082015260600190565b60208082526022908201527f456e756d657261626c655365743a20696e646578206f7574206f6620626f756e604082015261647360f01b606082015260800190565b6020808252602f908201527f416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e60408201526e0818591b5a5b881d1bc819dc985b9d608a1b606082015260800190565b6020808252601690820152750a8ded6cadc40d8cadccee8d040dcdee840dac2e8c6d60531b604082015260600190565b6020808252601b908201527f536166654d6174683a206164646974696f6e206f766572666c6f770000000000604082015260600190565b6020808252600b908201526a1d1e081d5b9b1bd8dad95960aa1b604082015260600190565b6020808252601290820152711cda59db985d1d5c995cc81a5b9d985a5b1960721b604082015260600190565b602080825260139082015272151bdad95b881b9bdd0814dd5c1c1bdc9d1959606a1b604082015260600190565b60208082526030908201527f416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e60408201526f2061646d696e20746f207265766f6b6560801b606082015260800190565b60208082526015908201527431b0b63632b91034b9903737ba1031b937b9b9b2b960591b604082015260600190565b60208082526017908201527f546f6b656e20616c726561647920537570706f72746564000000000000000000604082015260600190565b60208082526017908201527f4c6f636b3a3a4e6f7420537570706f727420546f6b656e000000000000000000604082015260600190565b6020808252601d908201527f416464726573733a2063616c6c20746f206e6f6e2d636f6e7472616374000000604082015260600190565b6020808252602a908201527f5361666545524332303a204552433230206f7065726174696f6e20646964206e6040820152691bdd081cdd58d8d9595960b21b606082015260800190565b6020808252602f908201527f416363657373436f6e74726f6c3a2063616e206f6e6c792072656e6f756e636560408201526e103937b632b9903337b91039b2b63360891b606082015260800190565b60405181810167ffffffffffffffff8111828210171561198057600080fd5b604052919050565b600067ffffffffffffffff82111561199e578081fd5b5060209081020190565b60005b838110156119c35781810151838201526020016119ab565b83811115610b5e5750506000910152565b6001600160a01b03811681146119e957600080fd5b5056fea2646970667358221220ae159d9971200769811ad876cc9bf81087951357be6506a2d0e799c8f4c61f0064736f6c634300060c0033"

// DeployEscrows deploys a new Ethereum contract, binding an instance of Escrows to it.
func DeployEscrows(auth *bind.TransactOpts, backend bind.ContractBackend, _relayers []common.Address) (common.Address, *types.Transaction, *Escrows, error) {
	parsed, err := abi.JSON(strings.NewReader(EscrowsABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(EscrowsBin), backend, _relayers)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Escrows{EscrowsCaller: EscrowsCaller{contract: contract}, EscrowsTransactor: EscrowsTransactor{contract: contract}, EscrowsFilterer: EscrowsFilterer{contract: contract}}, nil
}

// Escrows is an auto generated Go binding around an Ethereum contract.
type Escrows struct {
	EscrowsCaller     // Read-only binding to the contract
	EscrowsTransactor // Write-only binding to the contract
	EscrowsFilterer   // Log filterer for contract events
}

// EscrowsCaller is an auto generated read-only Go binding around an Ethereum contract.
type EscrowsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EscrowsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EscrowsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EscrowsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EscrowsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EscrowsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EscrowsSession struct {
	Contract     *Escrows          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// EscrowsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EscrowsCallerSession struct {
	Contract *EscrowsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// EscrowsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EscrowsTransactorSession struct {
	Contract     *EscrowsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// EscrowsRaw is an auto generated low-level Go binding around an Ethereum contract.
type EscrowsRaw struct {
	Contract *Escrows // Generic contract binding to access the raw methods on
}

// EscrowsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EscrowsCallerRaw struct {
	Contract *EscrowsCaller // Generic read-only contract binding to access the raw methods on
}

// EscrowsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EscrowsTransactorRaw struct {
	Contract *EscrowsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEscrows creates a new instance of Escrows, bound to a specific deployed contract.
func NewEscrows(address common.Address, backend bind.ContractBackend) (*Escrows, error) {
	contract, err := bindEscrows(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Escrows{EscrowsCaller: EscrowsCaller{contract: contract}, EscrowsTransactor: EscrowsTransactor{contract: contract}, EscrowsFilterer: EscrowsFilterer{contract: contract}}, nil
}

// NewEscrowsCaller creates a new read-only instance of Escrows, bound to a specific deployed contract.
func NewEscrowsCaller(address common.Address, caller bind.ContractCaller) (*EscrowsCaller, error) {
	contract, err := bindEscrows(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EscrowsCaller{contract: contract}, nil
}

// NewEscrowsTransactor creates a new write-only instance of Escrows, bound to a specific deployed contract.
func NewEscrowsTransactor(address common.Address, transactor bind.ContractTransactor) (*EscrowsTransactor, error) {
	contract, err := bindEscrows(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EscrowsTransactor{contract: contract}, nil
}

// NewEscrowsFilterer creates a new log filterer instance of Escrows, bound to a specific deployed contract.
func NewEscrowsFilterer(address common.Address, filterer bind.ContractFilterer) (*EscrowsFilterer, error) {
	contract, err := bindEscrows(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EscrowsFilterer{contract: contract}, nil
}

// bindEscrows binds a generic wrapper to an already deployed contract.
func bindEscrows(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(EscrowsABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Escrows *EscrowsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Escrows.Contract.EscrowsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Escrows *EscrowsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Escrows.Contract.EscrowsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Escrows *EscrowsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Escrows.Contract.EscrowsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Escrows *EscrowsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Escrows.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Escrows *EscrowsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Escrows.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Escrows *EscrowsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Escrows.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Escrows *EscrowsCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Escrows *EscrowsSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Escrows.Contract.DEFAULTADMINROLE(&_Escrows.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Escrows *EscrowsCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Escrows.Contract.DEFAULTADMINROLE(&_Escrows.CallOpts)
}

// RELAYERROLE is a free data retrieval call binding the contract method 0x926d7d7f.
//
// Solidity: function RELAYER_ROLE() view returns(bytes32)
func (_Escrows *EscrowsCaller) RELAYERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "RELAYER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RELAYERROLE is a free data retrieval call binding the contract method 0x926d7d7f.
//
// Solidity: function RELAYER_ROLE() view returns(bytes32)
func (_Escrows *EscrowsSession) RELAYERROLE() ([32]byte, error) {
	return _Escrows.Contract.RELAYERROLE(&_Escrows.CallOpts)
}

// RELAYERROLE is a free data retrieval call binding the contract method 0x926d7d7f.
//
// Solidity: function RELAYER_ROLE() view returns(bytes32)
func (_Escrows *EscrowsCallerSession) RELAYERROLE() ([32]byte, error) {
	return _Escrows.Contract.RELAYERROLE(&_Escrows.CallOpts)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Escrows *EscrowsCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Escrows *EscrowsSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Escrows.Contract.GetRoleAdmin(&_Escrows.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Escrows *EscrowsCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Escrows.Contract.GetRoleAdmin(&_Escrows.CallOpts, role)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_Escrows *EscrowsCaller) GetRoleMember(opts *bind.CallOpts, role [32]byte, index *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "getRoleMember", role, index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_Escrows *EscrowsSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _Escrows.Contract.GetRoleMember(&_Escrows.CallOpts, role, index)
}

// GetRoleMember is a free data retrieval call binding the contract method 0x9010d07c.
//
// Solidity: function getRoleMember(bytes32 role, uint256 index) view returns(address)
func (_Escrows *EscrowsCallerSession) GetRoleMember(role [32]byte, index *big.Int) (common.Address, error) {
	return _Escrows.Contract.GetRoleMember(&_Escrows.CallOpts, role, index)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_Escrows *EscrowsCaller) GetRoleMemberCount(opts *bind.CallOpts, role [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "getRoleMemberCount", role)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_Escrows *EscrowsSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _Escrows.Contract.GetRoleMemberCount(&_Escrows.CallOpts, role)
}

// GetRoleMemberCount is a free data retrieval call binding the contract method 0xca15c873.
//
// Solidity: function getRoleMemberCount(bytes32 role) view returns(uint256)
func (_Escrows *EscrowsCallerSession) GetRoleMemberCount(role [32]byte) (*big.Int, error) {
	return _Escrows.Contract.GetRoleMemberCount(&_Escrows.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Escrows *EscrowsCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Escrows *EscrowsSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Escrows.Contract.HasRole(&_Escrows.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Escrows *EscrowsCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Escrows.Contract.HasRole(&_Escrows.CallOpts, role, account)
}

// LockAmount is a free data retrieval call binding the contract method 0xe2095ab4.
//
// Solidity: function lockAmount(address , address ) view returns(uint256)
func (_Escrows *EscrowsCaller) LockAmount(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "lockAmount", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LockAmount is a free data retrieval call binding the contract method 0xe2095ab4.
//
// Solidity: function lockAmount(address , address ) view returns(uint256)
func (_Escrows *EscrowsSession) LockAmount(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _Escrows.Contract.LockAmount(&_Escrows.CallOpts, arg0, arg1)
}

// LockAmount is a free data retrieval call binding the contract method 0xe2095ab4.
//
// Solidity: function lockAmount(address , address ) view returns(uint256)
func (_Escrows *EscrowsCallerSession) LockAmount(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _Escrows.Contract.LockAmount(&_Escrows.CallOpts, arg0, arg1)
}

// SupportToken is a free data retrieval call binding the contract method 0x2a4f1621.
//
// Solidity: function supportToken(address ) view returns(address)
func (_Escrows *EscrowsCaller) SupportToken(opts *bind.CallOpts, arg0 common.Address) (common.Address, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "supportToken", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SupportToken is a free data retrieval call binding the contract method 0x2a4f1621.
//
// Solidity: function supportToken(address ) view returns(address)
func (_Escrows *EscrowsSession) SupportToken(arg0 common.Address) (common.Address, error) {
	return _Escrows.Contract.SupportToken(&_Escrows.CallOpts, arg0)
}

// SupportToken is a free data retrieval call binding the contract method 0x2a4f1621.
//
// Solidity: function supportToken(address ) view returns(address)
func (_Escrows *EscrowsCallerSession) SupportToken(arg0 common.Address) (common.Address, error) {
	return _Escrows.Contract.SupportToken(&_Escrows.CallOpts, arg0)
}

// TxUnlocked is a free data retrieval call binding the contract method 0x967145af.
//
// Solidity: function txUnlocked(string ) view returns(bool)
func (_Escrows *EscrowsCaller) TxUnlocked(opts *bind.CallOpts, arg0 string) (bool, error) {
	var out []interface{}
	err := _Escrows.contract.Call(opts, &out, "txUnlocked", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// TxUnlocked is a free data retrieval call binding the contract method 0x967145af.
//
// Solidity: function txUnlocked(string ) view returns(bool)
func (_Escrows *EscrowsSession) TxUnlocked(arg0 string) (bool, error) {
	return _Escrows.Contract.TxUnlocked(&_Escrows.CallOpts, arg0)
}

// TxUnlocked is a free data retrieval call binding the contract method 0x967145af.
//
// Solidity: function txUnlocked(string ) view returns(bool)
func (_Escrows *EscrowsCallerSession) TxUnlocked(arg0 string) (bool, error) {
	return _Escrows.Contract.TxUnlocked(&_Escrows.CallOpts, arg0)
}

// AddSupportToken is a paid mutator transaction binding the contract method 0x7010584c.
//
// Solidity: function addSupportToken(address ethTokenAddr, address relayTokenAddr) returns()
func (_Escrows *EscrowsTransactor) AddSupportToken(opts *bind.TransactOpts, ethTokenAddr common.Address, relayTokenAddr common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "addSupportToken", ethTokenAddr, relayTokenAddr)
}

// AddSupportToken is a paid mutator transaction binding the contract method 0x7010584c.
//
// Solidity: function addSupportToken(address ethTokenAddr, address relayTokenAddr) returns()
func (_Escrows *EscrowsSession) AddSupportToken(ethTokenAddr common.Address, relayTokenAddr common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.AddSupportToken(&_Escrows.TransactOpts, ethTokenAddr, relayTokenAddr)
}

// AddSupportToken is a paid mutator transaction binding the contract method 0x7010584c.
//
// Solidity: function addSupportToken(address ethTokenAddr, address relayTokenAddr) returns()
func (_Escrows *EscrowsTransactorSession) AddSupportToken(ethTokenAddr common.Address, relayTokenAddr common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.AddSupportToken(&_Escrows.TransactOpts, ethTokenAddr, relayTokenAddr)
}

// AddSupportTokens is a paid mutator transaction binding the contract method 0xab1494de.
//
// Solidity: function addSupportTokens(address[] ethTokenAddrs, address[] relayTokenAddrs) returns()
func (_Escrows *EscrowsTransactor) AddSupportTokens(opts *bind.TransactOpts, ethTokenAddrs []common.Address, relayTokenAddrs []common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "addSupportTokens", ethTokenAddrs, relayTokenAddrs)
}

// AddSupportTokens is a paid mutator transaction binding the contract method 0xab1494de.
//
// Solidity: function addSupportTokens(address[] ethTokenAddrs, address[] relayTokenAddrs) returns()
func (_Escrows *EscrowsSession) AddSupportTokens(ethTokenAddrs []common.Address, relayTokenAddrs []common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.AddSupportTokens(&_Escrows.TransactOpts, ethTokenAddrs, relayTokenAddrs)
}

// AddSupportTokens is a paid mutator transaction binding the contract method 0xab1494de.
//
// Solidity: function addSupportTokens(address[] ethTokenAddrs, address[] relayTokenAddrs) returns()
func (_Escrows *EscrowsTransactorSession) AddSupportTokens(ethTokenAddrs []common.Address, relayTokenAddrs []common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.AddSupportTokens(&_Escrows.TransactOpts, ethTokenAddrs, relayTokenAddrs)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.GrantRole(&_Escrows.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.GrantRole(&_Escrows.TransactOpts, role, account)
}

// Lock is a paid mutator transaction binding the contract method 0x4bbc170a.
//
// Solidity: function lock(address token, uint256 amount, address recipient) returns()
func (_Escrows *EscrowsTransactor) Lock(opts *bind.TransactOpts, token common.Address, amount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "lock", token, amount, recipient)
}

// Lock is a paid mutator transaction binding the contract method 0x4bbc170a.
//
// Solidity: function lock(address token, uint256 amount, address recipient) returns()
func (_Escrows *EscrowsSession) Lock(token common.Address, amount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.Lock(&_Escrows.TransactOpts, token, amount, recipient)
}

// Lock is a paid mutator transaction binding the contract method 0x4bbc170a.
//
// Solidity: function lock(address token, uint256 amount, address recipient) returns()
func (_Escrows *EscrowsTransactorSession) Lock(token common.Address, amount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.Lock(&_Escrows.TransactOpts, token, amount, recipient)
}

// RemoveSupportToken is a paid mutator transaction binding the contract method 0xe2769cfa.
//
// Solidity: function removeSupportToken(address ethTokenAddr) returns()
func (_Escrows *EscrowsTransactor) RemoveSupportToken(opts *bind.TransactOpts, ethTokenAddr common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "removeSupportToken", ethTokenAddr)
}

// RemoveSupportToken is a paid mutator transaction binding the contract method 0xe2769cfa.
//
// Solidity: function removeSupportToken(address ethTokenAddr) returns()
func (_Escrows *EscrowsSession) RemoveSupportToken(ethTokenAddr common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RemoveSupportToken(&_Escrows.TransactOpts, ethTokenAddr)
}

// RemoveSupportToken is a paid mutator transaction binding the contract method 0xe2769cfa.
//
// Solidity: function removeSupportToken(address ethTokenAddr) returns()
func (_Escrows *EscrowsTransactorSession) RemoveSupportToken(ethTokenAddr common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RemoveSupportToken(&_Escrows.TransactOpts, ethTokenAddr)
}

// RemoveSupportTokens is a paid mutator transaction binding the contract method 0x0daff621.
//
// Solidity: function removeSupportTokens(address[] addrs) returns()
func (_Escrows *EscrowsTransactor) RemoveSupportTokens(opts *bind.TransactOpts, addrs []common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "removeSupportTokens", addrs)
}

// RemoveSupportTokens is a paid mutator transaction binding the contract method 0x0daff621.
//
// Solidity: function removeSupportTokens(address[] addrs) returns()
func (_Escrows *EscrowsSession) RemoveSupportTokens(addrs []common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RemoveSupportTokens(&_Escrows.TransactOpts, addrs)
}

// RemoveSupportTokens is a paid mutator transaction binding the contract method 0x0daff621.
//
// Solidity: function removeSupportTokens(address[] addrs) returns()
func (_Escrows *EscrowsTransactorSession) RemoveSupportTokens(addrs []common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RemoveSupportTokens(&_Escrows.TransactOpts, addrs)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RenounceRole(&_Escrows.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RenounceRole(&_Escrows.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RevokeRole(&_Escrows.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Escrows *EscrowsTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Escrows.Contract.RevokeRole(&_Escrows.TransactOpts, role, account)
}

// Unlock is a paid mutator transaction binding the contract method 0xe4242dd5.
//
// Solidity: function unlock(address token, address from, address recipient, uint256 amount, string _txid, bytes[] signatures) returns()
func (_Escrows *EscrowsTransactor) Unlock(opts *bind.TransactOpts, token common.Address, from common.Address, recipient common.Address, amount *big.Int, _txid string, signatures [][]byte) (*types.Transaction, error) {
	return _Escrows.contract.Transact(opts, "unlock", token, from, recipient, amount, _txid, signatures)
}

// Unlock is a paid mutator transaction binding the contract method 0xe4242dd5.
//
// Solidity: function unlock(address token, address from, address recipient, uint256 amount, string _txid, bytes[] signatures) returns()
func (_Escrows *EscrowsSession) Unlock(token common.Address, from common.Address, recipient common.Address, amount *big.Int, _txid string, signatures [][]byte) (*types.Transaction, error) {
	return _Escrows.Contract.Unlock(&_Escrows.TransactOpts, token, from, recipient, amount, _txid, signatures)
}

// Unlock is a paid mutator transaction binding the contract method 0xe4242dd5.
//
// Solidity: function unlock(address token, address from, address recipient, uint256 amount, string _txid, bytes[] signatures) returns()
func (_Escrows *EscrowsTransactorSession) Unlock(token common.Address, from common.Address, recipient common.Address, amount *big.Int, _txid string, signatures [][]byte) (*types.Transaction, error) {
	return _Escrows.Contract.Unlock(&_Escrows.TransactOpts, token, from, recipient, amount, _txid, signatures)
}

// EscrowsLockIterator is returned from FilterLock and is used to iterate over the raw logs and unpacked data for Lock events raised by the Escrows contract.
type EscrowsLockIterator struct {
	Event *EscrowsLock // Event containing the contract specifics and raw log

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
func (it *EscrowsLockIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EscrowsLock)
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
		it.Event = new(EscrowsLock)
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
func (it *EscrowsLockIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EscrowsLockIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EscrowsLock represents a Lock event raised by the Escrows contract.
type EscrowsLock struct {
	EthToken   common.Address
	RelayToken common.Address
	Locker     common.Address
	Recipient  common.Address
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterLock is a free log retrieval operation binding the contract event 0x4e9dc37847123badcca2493e1da785f59908e20645214cf795e6914bfdbaada2.
//
// Solidity: event Lock(address ethToken, address relayToken, address locker, address recipient, uint256 amount)
func (_Escrows *EscrowsFilterer) FilterLock(opts *bind.FilterOpts) (*EscrowsLockIterator, error) {

	logs, sub, err := _Escrows.contract.FilterLogs(opts, "Lock")
	if err != nil {
		return nil, err
	}
	return &EscrowsLockIterator{contract: _Escrows.contract, event: "Lock", logs: logs, sub: sub}, nil
}

// WatchLock is a free log subscription operation binding the contract event 0x4e9dc37847123badcca2493e1da785f59908e20645214cf795e6914bfdbaada2.
//
// Solidity: event Lock(address ethToken, address relayToken, address locker, address recipient, uint256 amount)
func (_Escrows *EscrowsFilterer) WatchLock(opts *bind.WatchOpts, sink chan<- *EscrowsLock) (event.Subscription, error) {

	logs, sub, err := _Escrows.contract.WatchLogs(opts, "Lock")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EscrowsLock)
				if err := _Escrows.contract.UnpackLog(event, "Lock", log); err != nil {
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

// ParseLock is a log parse operation binding the contract event 0x4e9dc37847123badcca2493e1da785f59908e20645214cf795e6914bfdbaada2.
//
// Solidity: event Lock(address ethToken, address relayToken, address locker, address recipient, uint256 amount)
func (_Escrows *EscrowsFilterer) ParseLock(log types.Log) (*EscrowsLock, error) {
	event := new(EscrowsLock)
	if err := _Escrows.contract.UnpackLog(event, "Lock", log); err != nil {
		return nil, err
	}
	return event, nil
}

// EscrowsRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the Escrows contract.
type EscrowsRoleAdminChangedIterator struct {
	Event *EscrowsRoleAdminChanged // Event containing the contract specifics and raw log

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
func (it *EscrowsRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EscrowsRoleAdminChanged)
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
		it.Event = new(EscrowsRoleAdminChanged)
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
func (it *EscrowsRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EscrowsRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EscrowsRoleAdminChanged represents a RoleAdminChanged event raised by the Escrows contract.
type EscrowsRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Escrows *EscrowsFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*EscrowsRoleAdminChangedIterator, error) {

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

	logs, sub, err := _Escrows.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &EscrowsRoleAdminChangedIterator{contract: _Escrows.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Escrows *EscrowsFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *EscrowsRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _Escrows.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EscrowsRoleAdminChanged)
				if err := _Escrows.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
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
func (_Escrows *EscrowsFilterer) ParseRoleAdminChanged(log types.Log) (*EscrowsRoleAdminChanged, error) {
	event := new(EscrowsRoleAdminChanged)
	if err := _Escrows.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	return event, nil
}

// EscrowsRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the Escrows contract.
type EscrowsRoleGrantedIterator struct {
	Event *EscrowsRoleGranted // Event containing the contract specifics and raw log

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
func (it *EscrowsRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EscrowsRoleGranted)
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
		it.Event = new(EscrowsRoleGranted)
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
func (it *EscrowsRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EscrowsRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EscrowsRoleGranted represents a RoleGranted event raised by the Escrows contract.
type EscrowsRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Escrows *EscrowsFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*EscrowsRoleGrantedIterator, error) {

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

	logs, sub, err := _Escrows.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &EscrowsRoleGrantedIterator{contract: _Escrows.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Escrows *EscrowsFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *EscrowsRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _Escrows.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EscrowsRoleGranted)
				if err := _Escrows.contract.UnpackLog(event, "RoleGranted", log); err != nil {
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
func (_Escrows *EscrowsFilterer) ParseRoleGranted(log types.Log) (*EscrowsRoleGranted, error) {
	event := new(EscrowsRoleGranted)
	if err := _Escrows.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	return event, nil
}

// EscrowsRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the Escrows contract.
type EscrowsRoleRevokedIterator struct {
	Event *EscrowsRoleRevoked // Event containing the contract specifics and raw log

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
func (it *EscrowsRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EscrowsRoleRevoked)
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
		it.Event = new(EscrowsRoleRevoked)
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
func (it *EscrowsRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EscrowsRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EscrowsRoleRevoked represents a RoleRevoked event raised by the Escrows contract.
type EscrowsRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Escrows *EscrowsFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*EscrowsRoleRevokedIterator, error) {

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

	logs, sub, err := _Escrows.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &EscrowsRoleRevokedIterator{contract: _Escrows.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Escrows *EscrowsFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *EscrowsRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _Escrows.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EscrowsRoleRevoked)
				if err := _Escrows.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
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
func (_Escrows *EscrowsFilterer) ParseRoleRevoked(log types.Log) (*EscrowsRoleRevoked, error) {
	event := new(EscrowsRoleRevoked)
	if err := _Escrows.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	return event, nil
}

// EscrowsUnlockIterator is returned from FilterUnlock and is used to iterate over the raw logs and unpacked data for Unlock events raised by the Escrows contract.
type EscrowsUnlockIterator struct {
	Event *EscrowsUnlock // Event containing the contract specifics and raw log

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
func (it *EscrowsUnlockIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EscrowsUnlock)
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
		it.Event = new(EscrowsUnlock)
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
func (it *EscrowsUnlockIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EscrowsUnlockIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EscrowsUnlock represents a Unlock event raised by the Escrows contract.
type EscrowsUnlock struct {
	EthToken   common.Address
	RelayToken common.Address
	From       common.Address
	Recipient  common.Address
	Amount     *big.Int
	Txid       string
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterUnlock is a free log retrieval operation binding the contract event 0x5800596ed55e41c52e6e763640e29ff0d4c09c47408a512c4fb0974a452c1f0c.
//
// Solidity: event Unlock(address ethToken, address relayToken, address from, address recipient, uint256 amount, string txid)
func (_Escrows *EscrowsFilterer) FilterUnlock(opts *bind.FilterOpts) (*EscrowsUnlockIterator, error) {

	logs, sub, err := _Escrows.contract.FilterLogs(opts, "Unlock")
	if err != nil {
		return nil, err
	}
	return &EscrowsUnlockIterator{contract: _Escrows.contract, event: "Unlock", logs: logs, sub: sub}, nil
}

// WatchUnlock is a free log subscription operation binding the contract event 0x5800596ed55e41c52e6e763640e29ff0d4c09c47408a512c4fb0974a452c1f0c.
//
// Solidity: event Unlock(address ethToken, address relayToken, address from, address recipient, uint256 amount, string txid)
func (_Escrows *EscrowsFilterer) WatchUnlock(opts *bind.WatchOpts, sink chan<- *EscrowsUnlock) (event.Subscription, error) {

	logs, sub, err := _Escrows.contract.WatchLogs(opts, "Unlock")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EscrowsUnlock)
				if err := _Escrows.contract.UnpackLog(event, "Unlock", log); err != nil {
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

// ParseUnlock is a log parse operation binding the contract event 0x5800596ed55e41c52e6e763640e29ff0d4c09c47408a512c4fb0974a452c1f0c.
//
// Solidity: event Unlock(address ethToken, address relayToken, address from, address recipient, uint256 amount, string txid)
func (_Escrows *EscrowsFilterer) ParseUnlock(log types.Log) (*EscrowsUnlock, error) {
	event := new(EscrowsUnlock)
	if err := _Escrows.contract.UnpackLog(event, "Unlock", log); err != nil {
		return nil, err
	}
	return event, nil
}
