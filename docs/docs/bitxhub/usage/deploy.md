# 中继链部署

中继链用于应用链的跨链管理，以及跨链交易的可信验证与可靠路由，是一种实现IBTP

协议的开放许可链。

### 安装包获取

##### 源码下载

首先拉区bitxhub源代码：

```shell
git clone https://github.com/meshplus/bitxhub.git
```

代码目录结构如下：

| 目录      | 说明        |
| ------- | --------- |
| api     | 网关和GRPC服务 |
| cmd     | 命令行相关代码   |
| config  | 配置文件      |
| docs    | 设计和用户文档   |
| intenal | 项目内部相关代码  |
| pkg     | 项目重用相关代码  |
| scripts | 操作脚本      |
| tester  | 单元测试      |

##### 二进制下载

可以在github上下载已经打包好的二进制安装包，地址如下：`https://github.com/meshplus/bitxhub/releases`, 根据需要的版本进行下载即可。请注意，在bitxhub v1.6.0及之后，二进制包和部署配置示例文件将分为两个压缩包提供，其中配置文件是以四节点bitxhub集群为示例，文件命名以examples开头，如果只下载配置文件，仍需要将二进制程序拷贝到指定地方之后才能启动节点。

### 修改配置文件

中继链包括bitxhub.toml、network.tom和order.toml配置文件。下面以node1为例介绍如何修改配置文件。

##### 节点配置文件bitxhub.toml

bitxhub.toml文件是bitxhub启动的主要配置文件。各配置项说明如下：

| 配置项     | 说明                                  |
| ---------- | ------------------------------------- |
| solo       | 是否按照单节点模式启动BitXHub         |
| [port]     | gateway、grpc、pprof和monitor服务端口 |
| [monitor]  | 监控服务                              |
| [security] | 证书体系                              |
| [cert]     | 是否开启认证节点p2p通信证书           |
| [order]    | 共识模块，作为插件进行加载            |
| [gateway]  | 跨域配置                              |
| [ping]     | ping集群节点功能                      |
| [log]      | 日志输出相关设置                      |
| [executor] | 执行引擎类型                          |
| [genesis]  | 创世节点配置                          |

变更gateway、grpc、pprof和monitor等端口:

```javascript
[port]
  gateway = 9091
  grpc = 60011
  pprof = 53121
  monitor = 40011
```

共识算法类型选择（支持raft和rbft）：

```shell
[order]
  plugin = "plugins/raft.so" 
```

执行引擎类型选择（支持serial和parallel）：

```sh
[executor]
  type = "serial"
```

修改genesis的节点验证者地址，根据节点数量配置地址数量

```shell
[genesis]
addresses = [
  "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
  "0x79a1215469FaB6f9c63c1816b45183AD3624bE34",
  "0x97c8B516D19edBf575D72a172Af7F418BE498C37",
  "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8"
]
```

##### 网络配置文件network.toml

network.toml文件是bitxhub网络配置文件。各配置项说明如下：

| 配置项  | 说明                       |
| ------- | -------------------------- |
| N       | 集群节点数量               |
| id      | 当前节点标识               |
| new     | 判断当前节点是新加入的节点 |
| [nodes] | 集群节点信息               |
| account | 节点验证者地址             |
| hosts   | 节点网络地址               |
| id      | 节点标识                   |
| pid     | p2p网络唯一标识            |

配置过程变更nodes中各个节点的信息

```shell
[[nodes]]
account = "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"
hosts = ["/ip4/127.0.0.1/tcp/4001/p2p/"]
id = 1
pid = "QmXi58fp9ZczF3Z5iz1yXAez3Hy5NYo1R8STHWKEM9XnTL"
```

配置示例如下：

