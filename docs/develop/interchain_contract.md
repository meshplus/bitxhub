# 跨链合约编写规范

## 跨链合约设计说明

按照跨链合约的设计，我们需要在有跨链需求的应用链上部署两种合约。一个合约负责对接跨链网关Pier，为跨链管理合约Broker；一个合约负责具体的业务场景，为业务合约。业务合约需要跨链时，要统一将跨链请求提交到Broker合约上，Broker统一和Pier进行交互。一个Broker合约可以负责对接多个业务合约。

同时为了简化Broker的编写，我们设计了Broker合约和业务合约的相应接口。

### Broker 合约接口

```java
public interface Broker {
	// 提供给业务合约注册。注册且审核通过的业务合约才能调用Broker合约的跨链接口
	register(string id) Response

	// 提供给管理员审核已经注册的业务合约
	audit(string id, bool status) Response

	// getInnerMeta 是获取跨链请求相关的Meta信息的接口。以Broker所在的区块链为目的链的一系列跨链请求的序号信息。如果Broker在A链，则可能有多条链和A进行跨链，如B->A:3; C->A:5。返回的map中，key值为来源链ID，value对应该来源链已发送的最新的跨链请求的序号，如{B:3, C:5}。
	getInnerMeta() Response

	// getOuterMeta 是获取跨链请求相关的Meta信息的接口。以Broker所在的区块链为来源链的一系列跨链请求的序号信息。如果以Broker在A链，则A可能和多条链进行跨链，如A->B:3; A->C:5。返回的map中，key值为目的链ID，value对应已发送到该目的链的最新跨链请求的序号，如{B:3, C:5}。
	getOuterMeta() Response

	// getCallbackMeta 是获取跨链请求相关的Meta信息的接口。以Broker所在的区块链为来源链的一系列跨链请求的序号信息。如果Broker在A链，则A可能和多条链进行跨链，如A->B:3; A->C:5；同时由于跨链请求中支持回调操作，即A->B->A为一次完整的跨链操作，我们需要记录回调请求的序号信息，如A->B->:2; A->C—>A:4。返回的map中，key值为目的链ID，value对应到该目的链最新的带回调跨链请求的序号，如{B:2, C:4}。（注意 callbackMeta序号可能和outMeta是不一致的，这是由于由A发出的跨链请求部分是没有回调的）
	getCallbackMeta() Response

	// getInMessage 查询历史跨链请求。查询键值中srcChainID指定来源链，idx指定序号，查询结果为以Broker所在的区块链作为目的链的跨链请求。	
	getInMessage(string srcChainID, uint64 idx) Response

	// getOutMessage 查询历史跨链请求。查询键值中dstChainID指定目的链，idx指定序号，查询结果为以Broker所在的区块链作为来源链的跨链请求。
	getOutMessage(string dstChainID, uint64 idx) Response

	// 提供给业务合约发起跨链资产交换的接口
	InterchainTransferInvoke(string dstChainID, string destAddr, string args) Response

	// 提供给业务合约发起跨链数据交换的接口
	InterchainDataSwapInvoke(string dstChainID, string destAddr, string key) Response

	// 提供给业务合约发起通用的跨链交易的接口
	InterchainInvoke(string dstChainID, string sourceAddr, string destAddr, string func, string args, string callback) Response

	// 提供给跨链网关调用的接口，跨链网关收到跨链充值的请求时候调用
	interchainCharge(string srcChainID, uint64 index, string destAddr, string sender, string receiver, uint64 amount) Response

	// 提供给跨链网关调用的接口，跨链网关收到跨链转账执行结果时调用
	interchainConfirm(string srcChainID, uint64 index, string destAddr, bool status, string sender, uint64 amount) Response

	// 提供给跨链网关调用的接口，跨链网关收到跨链数据交换请求时调用
	interchainGet(string srcChainID, uint64 index, string destAddr, string key) Response

	// 提供给跨链网关调用的接口，跨链网关收到跨链数据交换执行结果时调用
	interchainSet(string srcChainID, uint64 index, string destAddr, string key, string value) Response

	// 提供给合约部署初始化使用
	initialize() Response
}
```

#### 重要接口说明

1. `InterchainInvoke` 接口

改接口是实现通用的跨链调用的接口。接受的参数有：目的链ID，发起跨链交易的业务合约地址或ID，目的链业务合约地址或ID，跨链调用的函数名，该函数的参数，回调函数名。

Broker会记录跨链交易相应的元信息，并对跨链交易进行编号，保证跨链交易有序进行。并且抛出跨链事件，以通知跨链网关跨链交易的产生。

2. `InterchainTransferInvoke` 接口

