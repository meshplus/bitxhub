# Java SDK
# 1 前言

此SDK文档面向BitXHub平台的应用开发者，提供BitXHub Java SDK的使用指南。

# 2 接口使用流程示例

## 2.1 基础流程示例

为了更好的理解接口的使用，本示例将从初始化Client，部署合约，调用合约和返回值解析这个大致流程作介绍，具体详细接口可参考第三章SDK文档。

### 2.1.1 初始化Client

使用默认的配置类初始化Grpc Client。

例如：

```java
GrpcClient client = new GrpcClientImpl(Config.defaultConfig());
```

使用定制化的配置类初始化Grpc Client。

例如：

```Java
Config config = Config.builder()
            .host("localhost")
            .port(60011)
            .ecKey(new ECKeyP256())
            .build();
GrpcClient client = new GrpcClientImpl(config);
```

### 2.1.2 部署合约

开发者需提供已经编译的`WebAssembly`文件。

例如：

```java
byte[] contractBytes = IOUtils.toByteArray(new FileInputStream("./example.wasm"));
```

通过client部署合约，部署完成后可以获取合约地址`contractAddress`。

例如：

```java
String contractAddress = client.deployContract(contractBytes);
```

### 2.1.3 调用合约

调用合约需传入合约地址、合约方法名和对应的参数。

例如：

```java
ReceiptOuterClass.Receipt receipt =
client.invokeXVMContract(contractAddress, "a", Types.i32(1), Types.i32(1)); \\方法名为a，传参1，传参2
```

### 2.1.4 返回值解析

得到返回值结果后，获得状态码可以判断是否调用成功，若调用成功，解析返回值可看到调用之后的结果。

例如：

```java
if (receipt.getStatus() == ReceiptOuterClass.Receipt.Status.SUCCESS) {
    log.info(receipt.getRet().toStringUtf8());  
}
```

### 2.1.5 完整示例

```java
//获取wasm字节数组
byte[] contractBytes = IOUtils.toByteArray(
new FileInputStream("./example.wasm"));

//部署合约，获取合约地址
String contractAddress = client.deployContract(contractBytes);

//调用合约，获取交易回执
ReceiptOuterClass.Receipt receipt = client.invokeXVMContract(contractAddress, "a", Types.i32(1), Types.i32(1));

//判断合约调用交易成功与否，打印合约调用数据
if (receipt.getStatus() == ReceiptOuterClass.Receipt.Status.SUCCESS) {
    log.info(receipt.getRet().toStringUtf8());
}
```

## 2.2 应用链管理流程示例

本示例展示应用链管理流程中的注册、审核以及注销操作。

### 2.2.1 应用链注册

调用BVM合约的`Register`方法，向BitXHub注册应用链。

例如：

```java
ArgOuterClass.Arg[] args = Types.toArgArray(
    Types.string(""), //validators
    Types.i32(0), //consensus_type
    Types.string("hyperchain"), //chain_type
    Types.string("税务链"), //name
    Types.string("趣链税务链"), //desc
    Types.string("1.8")); //version
    Types.string("")); //public key
ReceiptOuterClass.Receipt receipt = client.invokeBVMContract(BVMAddr.APPCHAIN_MANAGER_CONTRACT_ADDR, "Register", args);
```

获取到成功的交易回执后，解析交易回执内容。

例如：

```java
{
    "id": "0x5098cc26b0d485145fb8258d2e79c49886cd4662", \\应用链ID
    "name": "税务链",
    "validators": "", 
    "consensus_type": 0,
    "status": 0,
    "chain_type": "hyperchain",
    "desc": "趣链税务链",
    "version": "1.8",
    "public_key": ""
}
```

### 2.2.2 应用链审核

调用BVM合约的`Audit`方法，向BitXHub审核应用链。

例如：

```java
ArgOuterClass.Arg[] adultArgs = Types.toArgArray(
    Types.string(appchainID), //应用链ID
    Types.i32(1), //审核通过
    Types.string("")); //描述信息
ReceiptOuterClass.Receipt adultReceipt = client.invokeBVMContract(BVMAddr.APPCHAIN_MANAGER_CONTRACT_ADDR, "Audit", adultArgs);
```