```javascript
id = 1 # self id
n = 4 # the number of vp nodes
new = false # track whether the node is a new node

[[nodes]]
account = "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"
hosts = ["/ip4/127.0.0.1/tcp/4001/p2p/"]
id = 1
pid = "QmXi58fp9ZczF3Z5iz1yXAez3Hy5NYo1R8STHWKEM9XnTL"

[[nodes]]
account = "0x79a1215469FaB6f9c63c1816b45183AD3624bE34"
hosts = ["/ip4/127.0.0.1/tcp/4002/p2p/"]
id = 2
pid = "QmbmD1kzdsxRiawxu7bRrteDgW1ituXupR8GH6E2EUAHY4"

[[nodes]]
account = "0x97c8B516D19edBf575D72a172Af7F418BE498C37"
hosts = ["/ip4/127.0.0.1/tcp/4003/p2p/"]
id = 3
pid = "QmQUcDYCtqbpn5Nhaw4FAGxQaSSNvdWfAFcpQT9SPiezbS"

[[nodes]]
account = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8"
hosts = ["/ip4/127.0.0.1/tcp/4004/p2p/"]
id = 4
pid = "QmQW3bFn8XX1t4W14Pmn37bPJUpUVBrBjnPuBZwPog3Qdy"
```

##### 共识配置文件order.toml

order.toml文件是bitxhub共识配置文件。各配置项说明如下：

| 配置项 | 说明          |
| ------ | ------------- |
| [raft] | raft 相关配置 |
| [rbft] | rbft 相关配置 |
| [solo] | solo相关配置  |

配置示例如下（无特殊情况不要修改此配置）：

```javascript
[raft]
batch_timeout               = "0.3s"  # Block packaging time period.
tick_timeout                = "0.1s" # TickTimeout is the internal logical clock for the Node by a single tick, Election timeouts and heartbeat timeouts are in units of ticks.
election_tick               = 10 # ElectionTick is the number of Node.Tick invocations that must pass between elections.
heartbeat_tick              = 1  # HeartbeatTick is the number of Node.Tick invocations that must pass between heartbeats.
max_size_per_msg            = 1048576 # 1024*1024, MaxSizePerMsg limits the max size of each append message.
max_inflight_msgs           = 500  # MaxInflightMsgs limits the max number of in-flight append messages during optimistic replication phase.
check_quorum                = true # Leader steps down when quorum is not active for an electionTimeout.
pre_vote                    = true # PreVote prevents reconnected node from disturbing network.
disable_proposal_forwarding = true # This prevents blocks from being accidentally proposed by followers.

    [raft.mempool]
        batch_size          = 200   # How many transactions should the primary pack.
        pool_size           = 50000 # How many transactions could the txPool stores in total.
        tx_slice_size       = 10    # How many transactions should the node broadcast at once
        tx_slice_timeout    = "0.1s"  # Node broadcasts transactions if there are cached transactions, although set_size isn't reached yet

    [raft.syncer]
        sync_blocks = 1 # How many blocks should the behind node fetch at once
        snapshot_count = 1000  # How many apply index(blocks) should the node trigger at once

[rbft]        #RBFT configurations
set_size         = 25    # How many transactions should the node broadcast at once
batch_size       = 500   # How many transactions should the primary pack before sending pre-prepare
pool_size        = 50000 # How many transactions could the txPool stores in total
vc_period        = 0     # After how many checkpoint periods( Blocks = 10 * vcperiod ) the primary gets cycled automatically. ( Set 0 to disable )
check_interval   = "3m"  # interval of the check loop
tolerance_time   = "5m"  # The max tolerance time duration (in seconds) of out-of-date
batch_mem_limit  = false # Indicates whether limit batch mem size or not
batch_max_mem    = 10000 # The max memory size of one batch

    [rbft.timeout]
    	sync_state        = "3s"  # How long to wait quorum sync state response
        sync_interval     = "1m"  # How long to restart sync state process
        recovery          = "15s" # How long to wait before recovery finished(This is for release1.2)
        first_request     = "30s" # How long to wait before first request should come
        batch             = "0.5s"# Primary send a pre-prepare if there are pending requests, although batchsize isn't reached yet,
        request           = "6s"  # How long may a request(transaction batch) take between reception and execution, must be greater than the batch timeout
        null_request      = "9s"  # Primary send it to inform aliveness, must be greater than request timeout
        viewchange        = "8s"  # How long may a view change take
        resend_viewchange = "10s" # How long to wait for a view change quorum before resending (the same) view change
        clean_viewchange  = "60s" # How long to clean out-of-data view change message
        update            = "4s"  # How long may a update-n take
        set               = "0.1s" # Node broadcasts transactions if there are cached transactions, although set_size isn't reached yet

    [rbft.syncer]
        sync_blocks = 1 # How many blocks should the behind node fetch at once

[solo]
batch_timeout = "0.3s"  # Block packaging time period.

   [solo.mempool]
        batch_size          = 200   # How many transactions should the primary pack.
        pool_size           = 50000 # How many transactions could the txPool stores in total.
        tx_slice_size       = 10    # How many transactions should the node broadcast at once
        tx_slice_timeout    = "0.1s"  # Node broadcasts transactions if there are cached transactions, although set_size isn't reached yet
```

