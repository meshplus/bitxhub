# 跨链网关部署--接入Fabric

跨链网关Pier能够支持业务所在区块链（以下简称 应用链）便捷、快速的接入到跨链平台BitXHub中来，从而实现和其他业务区块链的跨链操作。跨链网关的部署需要提前确定应用链类型（对应不同的插件和配置），也需要提前在对应的应用链上部署跨链合约，为了符合用户的部署流程和提升操作体验，我们按接入应用链的类型来分别介绍说明跨链网关Pier的部署流程，主要分为在应用链上部署跨链合约、获取和修改Pier部署文件、注册应用链、部署验证规则和节点程序启动这五个章节。

## 在Fabric应用链上部署跨链合约

**注意：在此操作之前，您需要确认已经部署好Fabric1.4版本的应用链（推荐使用[官方的部署脚本](https://github.com/hyperledger/fabric-samples/tree/release-1.4)）**， 在fabric上部署跨链合约的过程本质上和部署其它合约没有区别，只是合约名称和代码文件需要替换，以下的操作可供参考。

1. 安装部署合约的工具fabric-cli（fabric官方提供）

   ```
   go get github.com/securekey/fabric-examples/fabric-cli/cmd/fabric-cli
   ```

2. 获取需要部署的合约文件并解压（需要与pier的版本一致，下面的{VERSION}根据实际情况更改，例如 v1.6.0）

   ```
   wget https://github.com/meshplus/pier-client-fabric/raw/{VERSION}/example/contracts.zip
   #解压
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



## 获取和修改Pier部署文件										

#### 安装包获取

##### 源码下载编译

部署跨链网关需要应用链插件，所以从源码安装的话还需要编译相应的应用链插件的二进制。

```shell
# 编译跨链网关本身
cd $HOME
git clone https://github.com/meshplus/pier.git
cd pier && git checkout {VERSION}
make prepare && make build

# 编译Fabric 插件
cd $HOME
git clone https://github.com/meshplus/pier-client-fabric.git
cd pier-client-fabric && git checkout {VERSION}
make fabric1.4

# 说明：1.fabric插件编译之后会在项目目录的之下的build目录生成相应的文件；2.pier编译之后会在项目bin目录生成相应的文件。
```

经过以上的步骤，相信您已经编译出了部署Pier节点（对接fabric应用链）所需的二进制文件，Pier节点运行还需要外部依赖库，均在项目build目录下（Macos使用libwasmer.dylib，Linux使用libwasmer.so）,建议将得到的二进制和适配的依赖库文件拷贝到同一目录，方便之后的操作。

##### 二进制直接下载

除了源码编译外，我们也提供了直接下载Pier及其插件二进制的方式，下载地址链接如下：[Pier二进制包下载](https://github.com/meshplus/pier/releases) 和 [fabric插件二进制包下载](https://github.com/meshplus/pier-client-fabric/releases)链接中已经包含了所需的二进制程序和依赖库，您只需跟据实际情况选择合适的版本和系统下载即可。

#### 修改配置文件

##### 修改Pier的配置

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
cp fabric-client ~/.pier/plugins
```

pier.toml 描述链跨链网关启动的必要配置，也是Pier的主要配置，具体的配置项和说明如下：

| 配置项         | 说明                                             |
| -------------- | ------------------------------------------------ |
| **[port]**     | http、grpc服务端口                               |
| **[log]**      | 日志输出相关设置                                 |
| **[mode]**     | 连接的中继链配置，包括relay\direct\union三种模式 |
| **[security]** | Tls配置                                          |
| **[HA]**       | 主备高可用配置                                   |
| **[appchain]** | 对接的应用链的基础配置信息                       |

主要需要修改的是端口信息、中继链的信息、应用链的信息

- 修改端口信息

```none
[port]
# 如果不冲突的话，可以不用修改
http  = 8987
pprof = 44555
```

- 修改中继链信息

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
# 所连接的应用链对应的Plugin文件在跨链网关配置文件夹下的相对路径
plugin = "fabric-client"
# 所连接的应用链的配置文件夹在跨链网关配置文件夹下的相对路径
config = "fabric"
```

##### 修改fabric插件配置

Fabric插件配置的模板在`pier-client-fabric`项目中，并且已经在GitHub上进行开源，所以直接在GitHub上下载代码即可

```shell
# 转到pier-client-fabric项目路径下
git clone https://github.com/meshplus/pier-client-fabric.git && cd pier-client-fabric
cp ./config $HOME/.pier/fabric
```

配置目录结构

```shell
├── crypto-config/
├── config.yaml
├── fabric.toml
└── fabric.validators
```

主要修改Fabric网络配置，验证证书，跨链合约设置：

- **Fabric证书配置**

  启动Fabric网络时，会生成所有节点（包括Order、peer等）的证书信息，并保存在 crypto-config文件夹中，Fabric插件和Fabric交互时需要用到这些证书。

  ```
  # 复制您所部署的Fabric所产生的crypto-config文件夹
  cp -r /path/to/crypto-config $HOME/.pier1/fabric/
  
  # 复制Fabric上验证人证书
  cp $HOME/.pier1/fabric/crypto-config/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/msp/signcerts/peer1.org2.example.com-cert.pem $HOME/.pier1/fabric/fabric.validators
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

  ## 注册Fabric应用链

  在启动跨链网关Pier之前，需要先注册应用链并部署绑定验证规则，这些操作均是Pier命令行发起，这一章我们介绍注册Fabric应用链的操作步骤。需要注意的是，在v1.6.0及以上的版本，注册应用链需要中继链BitXHub节点管理员进行投票，投票通过之后才能接入。

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
     bitxhub --repo ../node1 client governance vote --id 0xcb33b10104cd217aAB4b302e9BbeEd1957EDaA31-0 --info approve --reason approve
     # 投票完后会打印：vote successfully!
     ```

     当BitXHub集群超过半数的管理员投票通过后，应用链注册成功（如果BitXHub是solo模式，则只需要一票同意即可），可以通过如下命令查询提案状态：`bitxhub --repo ../node1 client governance proposals --type AppchainMgr `

  ## 部署Fabric验证规则

  应用链只有在可用状态下可以部署验证规则，即需要应用链注册成功后才可以进行规则部署。提前准备好验证规则文件validating.wasm，使用以下Pier命令行进行部署。

  ```
  #以用户目录下的pier为例
  pier --repo=~/.pier rule deploy --path=~/.pier/fabric/validating.wasm
  ```

  ## 启动跨链网关节点

  在完成以上步骤之后，可以启动跨链网关节点了

  ```
  #以用户目录下的pier为例
  pier --repo=~/.pier start
  ```

  观察日志信息没有报错信息，可以正常同步到中继链上的区块信息，即说明pier启动成功