### 2.2.3 应用链注销

调用BVM合约的`DeleteAppchain`方法，向BitXHub注销应用链。

例如：

```java
ArgOuterClass.Arg[] deleteArgs = Types.toArgArray(
    Types.string(appchainID); //应用链ID
ReceiptOuterClass.Receipt deleteReceipt = client.invokeBVMContract(BVMAddr.APPCHAIN_MANAGER_CONTRACT_ADDR, "DeleteAppchain", deleteArgs);
```

## 2.3 验证规则使用示例

本示例展示验证规则中的注册、审核操作，以及WebAssembly合约示例。

### 2.3.1 验证规则注册

调用BVM合约的`RegisterRule`方法，向应用链注册验证规则（WebAssembly合约），这里我们需要先注册应用链和部署验证规则合约，然后获取应用链ID和合约地址。

例如：

```java
ArgOuterClass.Arg[] ruleArgs = Types.toArgArray(
    Types.string(appchainID),
    Types.string(contractAddress));
ReceiptOuterClass.Receipt ruleReceipt = client.invokeBVMContract(BVMAddr.RULE_MANAGER_CONTRACT_ADDR, "RegisterRule", ruleArgs);
```

### 2.3.2 验证规则审核

调用BVM合约的`Audit`方法，向BitXHub审核验证规则。

例如：

```java
ArgOuterClass.Arg[] adultArgs = Types.toArgArray(
    Types.string(contractAddress), //验证规则的合约地址
    Types.i32(1), //审核通过
    Types.string("")); //描述信息
ReceiptOuterClass.Receipt adultReceipt = client.invokeBVMContract(BVMAddr.RULE_MANAGER_CONTRACT_ADDR, "Audit", adultArgs);
```

### 2.3.3 验证规则示例（WebAssembly合约, Fabric实例）

```rust
extern crate protobuf;
extern crate sha2;

use crate::crypto::ecdsa;
use crate::model::transaction;
use sha2::{Digest, Sha256};

pub fn verify(proof: &[u8], validator: &[u8]) -> bool {
  let cap =
    protobuf::parse_from_bytes::<transaction::ChaincodeActionPayload>(proof).expect("error");

  let cap_act = cap.action.unwrap();
  let endorsers = cap_act.endorsements;
  let mut digest = Sha256::new();
  let mut payload = cap_act.proposal_response_payload.to_owned();
  payload.extend(&endorsers[0].endorser);
  digest.input(&payload);
  let digest_byte = digest.result();

  return ecdsa::verify(
    &endorsers[0].signature,
    &digest_byte,
    &validator,
    ecdsa::EcdsaAlgorithmn::P256,
  );
}
```



# 3 SDK文档

## 3.1 初始化

### 3.1.1 初始化Client

用途：调用该接口获取与BitXHub交互的Client。

```java
GrpcClient client = new GrpcClientImpl(Config.defaultConfig());
```

入参：`Config` 是BitXHub的网络地址, 端口以及密钥的配置。

返回值：与BitXHub交互的`Client`。

### 3.1.2 停止Client

用途：调用该接口将与BitXHub交互的`Client`关闭。

```java
public void stop() throws InterruptedException
```

## 3.2 交易接口

### 3.2.1 发送交易

用途：调用该接口向BitXHub发送交易，交易类型包括普通交易、跨链交易和智能合约交易。

参数：

- `transaction`交易。
- `opts`跨链交易nonce。

```java
public String sendTransaction(TransactionOuterClass.Transaction transaction, TransactOpts opts);
```

用例：

```java
public void sendTransaction() {
    TransactionOuterClass.Transaction unsignedTx = TransactionOuterClass.Transaction.newBuilder()
                .setFrom(ByteString.copyFrom(from))
                .setTo(ByteString.copyFrom(to))
                .setTimestamp(Utils.genTimestamp())
                .setPayload(TransactionOuterClass.TransactionData.newBuilder().setAmount(100000L).build().toByteString())
                .build();

    TransactionOuterClass.Transaction signedTx = SignUtils.sign(unsignedTx, config.getEcKey());
    String txHash = client.sendTransaction(signedTx, null);
}
```