### 启动程序

##### 初始化配置

```
$ bitxhub init
```

##### 生成节点验证者私钥，并通过私钥获取验证者地址

```shell
$ bitxhub key gen --target ~/.bitxhub
$ bitxhub key address --path ~/.bitxhub/key.json
0x0beb9583C069aeC5B5C3B395a1Ee644BFdd5Ce0D
```

##### 获取节点p2p通信网络pid

```
$ bitxhub cert priv gen --name node --target ~/.bitxhub
$ bitxhub cert priv pid --path ~/.bitxhub/node.priv
QmWAaFDQ3p2Hj383WsBGU2nLMtsJk1aT9obXXXxL5UyUuA
```

##### 拷贝共识插件

```
$ mkdir ～/.bitxhub/plugins

1. 拷贝raft共识插件, 若源码编译即在internal/plugins/build目录内
   $ cp raft.so ～/.bitxhub/plugins

2. 拷贝rbft共识插件, 若源码编译即在bitxhub-order-rbft/build目录内
   $ cp rbft.so ~/.bitxhub/plugins
```

##### 启动bitxhub

```
$ bitxhub --repo ~/.bitxhub start
```

待节点集群打印出bitxhub的LOGO，表示bitxhub集群开始正常工作

![](../assets/bitxhub.png)



## 跨链网关部署

跨链网关Pier能够支持业务所在区块链便捷、快速的接入到跨链平台BitXHub中来，从而实现和其他业务区块链的跨链操作。该跨链网关支持跨链消息格式转换、跨链消息的路由、跨链操作的调用等核心功能，不仅保证不同格式的跨链消息能够安全可信的到达目标应用链，而且保证了跨链交易异常情况下来源链的安全。跨链网关为区块链互联形成网络提供了便捷的接入方式，旨在降低跨链互联的使用成本。下面是具体的安装部署教程。												

### 安装包获取

##### 源码安装

跨链网关启动的话需要应用链插件，所以从源码安装的话，还需要编译相应的应用链插件的二进制。

```shell
# 编译跨链网关本身
cd $HOME
git clone https://github.com/meshplus/pier.git
cd pier
make prepare && make install

# 编译Fabric
cd $HOME
git clone https://github.com/meshplus/pier-client-fabric.git
cd pier-client-fabric
make fabric1.4

# 编译以太坊私链插件
cd $HOME
git clone https://github.com/meshplus/pier-client-ethereum.git
cd pier-client-ethereum
make eth

# 插件执行make的编译之后，都会在项目目录的之下的build目录生成相应的 .so 文件
```

编译跨链网关步骤会在 $GOPATH/bin 下生成 pier 二进制，运行下面的命令查看是否安装成功：

```shell
pier version
```

如果正常安装会打印出类似下面的说明

