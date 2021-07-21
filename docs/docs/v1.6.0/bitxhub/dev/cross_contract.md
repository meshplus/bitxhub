# 跨链合约

按照跨链合约的设计，我们需要在有跨链需求的应用链上部署两种合约。一个合约负责对接跨链网关Pier，为跨链管理合约Broker；一个合约负责具体的业务场景，为业务合约。业务合约需要跨链时，要统一将跨链请求提交到Broker合约上，Broker统一和Pier进行交互。一个Broker合约可以负责对接多个业务合约。

跨链接入方无需对broker合约进行修改，直接部署使用即可；同时为了简化业务合约的编写，我们设计了业务合约的相应接口。

以下以以太坊上的solidity合约为例。

## Broker 合约接口

```solidity
  // 提供给业务合约注册。注册且审核通过的业务合约才能调用Broker合约的跨链接口
  function register(string addr) public

  // 提供给管理员审核已经注册的业务合约
  function audit(string addr, bool status) public returns(bool)

  // getInnerMeta 是获取跨链请求相关的Meta信息的接口。以Broker所在的区块链为目的链的一系列跨链请求的序号信息。
  // 如果Broker在A链，则可能有多条链和A进行跨链，如B->A:3; C->A:5。
  // 返回的map中，key值为来源链ID，value对应该来源链已发送的最新的跨链请求的序号，如{B:3, C:5}。
  function getInnerMeta() public view returns(address[] memory, uint64[] memory)

  // getOuterMeta 是获取跨链请求相关的Meta信息的接口。以Broker所在的区块链为来源链的一系列跨链请求的序号信息。
  // 如果以Broker在A链，则A可能和多条链进行跨链，如A->B:3; A->C:5。
  // 返回的map中，key值为目的链ID，value对应已发送到该目的链的最新跨链请求的序号，如{B:3, C:5}。
  function getOuterMeta() public view returns(address[] memory, uint64[] memory)

  // getCallbackMeta 是获取跨链请求相关的Meta信息的接口。以Broker所在的区块链为来源链的一系列跨链请求的序号信息。
  // 如果Broker在A链，则A可能和多条链进行跨链，如A->B:3; A->C:5；同时由于跨链请求中支持回调操作，即A->B->A为一次完整的跨链操作，
  // 我们需要记录回调请求的序号信息，如A->B->:2; A->C—>A:4。返回的map中，key值为目的链ID，value对应到该目的链最新的带回调跨链请求的序号，
  // 如{B:2, C:4}。（注意 callbackMeta序号可能和outMeta是不一致的，这是由于由A发出的跨链请求部分是没有回调的）
  function getCallbackMeta() public view returns(address[] memory, uint64[] memory)

  // getInMessage 查询历史跨链请求所在的区块高度。查询键值中srcChainID指定来源链，idx指定序号，查询结果为以Broker所在的区块链作为目的链的跨链请求所在的区块高度。
  function getInMessage(string srcChainID, uint64 idx) public view returns (uint)

  // getOutMessage 查询历史跨链请求所在的区块高度。查询键值中dstChainID指定目的链，idx指定序号，查询结果为以Broker所在的区块链作为来源链的跨链请求所在的区块高度。
  function getOutMessage(string dstChainID, uint64 idx) public view returns (uint)

  // 提供给跨链网关调用的接口，跨链网关收到跨链请求时会调用该接口。
  function invokeInterchain(address srcChainID, uint64 index, address destAddr, bool req, bytes calldata bizCallData) payable external
  	
  // 提供给跨链网关调用的接口，当跨链网关收到无效当跨链请求时会调用该接口。
  function invokeIndexUpdateWithError(address srcChainID, uint64 index, bool req, string memory err) public

  // 提供给业务合约发起通用的跨链交易的接口。
  function emitInterchainEvent(address destChainID, string memory destAddr, string memory funcs, string memory args, string memory argscb, string memory argsrb) public onlyWhiteList

  // 提供给合约部署初始化使用
  function initialize() public
```

### 重要接口说明

- `emitInterchainEvent` 

该接口是业务合约发起通用的跨链调用的接口。接收的参数有：目的链ID，目的链业务合约地址或ID，调用的函数名、回调函数名、回滚函数名，调用函数的参数，回调函数的参数，回滚函数的参数。

Broker会记录跨链交易相应的元信息，对跨链交易进行编号，保证跨链交易有序进行, 并且抛出跨链事件，以通知跨链网关跨链交易的产生。

- `invokeInterchain` 

