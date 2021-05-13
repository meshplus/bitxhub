# 跨链网关部署--接入Ethereum

跨链网关Pier能够支持业务所在区块链（以下简称 应用链）便捷、快速的接入到跨链平台BitXHub中来，从而实现和其他业务区块链的跨链操作。跨链网关的部署需要提前确定应用链类型（对应不同的插件和配置），也需要提前在对应的应用链上部署跨链合约，为了符合用户的部署流程和提升操作体验，我们按接入应用链的类型来分别介绍说明跨链网关Pier的部署流程，主要分为在应用链上部署跨链合约、获取和修改Pier部署文件、注册应用链、部署验证规则和节点程序启动这五个章节。

## 在Ethereum应用链上部署跨链合约

**注意：在此操作之前，您需要确认已经部署或可接入的Ethereum应用链**， 在Ethereum上部署跨链合约的过程本质上和部署其它合约没有区别，只是合约名称和代码文件需要替换。在Ethereum上部署合约的工具有很多，您可以使[Remix](https://remix.ethereum.org/)进行合约的编译和部署，这里关键的是跨链合约的获取。

1. 下载pier-client-ethereum源码

   ```
   git clone https://github.com/meshplus/pier-client-ethereum.git
   ```

2. 需要部署的合约文件就在example目录下，后缀名是.sol，注意切换到与Pier一致的分支或版本

3. 部署broker、transfer和data_swapper合约的过程不再赘述，需要特别说明的是在安装完broker合约后，需要将返回的合约地址填入transfer和data_swapper合约中`BrokerAddr`字段，这样业务合约才能正确跨链调用。此外，与Fabric一样，业务合约需要broker管理合约审计后，才能进行跨链交易。

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
git clone https://github.com/meshplus/pier-client-ethereum.git
cd pier-client-fabric && git checkout {VERSION}
make eth

# 说明：1.ethereum插件编译之后会在项目目录的之下的build目录生成相应的文件；2.pier编译之后会在项目bin目录生成相应的文件。
```

经过以上的步骤，相信您已经编译出了部署Pier节点（对接ethereum应用链）所需的二进制文件，Pier节点运行还需要外部依赖库，均在项目build目录下（Macos使用libwasmer.dylib，Linux使用libwasmer.so）,建议将得到的二进制和适配的依赖库文件拷贝到同一目录，方便之后的操作。

##### 二进制直接下载

除了源码编译外，我们也提供了直接下载Pier及其插件二进制的方式，下载地址链接如下：[Pier二进制包下载](https://github.com/meshplus/pier/releases) 和 [ethereum插件二进制包下载](https://github.com/meshplus/pier-client-ethereum/releases)链接中已经包含了所需的二进制程序和依赖库，您只需跟据实际情况选择合适的版本和系统下载即可。

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
cp eth-client ~/.pier/plugins
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
plugin = "eth-client"
# 所连接的应用链的配置文件夹在跨链网关配置文件夹下的相对路径
config = "ether"
```

##### 修改ethereum插件配置

Ethereum插件配置的模板在`pier-client-ethereum`项目中，并且已经在GitHub上进行开源，所以直接在GitHub上下载代码即可

```shell
# 转到pier-client-ethereum项目路径下
git clone https://github.com/meshplus/pier-client-ethereum.git && cd pier-client-ethereum
cp ./config $HOME/.pier/ether
```

重要配置如下：

```shell
├── account.key
├── ether.validators
├── ether.toml
├── password
└── validating.wasm
```

主要修改ethereum.toml文件，需要根据应用链实际情况填写，示例如下：

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

## 注册Ethereum应用链

在启动跨链网关Pier之前，需要先注册应用链并部署绑定验证规则，这些操作均是Pier命令行发起，这一章我们介绍注册Ethereum应用链的操作步骤。需要注意的是，在v1.6.0及以上的版本，注册应用链需要中继链BitXHub节点管理员进行投票，投票通过之后才能接入。

1. Pier命令行发起应用链注册

   ```
   # 以用户目录下的pier为例
   pier --repo=~/.pier appchain register --name=ethereum --type=ether --consensusType POS --validators=~/.pier1/ether/ether.validators --desc="ethereum appchain for test" --version=1.0.0
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

## 部署Ethereum验证规则

应用链只有在可用状态下可以部署验证规则，即需要应用链注册成功后才可以进行规则部署。提前准备好验证规则文件validating.wasm，使用以下Pier命令行进行部署。

```
#以用户目录下的pier为例
pier --repo=~/.pier rule deploy --path=~/.pier/ether/validating.wasm
```

## 启动跨链网关节点

在完成以上步骤之后，可以启动跨链网关节点了

```
#以用户目录下的pier为例
pier --repo=~/.pier start
```

观察日志信息没有报错信息，可以正常同步到中继链上的区块信息，即说明pier启动成功