是对`InterchainInvoke` 接口的封装，专门用于发起一次跨链转账业务。接受参数有：目的链ID，目的链业务合约地址或ID，发起转账账户名，接受转账账户名，转账金额。

3. `InterchainDataSwapInvoke` 接口

是对InterchainInvoke 接口的封装，专门用于发起一次跨链数据交换业务。接受参数有：目的链ID，目的链业务合约地址或ID，Key值。

4. `interchainCharge` 接口

接受跨链转账的接口。由Broker合约根据业务合约地址或ID对业务合约进行充值操作，并记录相应的元信息。接受参数有：来源链ID，交易序号，目的业务合约ID，发起账户名，接收账户名，转账金额。

5. `interchainConfirm` 接口

接受跨链转账的接口。由Broker合约根据业务合约地址或ID对业务合约进行充值操作，并记录相应的元信息。接受参数有：来源链ID，交易序号，目的业务合约ID，跨链转账交易状态，接收账户名，转账金额。

Broker需要根据跨链交易的状态决定是否进行回滚操作，并记录相应元信息。

6. `interchainGet` 接口

接受跨链数据获取的接口。接受参数有：来源链ID，交易序号，目的业务合约ID，Key值。

7. `interchainSet` 接口

接受跨链数据回写的接口。接受参数有：来源链ID，交易序号，目的业务合约ID，Key值，value值。 

### 业务合约接口

业务合约现阶段分为资产类和数据交换类的业务合约，由于资产类的有操作原子性和安全性的考虑，需要的接口实现上比数据交换类的业务合约更复杂。

#### Transfer 合约

```java
public interface Transfer {
	// 发起一笔跨链交易的接口
	transfer(string dstChainID, string destAddr, string sender, string receiver, string amount) Response

	// 提供给Broker合约收到跨链充值所调用的接口
	interchainCharge(string sender, string receiver, uint64 val) Response 

	// 跨链交易失败之后，提供给Broker合约进行回滚的接口
	interchainRollback(string sender, uint64 val) Response 
}
```

#### DataSwapper合约

```java
public interface DataSwapper {
	// 发起一个跨链获取数据交易的接口
	get(string dstChainID, string dstAddr, string key) Response

	// 提供给Broker合约调用，当Broker收到跨链获取数据的请求时取数据的接口
	interchainGet(string key) Response

	// 跨链获取到的数据回写的接口
	interchainSet(string key, string value) Response 
}
```

## 具体实现

对于想要接入到我们的跨链平台中的Fabric区块链，我们已经有提供跨链管理合约Broker和相应的Plugin，你只需要对你的业务合约进行一定的改造便可拥有跨链功能。

**如果是其他应用链，你可以根据我们的设计思路自行开发跨链管理合约以及相应的Plugin。**

现在我们已经有Solidity版本和chaincode版本编写的跨链合约样例实现，具体说明如下：

* [Solidity 跨链合约实现]()
* [chaincode 跨链合约实现]()

如果你需要新的语言编写合约，你可以按照我们的设计思路和参考实现进行进一步的开发。

### 改造业务合约

本章主要说明在Fabric应用链上，如何使用我们提供的跨链管理合约Broker，在你已有业务合约的基础上添加接口，以或得跨链能力。

#### 业务合约Demo

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

##### 发起跨链数据交换的接口

为了方便用户使用，我们在原来获取数据的接口基础增加这个功能：

```go
const (
	channelID            = "mychannel"
	brokerContractName   = "broker"
	interchainInvokeFunc = "InterchainDataSwapInvoke"
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
		b := util.ToChaincodeArgs(interchainInvokeFunc, args[0], args[1], args[2])
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
b := util.ToChaincodeArgs(interchainInvokeFunc, args[0], args[1], args[2])
response := stub.InvokeChaincode(brokerContractName, b, channelID)
```

这两行代码调用了我们的跨链管理合约，只需要提供参数：目的链ID，目的链上业务合约的地址，想要获取数据的`Key`值。

##### 跨链获取的接口

```go
func (s *KVStore) interchainGet(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
	   return shim.Error("incorrect number of arguments")
	}

   value, err := stub.GetState(args[0])
   if err != nil {
      return shim.Error(err.Error())
   }

   return shim.Success(value)
}
```

`interchainGet` 接受参数 `key`，在本合约中查询该`Key`值对应的`value`，并返回。该接口提供给`Broker`合约进行跨链获取数据的调用。

##### 跨链回写的接口

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

`interchainSet` 接受参数 `key`，在本合约中设置`Key`值对应的`value`。该接口提供给`Broker`合约回写跨链获取数据的时候进行调用。

### 总结

经过上面的改造，你的业务合约已经具备跨链获取数据的功能了，完整的代码可以参数[这里](https://github.com/meshplus/pier-client-fabric/tree/master/example)