该接口是跨链网关对业务合约进行跨链调用或回调/回滚的接口。 接收参数有：来源链ID，交易序号，目的业务合约ID，是否是跨链请求，业务合约调用方法和参数的封装数据。

跨链网关对要调用的目的合约的方法和参数进行封装，通过该接口实现对不同目的合约的灵活调用，并返回目的合约的调用函数的返回值。

## 业务合约接口

业务合约现阶段分为资产类和数据交换类的业务合约，由于资产类的有操作原子性和安全性的考虑，需要的接口实现上比数据交换类的业务合约更复杂。

### Transfer 合约

```solidity
  // 发起一笔跨链交易的接口
  function transfer(string dstChainID, string destAddr, string sender, string receiver, string amount) public

  // 提供给Broker合约收到跨链充值所调用的接口
  function interchainCharge(string sender, string receiver, uint64 val) public onlyBroker returns(bool)

  // 跨链交易失败之后，提供给Broker合约进行回滚的接口
  function interchainRollback(string sender, uint64 val) public onlyBroker

  // 获取transfer合约中某个账户的余额
  function getBalance(string id) public view returns(uint64)

  // 在transfer合约中给某个账户设置一定的余额
  function setBalance(string id, uint64 amount) public
}
```

### DataSwapper合约

```solidity
  // 发起一个跨链获取数据交易的接口
  function get(string dstChainID, string dstAddr, string key) public

  // 提供给Broker合约调用，当Broker收到跨链获取数据的请求时取数据的接口
  function interchainGet(string key) public onlyBroker returns(bool, string memory)

  // 跨链获取到的数据回写的接口
  function interchainSet(string key, string value) public onlyBroker
```

## 具体实现

对于想要接入到我们的跨链平台中的Fabric区块链，我们已经有提供跨链管理合约Broker和相应的Plugin，你只需要对你的业务合约进行一定的改造便可拥有跨链功能。

**如果是其他应用链，你可以根据我们的设计思路自行开发跨链管理合约以及相应的Plugin。**

现在我们已经有Solidity版本和chaincode版本编写的跨链合约样例实现，具体说明如下：