### 3.2.2 查询交易回执

参数：

- `hash`交易哈希。

```java
ReceiptOuterClass.Receipt getReceipt(String hash);
```

### 3.2.3 查询交易

参数：

- `hash`交易哈希。

```java
Broker.GetTransactionResponse getTransaction(String hash);
```

## 3.3 合约接口

合约类型：

- BVM：BitXHub内置合约

- XVM：WebAssembly合约

### 3.3.1 部署合约

用途：调用该接口向BitXHub部署XVM合约。

参数：

- `contract`合约数据。

```java
String deployContract(byte[] contract);
```

### 3.3.2 调用合约

用途：调用该接口向BitXHub调用BVM或者XVM合约。

参数：

- `vmType`合约类型：BVM和XVM。
- `contractAddress`合约地址。
- `method `合约方法；
- `args`合约方法参数。
```java
ReceiptOuterClass.Receipt invokeContract(TransactionOuterClass.TransactionData.VMType vmType, String contractAddress, String method, ArgOuterClass.Arg... args);
```

用例：

```java
public void invokeContract() throws IOException {
    byte[] contractBytes = IOUtils.toByteArray(
            new FileInputStream("./example.wasm"));
    String contractAddress = client.deployContract(contractBytes);

    ReceiptOuterClass.Receipt receipt = client.invokeContract(TransactionOuterClass.TransactionData.VMType.XVM
            , contractAddress, "a", Types.i32(1), Types.i32(1));
}
```

## 3.4 区块接口

### 3.4.1 查询区块

参数：

- `value`区块高度或者区块哈希。
- `type` 查询类型。`type类型`。

```java
BlockOuterClass.Block getBlock(String value, Broker.GetBlockRequest.Type type);
```



### 3.4.2 批量查询区块

用途：批量查询区块，返回指定块高度范围（start到end）的区块信息。

参数：

- `start`指定范围的起始区块高度。
- `end`指定范围的结束区块高度。

```java
Broker.GetBlocksResponse getBlocks(Long start, Long end);
```



### 3.4.3 查询区块Meta

用途：返回当前链的高度和区块哈希。

```java
Chain.ChainMeta getChainMeta();
```



### 3.4.4 查询区块链状态

用途：返回当前区块链共识的状态（正常或者不正常）。

```java
Broker.Response getChainStatus();
```



## 3.5 订阅接口

### 3.5.1 订阅事件

用途：调用该接口向BitXHub发起订阅事件。

参数：

- `streamObserver` 事件通道。
- `type` 事件类型。

用例：
```java
void subscribe(Broker.SubscriptionRequest.Type type, StreamObserver<Broker.Response> observer);
```

```java
public void subscribe() throws InterruptedException {
    CountDownLatch asyncLatch = new CountDownLatch(1);
    StreamObserver<Broker.Response> observer = new StreamObserver<Broker.Response>() {
        @Override
        public void onNext(Broker.Response response) {
            ByteString data = response.getData();
            BlockOuterClass.Block block = null;
            try {
                block = BlockOuterClass.Block.parseFrom(data);
            } catch (InvalidProtocolBufferException e) {
                e.printStackTrace();
            }
        }

        @Override
        public void onError(Throwable throwable) {
            throwable.printStackTrace();
            asyncLatch.countDown();
        }

        @Override
        public void onCompleted() {
            asyncLatch.countDown();
        }
    };

    client.subscribe(Broker.SubscriptionRequest.Type.BLOCK, observer);
    asyncLatch.await();
}
```



## 3.6 其它接口

### 3.6.1 查询节点网络信息

用途：返回当前区块链网络的节点信息。

```java
Broker.Response getNetworkMeta();
```

### 3.6.2 查询账户余额

参数：

- `address`地址。

```java
Broker.Response getAccountBalance(String address);
```