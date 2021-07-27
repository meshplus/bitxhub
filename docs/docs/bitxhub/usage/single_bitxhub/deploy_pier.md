# 跨链网关部署

跨链网关Pier能够支持业务所在区块链（以下简称 应用链）便捷、快速的接入到跨链平台BitXHub中来，从而实现和其他业务区块链的跨链操作。跨链网关的部署需要提前确定应用链类型（对应不同的插件和配置），也需要提前在对应的应用链上部署跨链合约，为了符合用户的部署流程和提升操作体验，我们将跨链网关的部署依次分为在应用链上部署跨链合约、获取和修改Pier部署文件、注册应用链、部署验证规则和节点程序启动这五个步骤。下面分别以Ethereum和Fabric为例进行介绍。

## Ethereum

### 1. 在应用链上部署跨链合约

目前中继链和跨链网关已经支持Fabric、Ethereum、Bcos、Cita和Hyperchain四种应用链接入并完成跨链交易，如果您有兴趣，也可以参与开发适配另外种类应用链的插件和合约。对于不同的应用链，一般都有自己的客户端调用工具，用来部署和调用链上的合约，这里简单说明如何获取已支持应用链的跨链合约及部署合约的注意事项。

**注意：在此操作之前，您需要确认已经部署或可接入的Ethereum应用链**， 在Ethereum上部署跨链合约的过程和部署其它合约没有区别，只是合约名称和代码文件需要替换。在Ethereum上部署合约的工具有很多，您可以使[Remix](https://remix.ethereum.org/)进行合约的编译和部署，这里关键的是跨链合约的获取。

1. 下载pier-client-ethereum源码

   ```
   git clone https://github.com/meshplus/pier-client-ethereum.git && git checkout ${VERSION}
   ```

2. 需要部署的合约文件就在example目录下，broker.sol是跨链管理合约，transfer.sol和data_swapper.sol是示例业务合约，需要首先部署broker合约。

3. 部署broker、transfer和data_swapper合约的过程不再赘述，需要特别说明的是在安装完broker合约后，需要将返回的合约地址填入transfer和data_swapper合约中`BrokerAddr`字段，这样业务合约才能正确跨链调用。

4. 业务合约均需broker管理合约审核通过后才能进行跨链交易，调用broker合约的audit方法即是审核合约，其参数依次是业务合约地址和合约状态（数字1表示审核通过，数字2表示审核失败）。

### 2. 获取和修改Pier部署文件

#### 文件获取

##### 源码下载编译

部署跨链网关需要应用链插件，所以从源码安装的话还需要编译相应的应用链插件的二进制。

```shell
# 编译跨链网关本身
cd $HOME
git clone https://github.com/meshplus/pier.git
cd pier && git checkout ${VERSION}
make prepare && make build

# 编译Ethereum 插件
cd $HOME
git clone https://github.com/meshplus/pier-client-ethereum.git
cd pier-client-ethereum && git checkout ${VERSION}
make eth

# 说明：1.ethereum插件编译之后会在插件项目的build目录生成eth-client文件；2.pier编译之后会在跨链网关项目bin目录生成同名的二进制文件。
```

经过以上的步骤，相信您已经编译出了部署Pier和ethereum插件的二进制文件，Pier节点运行还需要外部依赖库，均在项目build目录下（Macos使用libwasmer.dylib，Linux使用libwasmer.so）,建议将得到的二进制和适配的依赖库文件拷贝到同一目录，然后使用 `export LD_LIBRARY_PATH=$(pwd)`命令指定依赖文件的路径，方便之后的操作。

##### 二进制直接下载

除了源码编译外，我们也提供了直接下载Pier及其插件二进制的方式，下载地址链接如下：[Pier二进制包下载](https://github.com/meshplus/pier/releases) 和 [ethereum插件二进制包下载](https://github.com/meshplus/pier-client-ethereum/releases)链接中已经包含了所需的二进制程序和依赖库，您只需跟据操作系统的实际情况进行选择和下载即可。

#### 修改Pier自身的配置

在进行应用链注册、验证规则部署等步骤之前，需要初始化跨链网关的配置目录，以用户目录下的pier为例：

```shell
pier --repo=~/.pier init
```

该命令会生成跨链网关的一些基础配置文件模板，使用 tree 命令可查看目录信息：

```text
tree -L 1 ~/.pier

├── api
├── certs
├── key.json
├── node.priv
└── pier.toml

1 directory, 4 files
```

导入ethereum插件二进制文件

```
mkdir -p ~/.pier/plugins
# 将ethereum插件拷贝到plugins目录下
cp eth-client ~/.pier/plugins
```

pier.toml 描述链跨链网关启动的必要配置，也是Pier的主要配置，具体的配置项和说明如下：

| 配置项         | 说明                                             |
| -------------- | ------------------------------------------------ |
| **[port]**     | http、grpc服务端口                               |
| **[log]**      | 日志输出相关设置                                 |
| **[mode]**     | 连接的中继链配置，包括relay\direct\union三种模式 |
| **[security]** | tls配置                                          |
| **[HA]**       | 主备高可用配置                                   |
| **[appchain]** | 对接的应用链的基础配置信息                       |

主要需要修改的是端口信息、中继链的信息、应用链的信息

- 修改端口信息

```none
[port]
# 如果不冲突的话，可以不用修改
http  = 44544
pprof = 44555
```

- 修改中继链信息（一般只修改addrs字段，指定bitxhub的节点地址）

```none
[mode]
type = "relay" # relay or direct
[mode.relay]
addrs = ["localhost:60011", "localhost:60012", "localhost:60013", "localhost:60014"]
quorum = 2
validators = [
    "0x000f1a7a08ccc48e5d30f80850cf1cf283aa3abd",
    "0xe93b92f1da08f925bdee44e91e7768380ae83307",
    "0xb18c8575e3284e79b92100025a31378feb8100d6",
    "0x856E2B9A5FA82FD1B031D1FF6863864DBAC7995D",
]
```

- 修改应用链信息

```none
[appchain]
# ethereum插件文件的名称
plugin = "eth-client"
# ethereum配置文件夹在跨链网关配置文件夹下的相对路径
config = "ether"
```

#### 修改ethereum插件的配置

ethereum插件的配置目录即是上一步中的ether文件夹，它的模板在`pier-client-ethereum`项目（之前拉取跨链合约时已经clone），直接在GitHub上下载代码即可

```shell
# 切换到pier-client-ethereum项目路径下
cd pier-client-ethereum
cp ./config $HOME/.pier/ether
```

其中重要配置如下：

```shell
├── account.key
├── broker.abi
├── ether.validators
├── ethereum.toml
├── password
└── validating.wasm
```

account.key和password建议换成应用链上的真实账户，且须保证有一定金额（ethereum上调用合约需要gas费），broker.abi可以使用示例，也可以使用您自己编译/部署broker合约时返回的abi，ether.validators和validating.wasm一般不需要修改。ethereum.toml是需要重点修改的，需要根据应用链实际情况填写ethereum网络地址、broker合约地址及abi，账户的key等，示例如下：

```
[ether]
addr = "wss://kovan.infura.io/ws/v3/cc512c8c74c94938aef1c833e1b50b9a"
name = "ether-kovan"
## 此处合约地址需要替换成变量代表的实际字符串
contract_address = "$brokerAddr-kovan"
abi_path = "broker.abi"
key_path = "account.key"
password = "password"
```

至此，对接ethereum应用链的pier及其插件的配置已经完成，接下来需要进行应用链注册和验证规则部署后，再启动pier节点。

### 3. 注册ethereum应用链

在启动跨链网关Pier之前，需要先注册应用链并部署验证规则，这些操作均是Pier命令行发起。需要注意的是，在v1.6.0及以上的版本，注册应用链需要中继链BitXHub节点管理员进行投票，投票通过之后才能接入。

1. Pier命令行发起应用链注册

   ```
   # 以用户目录下的pier为例
   ./pier --repo=~/.pier appchain register --name=ethereum --type=ether --consensusType POS --validators=~/.pier1/ether/ether.validators --desc="ethereum appchain for test" --version=1.0.0
   # 发起注册后会打印出应用链id和提案id
   appchain register successfully, chain id is 0xcb33b10104cd217aAB4b302e9BbeEd1957EDaA31, proposal id is 0xcb33b10104cd217aAB4b302e9BbeEd1957EDaA31-0
   ```

2. 中继链节点依次投票

   ```
   # 进入bitxhub节点的安装目录，用上一步得到的提案id进行投票
   ./bitxhub --repo ../node1 client governance vote --id 0xcb33b10104cd217aAB4b302e9BbeEd1957EDaA31-0 --info approve --reason approve
   # 投票完后会打印：vote successfully!
   # 如果是多个bitxhub节点组成的集群，依次指定各节点的安装目录进行投票
   ```

   当BitXHub集群超过半数的管理员投票通过后，应用链注册成功（如果BitXHub是solo模式，则只需要一票同意即可），可以通过如下命令查询提案状态：`bitxhub --repo ../node1 client governance proposals --type AppchainMgr `

### 4. 部署Ethereum验证规则

应用链只有在可用状态下可以部署验证规则，即需要应用链注册成功且中继链投票通过后才可以进行规则部署。提前准备好验证规则文件validating.wasm，使用以下Pier命令行进行部署。

```
#以用户目录下的pier为例
pier --repo=~/.pier rule deploy --path=~/.pier/ether/validating.wasm
```

### 5. 启动跨链网关节点

在完成以上步骤之后，可以启动跨链网关节点了

```
#以用户目录下的pier为例
pier --repo=~/.pier start
```

观察日志信息没有报错信息，可以正常同步到中继链上的区块信息，即说明pier启动成功。



## Fabric

### 1. 在应用链上部署跨链合约

目前中继链和跨链网关已经支持Fabric、Ethereum、Bcos、Cita和Hyperchain四种应用链接入并完成跨链交易，如果您有兴趣，也可以参与开发适配另外种类应用链的插件和合约。对于不同的应用链，一般都有自己的客户端调用工具，用来部署和调用链上的合约，这里简单说明如何获取已支持应用链的跨链合约及部署合约的注意事项。

**注意：在此操作之前，您需要确认已经部署或可接入的Fabric应用链**（推荐使用[官方的部署脚本](https://github.com/hyperledger/fabric-samples/tree/release-1.4)）， 在Fabric上部署跨链合约的过程和部署其它合约没有区别，只是合约名称和代码文件需要替换，以下的操作可供参考。

1. 安装部署合约的工具fabric-cli（fabric官方提供）

   ```
   go get github.com/securekey/fabric-examples/fabric-cli/cmd/fabric-cli
   ```

2. 获取需要部署的合约文件并解压（需要与pier的版本一致，下面的{VERSION}根据实际情况更改，例如 v1.6.2）

   ```
   git clone https://github.com/meshplus/pier-client-ethereum.git && git checkout ${VERSION}
   # 需要部署的合约文件就在example目录下
   # 或者是用以下方式直接下载合约压缩包
   wget https://github.com/meshplus/pier-client-fabric/raw/{VERSION}/example/contracts.zip
   #解压即可
   unzip -q contracts.zip
   ```

3. 部署broker、transfer和data_swapper合约

```
#安装和示例化broker合约（必需）
fabric-cli chaincode install --gopath ./contracts --ccp broker --ccid broker --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
fabric-cli chaincode instantiate --ccp broker --ccid broker --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel

#安装和示例化transfer合约（可选）
fabric-cli chaincode install --gopath ./contracts --ccp transfer --ccid transfer --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
fabric-cli chaincode instantiate --ccp transfer --ccid transfer --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel

#安装和示例化data_swapper合约（可选）
fabric-cli chaincode install --gopath ./contracts --ccp data_swapper --ccid data_swapper --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
fabric-cli chaincode instantiate --ccp data_swapper --ccid data_swapper --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel

#业务合约需要broker管理合约审计后，才能进行跨链交易
fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "transfer", "1"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "data_swapper", "1"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
```

4. 业务合约均需broker管理合约审核通过后才能进行跨链交易，调用broker合约的audit方法即是审核合约，其参数依次是业务合约地址和合约状态（数字1表示审核通过，数字2表示审核失败）。

### 2. 获取和修改Pier部署文件

#### 文件获取

##### 源码下载编译

部署跨链网关需要应用链插件，所以从源码安装的话还需要编译相应的应用链插件的二进制。

```shell
# 编译跨链网关本身
cd $HOME
git clone https://github.com/meshplus/pier.git
cd pier && git checkout ${VERSION}
make prepare && make build

# 编译Ethereum 插件
cd $HOME
git clone https://github.com/meshplus/pier-client-fabric.git
cd pier-client-fabric && git checkout ${VERSION}
make fabric1.4

# 说明：1.fabric插件编译之后会在插件项目的build目录生成fabric-client-1.4文件；2.pier编译之后会在跨链网关项目bin目录生成同名的二进制文件。
```

经过以上的步骤，相信您已经编译出了部署Pier和ethereum插件的二进制文件，Pier节点运行还需要外部依赖库，均在项目build目录下（Macos使用libwasmer.dylib，Linux使用libwasmer.so）,建议将得到的二进制和适配的依赖库文件拷贝到同一目录，然后使用 `export LD_LIBRARY_PATH=$(pwd)`命令指定依赖文件的路径，方便之后的操作。

##### 二进制直接下载

除了源码编译外，我们也提供了直接下载Pier及其插件二进制的方式，下载地址链接如下：[Pier二进制包下载](https://github.com/meshplus/pier/releases) 和 [fabric插件二进制包下载](https://github.com/meshplus/pier-client-fabric/releases)链接中已经包含了所需的二进制程序和依赖库，您只需跟据操作系统的实际情况进行选择和下载即可。

#### 修改Pier自身的配置

在进行应用链注册、验证规则部署等步骤之前，需要初始化跨链网关的配置目录，以用户目录下的pier为例：

```shell
pier --repo=~/.pier init
```

该命令会生成跨链网关的一些基础配置文件模板，使用 tree 命令可查看目录信息：

```text
tree -L 1 ~/.pier

├── api
├── certs
├── key.json
├── node.priv
└── pier.toml

1 directory, 4 files
```

导入fabric插件二进制文件

```
mkdir -p ~/.pier/plugins
# 将ethereum插件拷贝到plugins目录下
cp fabric-client-1.4 ~/.pier/plugins/
```

pier.toml 描述链跨链网关启动的必要配置，也是Pier的主要配置，具体的配置项和说明如下：

| 配置项         | 说明                                             |
| -------------- | ------------------------------------------------ |
| **[port]**     | http、grpc服务端口                               |
| **[log]**      | 日志输出相关设置                                 |
| **[mode]**     | 连接的中继链配置，包括relay\direct\union三种模式 |
| **[security]** | tls配置                                          |
| **[HA]**       | 主备高可用配置                                   |
| **[appchain]** | 对接的应用链的基础配置信息                       |

主要需要修改的是端口信息、中继链的信息、应用链的信息

- 修改端口信息

```none
[port]
# 如果不冲突的话，可以不用修改
http  = 44544
pprof = 44555
```

- 修改中继链信息（一般只修改addrs字段，指定bitxhub的节点地址）

```none
[mode]
type = "relay" # relay or direct
[mode.relay]
addrs = ["localhost:60011", "localhost:60012", "localhost:60013", "localhost:60014"]
quorum = 2
validators = [
    "0x000f1a7a08ccc48e5d30f80850cf1cf283aa3abd",
    "0xe93b92f1da08f925bdee44e91e7768380ae83307",
    "0xb18c8575e3284e79b92100025a31378feb8100d6",
    "0x856E2B9A5FA82FD1B031D1FF6863864DBAC7995D",
]
```

- 修改应用链信息

```none
[appchain]
# ethereum插件文件的名称
plugin = "fabric-client-1.4"
# ethereum配置文件夹在跨链网关配置文件夹下的相对路径
config = "fabric"
```

#### 修改fabric插件的配置

fabric插件的配置目录即是上一步中的fabric文件夹，它的模板在`pier-client-fabric`项目（之前拉取跨链合约时已经clone），直接在GitHub上下载代码即可

```shell
# 切换到pier-client-fabric项目路径下
cd pier-client-fabric
cp ./config $HOME/.pier/fabric
```

其中重要配置如下：

```shell
├── crypto-config/
├── config.yaml
├── fabric.toml
├── fabric.validators
└── validating.wasm
```

主要修改Fabric网络配置，验证证书，跨链合约设置：

- **fabric证书配置**

  启动Fabric网络时，会生成所有节点（包括Order、peer等）的证书信息，并保存在 crypto-config文件夹中，Fabric插件和Fabric交互时需要用到这些证书。

  ```
  # 复制您所部署的Fabric所产生的crypto-config文件夹
  cp -r /path/to/crypto-config $HOME/.pier/fabric/
  
  # 复制Fabric上验证人证书
  cp $HOME/.pier/fabric/crypto-config/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/msp/signcerts/peer1.org2.example.com-cert.pem $HOME/.pier/fabric/fabric.validators
  ```

- **修改Plugin配置文件 config.yaml **

  `config.yaml`文件记录的Fabric网络配置（用您的网络拓扑配置文件替换这个样例文件），需要使用绝对路径，把所有的路径都修改为 `crypto-config`文件夹所在的绝对路径

  ```
  path: {CONFIG_PATH}/fabric/crypto-config => path: /home/alex/.pier/fabric/crypto-config
  ```

  替换为您部署的Fabric网络的拓扑设置文件即可，同时需要修改所有的Fabric 的IP地址，如：

  ```
  url: grpcs://localhost:7050 => url: grpcs://10.1.16.48:7050
  ```

- **修改Plugin配置文件 fabric.toml**

  配置项和说明：

  | 配置项           | 说明                                |
  | ---------------- | ----------------------------------- |
  | **addr**         | Fabric 区块链所在的服务器地址和端口 |
  | **event_filter** | 跨链合约中抛出的跨链事件的名称      |
  | **username**     | Fabric用户名称                      |
  | **ccid**         | 所部署的跨链合约名称                |
  | **channel_id**   | 部署的跨链合约所在的channel         |
  | **org**          | 部署的跨链合约所在的org             |

  示例配置

  ```
  addr = "localhost:7053" // 若Fabric部署在服务器上，该为服务器地址
  event_filter = "interchain-event-name"
  username = "Admin"
  ccid = "broker" // 若部署跨链broker合约名字不是broker需要修改
  channel_id = "mychannel"
  org = "org2"
  ```

- **修改Plugin配置文件fabric.validators**

  fabric.validators 是Fabric验证人的证书，配置示例：

  ```
  -----BEGIN CERTIFICATE-----
  MIICKTCCAc+gAwIBAgIRAIBO31aZaSZoEYSy2AJuhJcwCgYIKoZIzj0EAwIwczEL
  MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
  cmFuY2lzY28xGTAXBgNVBAoTEG9yZzIuZXhhbXBsZS5jb20xHDAaBgNVBAMTE2Nh
  Lm9yZzIuZXhhbXBsZS5jb20wHhcNMjAwMjA1MDgyMjAwWhcNMzAwMjAyMDgyMjAw
  WjBqMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMN
  U2FuIEZyYW5jaXNjbzENMAsGA1UECxMEcGVlcjEfMB0GA1UEAxMWcGVlcjEub3Jn
  Mi5leGFtcGxlLmNvbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABG3jszFPTbGm
  dAYg2BxmHMTDKfQReNw3p9ttMK130qF5lQo5zLBG8Sa3viOCLnvjjg6A/P+yKnwv
  isI/jEVE8T2jTTBLMA4GA1UdDwEB/wQEAwIHgDAMBgNVHRMBAf8EAjAAMCsGA1Ud
  IwQkMCKAIMVL+daK7nMGr2/AQIXTSPFkdd3UiPVDkWtkh5ujnalEMAoGCCqGSM49
  BAMCA0gAMEUCIQDMYOQiYeMiQZTxlRkj/3/jjYvwwdCcX5AWuFmraiHkugIgFkX/
  6uiTSD0lz8P+wwlLf24cIABq2aZyi8q4gj0YfwA=
  -----END CERTIFICATE-----
  ```


至此，对接fabric应用链的pier及其插件的配置已经完成，接下来需要进行应用链注册和验证规则部署后，再启动pier节点。

### 3. 注册fabric应用链

在启动跨链网关Pier之前，需要先注册应用链并部署验证规则，这些操作均是Pier命令行发起。需要注意的是，在v1.6.0及以上的版本，注册应用链需要中继链BitXHub节点管理员进行投票，投票通过之后才能接入。

1. Pier命令行发起应用链注册

   ```
   # 以用户目录下的pier为例
   pier --repo=~/.pier appchain register --name=fabric --type=fabric --consensusType raft --validators=~/.pier1/fabric/fabric.validators --desc="fabric appchain for test" --version=1.4.3
   # 发起注册后会打印出应用链id和提案id
   appchain register successfully, chain id is 0xcb33b10104cd217aAB4b302e9BbeEd1957EDaA31, proposal id is 0xcb33b10104cd217aAB4b302e9BbeEd1957EDaA31-0
   ```

2. 中继链节点依次投票

   ```
   # 进入bitxhub节点的安装目录，用上一步得到的提案id进行投票
   ./bitxhub --repo ../node1 client governance vote --id 0xcb33b10104cd217aAB4b302e9BbeEd1957EDaA31-0 --info approve --reason approve
   # 投票完后会打印：vote successfully!
   # 如果是多个bitxhub节点组成的集群，依次指定各节点的安装目录进行投票
   ```

   当BitXHub集群超过半数的管理员投票通过后，应用链注册成功（如果BitXHub是solo模式，则只需要一票同意即可），可以通过如下命令查询提案状态：`bitxhub --repo ../node1 client governance proposals --type AppchainMgr `

### 4. 部署fabric验证规则

应用链只有在可用状态下可以部署验证规则，即需要应用链注册成功且中继链投票通过后才可以进行规则部署。提前准备好验证规则文件validating.wasm，使用以下Pier命令行进行部署。

```
#以用户目录下的pier为例
pier --repo=~/.pier rule deploy --path=~/.pier/fabric/validating.wasm
```

### 5. 启动跨链网关节点

在完成以上步骤之后，可以启动跨链网关节点了

```
#以用户目录下的pier为例
pier --repo=~/.pier start
```

观察日志信息没有报错信息，可以正常同步到中继链上的区块信息，即说明pier启动成功。