- [__Solidity 跨链合约实现__](https://github.com/meshplus/pier-client-ethereum/tree/master/example)

- [__Chaincode 跨链合约实现__](https://github.com/meshplus/pier-client-fabric/tree/master/example)

如果你需要新的语言编写合约，你可以按照我们的设计思路和参考实现进行进一步的开发。

现在我们支持Hyperchain EVM合约、以太坊私链Solidity合约、BCOS EVM合约以及Fabric Chaincode合约。

## Hyperchain、以太坊、BCOS上的EVM合约

本节主要说明在支持EVM合约的应用链上，如何使用我们提供的跨链管理合约Broker，在你已有的Solidity业务合约的基础上添加接口，以获得跨链能力。

当然不同的区块链可能在以太坊的EVM上做了一些二次开发和新增功能，请根据具体区块链的文档相应修改代码。

### 业务合约Demo

假设你已经有了一个简单的KV存储的业务合约，代码如下：

```solidity
pragma solidity >=0.5.7;

contract DataSwapper {
    mapping(string => string) dataM; // map for accounts

    // 数据交换类的业务合约
    function getData(string memory key) public returns(string memory) {
        return dataM[key];
    }

    function setData(string memory key, string memory value) public {
        dataM[key] = value;
    }
}

```

现在你想在这个合约的基础上增加一个跨链获取数据的功能，如果使用我们的跨链管理合约提供的接口，很简单的增加几个接口就可以了。

### 发起跨链数据交换的接口

```solidity
contract DataSwapper {
    // broker合约地址
	address BrokerAddr = 0x2346f3BA3F0B6676aa711595daB8A27d0317DB57;
    Broker broker = Broker(BrokerAddr);

	...

	function get(address destChainID, string memory destAddr, string memory key) public {
        broker.emitInterchainEvent(destChainID, destAddr, "interchainGet,interchainSet,", key, key, "");
	}
}

contract Broker {
    function emitInterchainEvent(address destChainID, string memory destAddr, string memory funcs, string memory args, string memory argscb, string memory argsrb) public;
}
```

其中Broker的地址和该业务合约需要使用到的接口需要在业务合约中声明，然后直接调用该接口发起跨链交易。

### 跨链获取的接口

```solidity
contract DataSwapper {
    
    ...

    modifier onlyBroker {
            require(msg.sender == BrokerAddr, "Invoker are not the Broker");
            _;
    }
    
    function interchainGet(string memory key) public onlyBroker returns(bool, string memory) {
            return (true, dataM[key]);
    }
}
```
我们规定跨链调用的接口的第一个返回值类型必须是bool类型，它用来表示跨链调用是否成功。
其中onlyBroker是进行跨链权限控制的修饰器。该接口和下面的跨链回写接口均是提供给Broker合约进行调用，也是其他应用链发来的跨链交易执行时需要调用的接口。

### 跨链回写的接口

```solidity
contract DataSwapper {

    ...

    function interchainSet(string memory key, string memory value) public onlyBroker {
        setData(key, value);
    }

    ...

}

```

## Fabric

本节主要说明在Fabric应用链上，如何使用我们提供的跨链管理合约Broker，在你已有业务合约的基础上添加接口，以获得跨链能力。

### 业务合约Demo

假设你已经有了一个简单的KV存储的业务合约，代码如下：

```go
type KVStore struct{}

func (s *KVStore) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (s *KVStore) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	function, args := stub.GetFunctionAndParameters()

	fmt.Printf("invoke: %s\n", function)
	switch function {
	case "get":
		return s.get(stub, args)
	case "set":
		return s.set(stub, args)
	default:
		return shim.Error("invalid function: " + function + ", args: " + strings.Join(args, ","))
	}
}

func (s *KVStore) get(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// args[0]: key
	value, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(value)
}

// get is business function which will invoke the to,tid,id
func (s *KVStore) set(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("incorrect number of arguments")
	}

	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func main() {
	err := shim.Start(new(KVStore))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
```

现在你想在这个合约的基础上增加一个跨链获取数据的功能，如果使用我们的跨链管理合约提供的接口，很简单的增加几个接口就可以了。

### 发起跨链数据交换的接口

为了方便用户使用，我们在原来获取数据的接口基础上增加这个功能：

```go
const (
	channelID               = "mychannel"
	brokerContractName      = "broker"
    emitInterchainEventFunc = "EmitInterchainEvent"
)

func (s *KVStore) get(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	switch len(args) {
	case 1:
		// args[0]: key
		value, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(value)
	case 3:
		// args[0]: destination appchain id
		// args[1]: destination contract address
		// args[2]: key
        b := util.ToChaincodeArgs(emitInterchainEventFunc, args[0], args[1], "interchainGet", args[2], "interchainSet", args[2], "", "")
		response := stub.InvokeChaincode(brokerContractName, b, channelID)

		if response.Status != shim.OK {
			return shim.Error(fmt.Errorf("invoke broker chaincode %s error: %s", brokerContractName, response.Message).Error())
		}

		return shim.Success(nil)
	default:
		return shim.Error("incorrect number of arguments")
	}
}
```

由于我们的跨链管理合约一旦部署之后，chaincode name和所在的channel和跨链接口都是不变的，所以在业务变量中直接使用常量指定Broker合约的相关信息。

```go
b := util.ToChaincodeArgs(emitInterchainEventFunc, args[0], args[1], "interchainGet", args[2], "interchainSet", args[2], "", "")
response := stub.InvokeChaincode(brokerContractName, b, channelID)
```

这两行代码调用了我们的跨链管理合约，只需要提供参数：目的链ID，目的链上业务合约的地址，跨链调用函数，跨链调用函数参数，回调函数，回调函数参数，最后两个参数为回滚函数和回滚函数参数，因为该场景下即使目的链执行跨链交易失败，来源链也无需回滚，因此无需提供回滚信息。

### 跨链获取的接口

```go
func (s *KVStore) interchainGet(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
	   return shim.Error("incorrect number of arguments")
	}

   value, err := stub.GetState(args[0])
   if err != nil {
      return shim.Error(err.Error())
   }

   return shim.Success(value)
}
```

`interchainGet` 接收参数 `key`，在本合约中查询该`Key`值对应的`value`，并返回。该接口提供给`Broker`合约进行跨链获取数据的调用。

### 跨链回写的接口

```go
func (s *KVStore) interchainSet(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
	   return shim.Error("incorrect number of arguments")
	}

	err := stub.PutState(args[0], []byte(args[1]))
	if err != nil {
	   return shim.Error(err.Error())
	}

	return shim.Success(nil)
}
```

`interchainSet` 接收参数 `key`，在本合约中设置`Key`值对应的`value`。该接口提供给`Broker`合约回写跨链获取数据的时候进行调用。



## 总结

经过上面的改造，你的业务合约已经具备跨链获取数据的功能了，完整的代码可以参考[__这里__](https://github.com/meshplus/pier-client-fabric/tree/master/example)