```text
Pier version: 1.0.0-b01a80a
App build date: 2020-08-27T10:28:05
System version: darwin/amd64
Golang version: go1.13
```

代码目录结构如下：

| 目录       | 说明                 |
| -------- | ------------------ |
| agent    | 对接BitXHub的Client模块 |
| cmd      | 命令行相关代码            |
| internal | 项目内部相关代码           |
| pkg      | 项目重用相关代码           |
| plugins  | 对接应用链的Client接口     |
| scripts  | 操作脚本               |

**3.2.1.2 二进制安装**

没有现有编译环境的用户，也可以在GitHub开源仓库下载编译好的二进制，地址：`https://github.com/meshplus/pier/releases`, 根据需要的版本进行下载即可。该部署包中包含了 Pier跨链网关的二进制和 pier-client-fabric 和 pier-client-ethereum 的应用链插件的二进制。

### 修改配置文件

在进行应用链注册、验证规则部署等步骤之前，需要初始化跨链网关的配置目录

```shell
#以用户目录下的pier为例
pier --repo=~/pier init
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

导入插件二进制（hyperchain的插件二进制和配置文件示例需要内部授权）

```
mkdir -p ~/.pier/plugins
cp fabric-client-1.4.so ~/.pier/plugins
```

pier.toml 文件描述链跨链网关启动的必要配置，具体的配置项和说明如下：

| 配置项        | 说明                    |
| ---------- | --------------------- |
| [port]     | http、grpc服务端口         |
| [log]      | 日志输出相关设置              |
| [bitxhub]  | 连接的bitxhub的IP地址、验证人地址 |
| [appchain] | 对接的应用链的基础配置信息         |

主要需要修改的部分是端口信息、中继链的信息、应用链的信息

- 修改端口信息

```none
[port]
// 如果不冲突的话，可以不用修改
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
// 所连接的应用链对应的Plugin文件在跨链网关配置文件夹下的相对路径
plugin = "fabric-client-1.4.so"
// 所连接的应用链的配置文件夹在跨链网关配置文件夹下的相对路径
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
  # 复制你所部署的Fabric所产生的crypto-config文件夹
  cp -r /path/to/crypto-config $HOME/.pier1/fabric/
  
  # 复制Fabric上验证人证书
  cp $HOME/.pier1/fabric/crypto-config/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/msp/signcerts/peer1.org2.example.com-cert.pem $HOME/.pier1/fabric/fabric.validators
  ```

- **修改Plugin配置文件 config.yaml **

  `config.yaml`文件记录的Fabric网络配置（如果你是按照你自己的网络拓扑部署的Fabric，用你的网络拓扑配置文件替换这个样例文件），需要使用绝对路径，把所有的路径都修改为 `crypto-config`文件夹所在的绝对路径

  ```
  path: {CONFIG_PATH}/fabric/crypto-config => path: /home/alex/.pier/fabric/crypto-config
  ```

  替换为你部署的Fabric网络的拓扑设置文件即可，同时需要修改所有的Fabric 的IP地址，如：

  ```
  url: grpcs://localhost:7050 => url: grpcs://10.1.16.48:7050
  ```

- **修改Plugin配置文件 fabric.toml**

  配置项和说明：

  | 配置项       | 说明                                |
  | ------------ | ----------------------------------- |
  | addr         | Fabric 区块链所在的服务器地址和端口 |
  | event_filter | 跨链合约中抛出的跨链事件的名称      |
  | username     | Fabric用户名称                      |
  | ccid         | 所部署的跨链合约名称                |
  | channel_id   | 部署的跨链合约所在的channel         |
  | org          | 部署的跨链合约所在的org             |

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

  ### 启动程序

  ```
  #以用户目录下的pier1为例
  pier --repo=~/pier start
  ```

  观察日志信息没有报错信息，pier启动成功

  **说明：因为跨链合约和验证规则的部署涉及到不同应用链的细节，且需依赖应用链的安装部署，具体操作请见快速开始手册或使用文档，这里不再赘述**

