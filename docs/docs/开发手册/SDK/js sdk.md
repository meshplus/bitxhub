# Js SDK
# 1 前言

此SDK文档面向BitXHub平台的应用开发者，提供BitXHub JS SDK的使用指南。

# 2 接口使用流程示例

为了更好的理解接口的使用，本示例将从初始化Client，部署合约，调用合约和返回值解析这个大致流程作介绍，具体详细接口可参考第三章SDK文档。

## 2.1 基本流程
### 2.1.1 安装并初始化Client

可以通过npm安装JS SDK并引入到JS的项目中
```shell
npm install @meshplus/js-bitxhub-client@1.5.0
```

JS SDK分为几个磨块供用户分开进行调用，分别为：Client, PbType, Config, Transaction, Block, TripleDES以及AES。

用户在调用JS SDK与BitXHub进行交互时，主要是需要使用Client, PbType和Config这三个模块。

用户引入JS SDK的库以后首先需要配置网络地址和接口。

例如：

```javascript
import { Config } from '@meshplus/js-bitxhub-client';

Config.setHosts(["localhost"]);
Config.setPorts(["9091"]);
```

初始化Client，所有操作将通过该对象与BitXHub交互。

例如：

```javascript
import { Client } from '@meshplus/js-bitxhub-client';

let client = new Client(privateKey);
```

### 2.1.2 部署合约

开发者需提供已经编译的`WebAssembly`文件。

例如：

```javascript
import { fs } from 'fs';

let contract = fs.readFileSync("./testdata/example.wasm");
```

通过client部署合约，部署完成后可以获取合约地址`addr`。

例如：

```javascript
let address = await cli.DeployContract(contract) 
```

### 2.1.3 调用合约

调用合约需传入合约地址、合约方法名和对应的参数。

例如：

```javascript
result = cli.InvokeContract(0, address, "a", PbType.pbInt32(1), PbType.pbInt32(2))
//第一个参数指定调用XVM合约还是BVM合约，第二个参数是合约地址， 方法名为a，传参1，传参2
```

### 2.1.4 完整示例

```javascript
import { fs } from 'fs';
import { Client } from '@meshplus/js-bitxhub-client';

let contract = fs.readFileSync("./testdata/example.wasm");

let client = new Client(privateKey);
//部署合约，获取合约地址
let address = await cli.DeployContract(contract);

//调用合约，获取交易回执
result = cli.InvokeContract(1, address, "a", PbType.pbInt32(1), PbType.pbInt32(2));

//打印合约返回数据
console.log(result);
```

## 2.2 应用链管理流程示例

本示例展示应用链管理流程中的注册、审核以及注销操作。

### 2.2.1 应用链注册

调用BVM合约的`Register`方法，向BitXHub注册应用链。

例如：

```javascript
let ret = cli.InvokeContract(0, InterchainContractAddr, "Register", PbType.pbString(validator),
    PbType.pbInt32(0), PbType.pbString(chainType), PbType.pbString(name),
    PbType.pbString(desc), PbType.pbString(version), PbType.pbString(pubKey)
);
```

获取到成功的交易回执后，得到交易回执内容。

例如：

```javascript
{
    "id": "0x5098cc26b0d485145fb8258d2e79c49886cd4662", \\应用链ID
    "name": "税务链",
    "validators": "", 
    "consensus_type": 0,
    "status": 0,
    "chain_type": "hyperchain",
    "desc": "趣链税务链",
    "version": "1.8"
}
```

### 2.2.2 应用链审核

调用BVM合约的`Aduit`方法，向BitXHub审核应用链。

例如：

```javascript
let ret = cli.InvokeContract(0, InterchainContractAddr, "Aduit", PbType.pbString(address),
    PbType.pbInt32(1), PbType.pbString(desc)
);
```

### 2.2.3 应用链注销

调用BVM合约的`DeleteAppchain`方法，向BitXHub注销应用链。

例如：

```javascript
let ret = cli.InvokeContract(0, InterchainContractAddr, "DeleteAppchain", PbType.pbString(address));
```

## 2.3 验证规则使用示例

本示例展示验证规则中的注册、审核操作，以及WebAssembly合约示例。

### 2.3.1 验证规则注册

调用BVM合约的`RegisterRule`方法，向应用链注册验证规则（WebAssembly合约），这里我们需要先注册应用链和部署验证规则合约，然后获取应用链ID和合约地址。

例如：

```javascript
let ret = cli.InvokeContract(0, RoleContractAddr, "RegisterRule", PbType.pbString(chainAddr), PbType.pbString(contractAddr));
```

### 2.3.2 验证规则审核

调用BVM合约的`Aduit`方法，向BitXHub审核验证规则。

例如：

```javascript
let ret = cli.InvokeContract(0, RoleContractAddr, "Aduit", PbType.pbString(chainAddr),
    PbType.pbInt32(1), PbType.pbString(desc)
);
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

## 3.1 交易接口

### 3.1.1 发送交易

用途：调用该接口向中继链发送交易，交易类型包括普通交易、跨链交易和智能合约。

参数：

- `tx`交易实例。

```javascript
function SendTransaction(transaction)
```



### 3.1.2 查询交易回执

用途：调用该接口向BitXHub查询交易回执。

参数：

- `hash`交易哈希。

```javascript
function GetReceipt(hash)
```

### 3.1.3 查询交易

用途：调用该接口向BitXHub查询交易。

参数：

- `hash`交易哈希。

```javascript
function GetTransaction(hash)
```



## 3.2 合约接口

合约类型：

- BVM：BitXHub内置合约。

- XVM：WebAssembly合约。

### 3.2.1 部署合约

用途：调用该接口向BitXHub部署XVM合约，返回合约地址。

参数：

- `ctx`wasm合约编译后的字节数据。


```javascript
function DeployContract(ctx)
```

### 3.2.2 调用合约

用途：该接口向中继链调用合约获取交易回执。

参数：

- `vmType`合约类型：BVM和XVM；

- `address`合约地址；

- `method`合约方法；

- `args`合约方法参数。


```javascript
function InvokeContract(vmType, address, method, ...args)
```

## 3.3 区块接口

### 3.3.1 查询区块

参数：

- `value`区块高度或者区块哈希。
- `type` 查询类型。

```javascript
function GetBlock(type, value)
```



### 3.3.2 批量查询区块

用途：批量查询区块，返回指定块高度范围（start到end）的区块信息。

参数：

- `start`指定范围的起始区块高度。
- `end`指定范围的结束区块高度。

```javascript
function GetBlocks(start, end)
```
