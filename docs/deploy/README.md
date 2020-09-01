# BitXHub v1.0.0 规范化部署文档

# 前言

本部署文档将介绍如何部署一个四节点的中继链，以及一条Fabric应用链和一条以太坊应用链。并对应两条应用链部署各自部署跨链网关，以及如何部署和调用跨链合约进行跨链转账操作。

## 前提

本文默认是在在Linux机器上进行操作，有些基础的工具可能要在服务器上提前安装：

1. **git**
2. **docker**（部署Fabric区块链的服务器需要）

# 部署流程

## 部署中继链

### 1.准备

该文档将介绍如何部署一个拥有4个节点的BitXHub集群，操作步骤会较其他系统的部署稍繁琐一些，用户需要**分别登录到4台服务器（或者在一台服务器上设置不同端口)**上进行操作。

这里假设4台服务器的IP分别为`node1`、`node2`、`node3`和`node4`。操作用户都是`bitxhub`。

### 2. 一键脚本部署

在bitxhub项目下，提供了一键部署的脚本，适合在有项目权限的情况下进行部署。

```javascript
bitxhub/scripts
├── build
├── certs
├── cluster.sh
├── config.sh
├── cross_compile.sh
├── deploy.sh
├── prepare.sh
├── quick_start
├── solo.sh
└── x.sh
```

进入bitxhub项目，运行下面的命令进行部署：

```shell
## -a 为服务器地址（需要有ssh登陆权限，服务器安装tmux窗口管理器）
## -n 为需要在服务器上部署的结点数量
## -r 是否需要重新编译项目，可设为true和false
## -u 服务器ssh用户名
## -p bitxhub部署相对路径
## e.g. bash deploy.sh -a 40.125.161.213 -n 4 -r false -u root -p bitxhub

$ bash deploy.sh [-a <bitxhub_addr>] [-n <node_num>] [-r <if_recompile>] [-u <username>] [-p <build_path>]
```

#### 插件化部署（可选）

进入plugins子目录，运行下面的命令进行编译共识算法插件：

```shell
## make raft编译共识算法
$ make raft
```

编译完成后，节点会根据bitxhub.toml文件中的order配置加载不同的共识算法。

### 3. 规范化部署

#### 前言

该文档将介绍如何部署一个拥有4个节点的BitXHub集群，操作步骤会较其他系统的部署稍繁琐一些，用户需要**分别登录到4台服务器（或者在一台服务器上设置不同端口**上进行操作。

这里假设4台服务器的IP分别为`node1`、`node2`、`node3`和`node4`。操作用户都是`bitxhub`。

#### 3.1 获取安装包

从下面的命令下载获取BitXHub的 tar 包

```shell
$ ssh bitxhub@node1
$ mkdir $HOME/.bitxhub
# 在Mac环境下
$ wget https://github.com/meshplus/bitxhub/releases/download/v1.0.0-rc1/build_macos_x86_64_v1.0.0-rc1.tar.gz
$ tar xvf build_macos_x86_64_v1.0.0-rc1.tar.gz -C $HOME/.bitxhub
# 在Linux环境下
$ wget https://github.com/meshplus/bitxhub/releases/download/v1.0.0-rc1/build_linux-amd64_v1.0.0-rc1.tar.gz
$ tar xvf build_linux-amd64_v1.0.0-rc1.tar.gz -C $HOME/.bitxhub

# bitxhub运行需要libwasmer.so动态链接库
$ export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$HOME/.bitxhub/build
```

解压完成之后，会在 `$HOME/.bitxhub` 下看到如下build目录

```javascript
.
├── addresses
├── agency.cert
├── agency.priv
├── bitxhub
├── ca.cert
├── ca.priv
├── libwasmer.so
├── node1
│   ├── README.md
│   ├── api
│   ├── bitxhub.toml
│   ├── certs/
│   ├── genesis.json
│   ├── network.toml
│   ├── order.toml
│   ├── plugins
│   └── start.sh
├── node2
│   ├── README.md
│   ├── api
│   ├── bitxhub.toml
│   ├── certs/
│   ├── genesis.json
│   ├── network.toml
│   ├── order.toml
│   ├── plugins
│   └── start.sh
├── node3
│   ├── README.md
│   ├── api
│   ├── bitxhub.toml
│   ├── certs/
│   ├── genesis.json
│   ├── network.toml
│   ├── order.toml
│   ├── plugins
│   └── start.sh
├── node4
│   ├── README.md
│   ├── api
│   ├── bitxhub.toml
│   ├── certs/
│   ├── genesis.json
│   ├── network.toml
│   ├── order.toml
│   ├── plugins
│   └── start.sh
├── pids
├── raft.so
└── solo.so
```

该部署包已经包含了四个节点的配置目录，如果是在多台服务器上部署，每台服务器上操作时只需要修改其中一个节点的配置目录即可。

#### 3.2 修改配置文件

下面以node1为例介绍如何修改配置文件

**修改bitxhub.toml文件**

```shell
title = "BitXHub configuration file"
# 是否按照单结点模式启动BitXHub
solo = false

# BitXHub提供服务的端口，确保和已占用的端口不冲突
[port]
  grpc = 60011
  gateway = 9091
  pprof = 53121

[pprof]
  enable = true

# 网关白名单
[gateway]
    allowed_origins = ["*"]

# 日志输出相关设置
[log]
  level = "info"
  dir = "logs"
  filename = "bitxhub.log"
  report_caller = false
  [log.module]
    p2p = "info"
    consensus = "info"
    executor = "info"
    router = "info"
    api = "info"
    coreapi = "info"

[cert]
  verify = true

# BitXHub使用的共识算法，共识模块作为插件进行加载
[order]
  plugin = "plugins/raft.so"

# BitXHub启动的创世块信息
[genesis]
    addresses = [
        "0xe6f8c9cf6e38bd506fae93b73ee5e80cc8f73667",
        "0x8374bb1e41d4a4bb4ac465e74caa37d242825efc",
        "0x759801eab44c9a9bbc3e09cb7f1f85ac57298708",
        "0xf2d66e2c27e93ff083ee3999acb678a36bb349bb"
    ]
```

**修改network.toml文件**

```shell
# BitXHub结点的IP和端口信息，BitXHub结点的id具有唯一性
N = 4
# 推荐在node1服务器上配置为id=1,其他node的服务器ID递增
id = 1

[[nodes]]
  addr = "/ip4/127.0.0.1/tcp/4001/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5ngmL"
  id = 1

[[nodes]]
  addr = "/ip4/127.0.0.1/tcp/4002/p2p/QmTGbPAfCYiAYDwYytt3QQLn3fq79dzciyP9kTFYWr8Lqb"
  id = 2

[[nodes]]
  addr = "/ip4/127.0.0.1/tcp/4003/p2p/QmNxNoU52ZmSaeFS9MEUHAvusp6iqoqZRKnpoKwUvHkVdB"
  id = 3

[[nodes]]
  addr = "/ip4/127.0.0.1/tcp/4004/p2p/QmaKBzZw94uqRRr5w8n4DMzrYcJ8V9VkyVYRxBSYYvi1te"
  id = 4

```

按照上面配置的注释相应的进行修改，其中需要注意的是：每个节点的 `addr` 的最后一段ID需要进行修改。

```shell
 $ $HOME/.bitxhub/build/bitxhub key pid --path $HOME/.bitxhub/build/node1/certs/node.priv
```

运行上面的命令会得到类似于 `QmP62PJJBSZCYLDdfFEraxG4pACAR2k83JEDY59zsM4HD2` 的一个ID，将此ID替换 `network.toml` 中 `node1` 的 `addr` 后缀即可，比如：

```shell
addr = "/ip4/127.0.0.1/tcp/4001/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5ngmL"
=> 修改为
addr = "/ip4/127.0.0.1/tcp/4001/p2p/QmP62PJJBSZCYLDdfFEraxG4pACAR2k83JEDY59zsM4HD2"
```

其他节点同理进行修改即可。

**修改order.toml**

```javascript
# 共识算法插件的配置文件

[raft]
election_tick               = 10 # ElectionTick is the number of Node.Tick invocations that must pass between elections.
heartbeat_tick              = 1 # HeartbeatTick is the number of Node.Tick invocations that must pass between heartbeats.
max_size_per_msg            = 1048576 # 1024*1024, MaxSizePerMsg limits the max size of each append message.
max_inflight_msgs           = 500 # MaxInflightMsgs limits the max number of in-flight append messages during optimistic replication phase.
check_quorum                = true # Leader steps down when quorum is not active for an electionTimeout.
pre_vote                    = true # PreVote prevents reconnected node from disturbing network.
disable_proposal_forwarding = true # This prevents blocks from being accidentally proposed by followers.
    [raft.tx_pool]
        pack_size           = 500 # How many transactions should the primary pack.
        pool_size           = 50000 # How many transactions could the txPool stores in total.
        block_tick          = "500ms" # Block packaging time period.
```

#### 3.3 启动BitXHub

 将bitxhub和raft.so二进制放到各自的node目录下，再执行下面的命令

```shell
$ cp $HOME/.bitxhub/build/bitxhub $HOME/.bitxhub/build/node/
$ cp $HOME/.bitxhub/build/raft.so $HOME/.bitxhub/build/node/plugins/
$ cd $HOME/.bitxhub/build/node1
$ bash start.sh
```

等待BitXHub进行共识，打印出BitXHub标志后即部署成功。

## 部署Fabric和跨链合约

### 1. 部署Fabric

按照Fabric[官方文档](https://hyperledger-fabric.readthedocs.io/en/release-1.4/build_network.html)启动一条默认配置的Fabric 1.4.3版本的区块链即可。

在你的机器上启动Fabric 1.4.3 的应用链之后，会得到一个crypto-config的文件夹，这个文件夹在后面跨链网关的部署流程中会用到。

### 2. 部署 chaincode

要将Fabric接入跨链系统，需要在Fabric上部署跨链合约。

下载chaincode跨链合约

```shell
$ wget https://github.com/meshplus/pier-client-fabric/raw/master/example/contracts.zip
$ unzip contract.zip
```

我们提供的跨链合约提供了一个跨链管理合约 `broker` 和两个样例应用合约 `transfer`和 `data_swapper`

```shell
└── src
    ├── broker
    │   ├── broker.go
    │   ├── data_swapper.go
    │   ├── helper.go
    │   ├── meta.go
    │   └── transfer.go
    ├── data_swapper
    │   └── data_swapper.go
    └── transfer
        ├── helper.go
        └── transfer.go
```

将这几个合约部署到Fabric上即可，chaincode名称推荐使用对应的项目名不用改变。

**broker合约对应用合约进行审核**

```go
func (broker *Broker) audit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	channel := args[0]
	chaincodeName := args[1]
	status := args[2]
  ...
}
```

调用broker合约中的 `audit` 方法，参数是 fabric网络中channel名 + chaincode名称 + 1（数字1表示审核代码通过）

## 部署以太坊和跨链合约

### 1. 部署以太坊私链

使用以太坊私链只是为了演示跨链流程，建议使用Geth 客户端的Dev 模式进行启动，并且需要开启websocket端口和RPC端口。

### 2. 部署Solidity跨链合约

下载Solidity跨链合约

```shell
$ git clone https://github.com/meshplus/pier-client-ethereum.git
$ cd pier-client-ethereum/example
```

我们提供的Solidity跨链合约提供了一个跨链管理合约 `broker` 和两个样例应用合约 `transfer`和 `data_swapper`

```shell
.
├── broker.sol
├── data_swapper.sol
└── transfer.sol
```

部署合约的先后顺序需要注意，需要首先部署 `broker` 合约，得到它的地址比如是 `0xD3880ea40670eD51C3e3C0ea089fDbDc9e3FBBb4`

再修改 `data_swapper` 和 `transfer` 代码中的 `broker` 地址，以 `data_swapper` 为例说明如何修改：

```javascript
contract DataSwapper {
	address BrokerAddr = 0x2346f3BA3F0B6676aa711595daB8A27d0317DB57; // 替换为你部署broker得到的地址
	Broker broker = Broker(BrokerAddr);
  ....
}
```

才能进行 `data_swapper` 和 `transfer` 的部署操作。

**broker合约对应用合约进行审核**

```javascript
function audit(address addr, int64 status) public returns(bool)
```

调用broker合约中的 `audit` 方法，参数是之前部署得到的 `transfer` 或者 `data_swapper` 合约地址 + 1（数字1表示审核代码通过）

## 启动跨链网关

### 1. 获取安装包

从下面的链接获取跨链网关的二进制安装包并进行安装：

```shell
# 在Mac环境下
$ wget https://github.com/meshplus/pier/releases/download/v1.0.0-rc1/pier-macos-x86-64.tar.gz
$ mkdir pier-binary && tar xvf pier-macos-x86-64.tar.gz -C pier-binary
# 在Linux环境下
$ wget https://github.com/meshplus/pier/releases/download/v1.0.0-rc1/pier-linux-amd64.tar.gz
$ mkdir pier-binary && tar xvf pier-linux-amd64.tar.gz -C pier-binary

# pier运行需要libwasmer.so动态链接库(Linux下)或者 libwasmer.dylib（Mac下）
# Linux下
$ wget https://raw.githubusercontent.com/meshplus/bitxhub/master/build/libwasmer.so -O pier-binary/libwasmer.so
# Mac下
$ wget https://raw.githubusercontent.com/meshplus/bitxhub/master/build/libwasmer.dylib -O pier-binary/libwasmer.dylib
# 两个平台都要执行
$ export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$(pwd)/pier-binary/
```

解压后可以看到下面的文件

```shell
- pier-binary
├── eth-client.so
├── fabric1.4-client.so
├── pier
```

eth-client.so 和 fabric1.4-client.so分别用于以太坊和Fabric的跨链网关的启动。

并将 `pier` 放到环境变量目录下

```shell
cp pier-binary/pier /usr/local/bin/
```

### 2. 跨链网关配置

Pier 需要配置中继链和应用链的相关信息，可以指定配置目录，默认路径为 `$HOME/.pier`

```shell
# fabric 应用链初始化为$HOME/.pier1，以太坊应用链初始化为$HOME/.pier2，区分开即可
$ pier --repo=$HOME/.pier1 init

# 查看具体的配置内容
$ cat $HOME/.pier1/pier.toml
```

主要需要修改的部分是端口信息、中继链的信息、应用链的信息

- 修改端口信息

```toml
[port]
# 如果不冲突的话，可以不用修改
http = 44544
pprof = 44555
```

- 修改中继链信息

```toml
[bitxhub]
# 在服务器启动的bitxhub需要修改为服务器地址
addr = "localhost:60011"

# 修改为BitXHub节点的地址
validators = [
    "0x000f1a7a08ccc48e5d30f80850cf1cf283aa3abd",
    "0xe93b92f1da08f925bdee44e91e7768380ae83307",
    "0xb18c8575e3284e79b92100025a31378feb8100d6",
    "0x856E2B9A5FA82FD1B031D1FF6863864DBAC7995D",
]
```

- 修改应用链信息

```toml
[appchain]
# 所连接的应用链对应的Plugin文件在跨链网关配置文件夹下的相对路径
plugin = "fabric-client-1.4.so"

# 所连接的应用链的配置文件夹在跨链网关配置文件夹下的相对路径
config = "fabric"
```

### 3. Fabric应用链插件配置

插件配置的模板在`pier-client-fabric`项目中

```shell
$ mkdir -p $HOME/.pier1/plugins

# 转到下载解压后的二进制文件夹
$ cp ./pier-binary/fabric1.4-client.so $HOME/.pier1/plugins/fabric-client-1.4.so

# 转到pier-client-fabric项目路径下
$ git clone https://github.com/meshplus/pier-client-fabric.git && cd pier-client-fabric
$ cp ./config $HOME/.pier1/fabric
```

插件配置主要文件有

```
├── fabric
│   ├── config.yaml
|   └── crypto-config/
│   └── fabric.toml
│   └── fabric.validators
```

主要修改Fabric网络配置，验证证书，跨链合约设置：

- Fabric网络配置

```shell
# 复制你所部署的Fabric所产生的crypto-config文件夹
$ cp -r /path/to/crypto-config $HOME/.pier1/fabric/

# 复制Fabric上验证人证书
$ cp $HOME/.pier1/fabric/crypto-config/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/msp/signcerts/peer1.org2.example.com-cert.pem $HOME/.pier1/fabric/fabric.validators
```

- 修改 config.yaml 文件

`config.yaml`文件记录的Fabric网络配置，需要使用绝对路径，把所有的路径都修改为 `crypto-config`文件夹所在的绝对路径。

```
path: {CONFIG_PATH}/fabric/crypto-config => path: $HOME/.pier/fabric/crypto-config
```

同时需要修改所有的Fabric 的IP地址，如：

```
url: grpcs://localhost:7050 => url: grpcs://10.1.16.48:7050
```

- 修改跨链合约相关配置

修改 fabric.toml 文件

```toml
addr = "localhost:7053" # 若Fabric部署在服务器上，该为服务器地址
event_filter = "interchain-event-name"
username = "Admin"
ccid = "broker" # 若部署跨链broker合约名字不是broker需要修改
channel_id = "mychannel"
org = "org2"
```

### 4. 以太坊应用链插件配置

插件配置的模板在`pier-client-ethereum`项目中

```shell
$ mkdir -p $HOME/.pier2/plugins

# 转到下载解压后的二进制文件夹
$ cp ./pier-binary/eth-client.so $HOME/.pier2/plugins/eth-client.so

# 转到pier-client-ethereum项目路径下
$ git clone https://github.com/meshplus/pier-client-ethereum.git && cd pier-client-ethereum
$ cp ./config $HOME/.pier2/ether
```

插件配置主要文件有

```
ether
├── account.key
├── broker.abi
├── ether.validators
├── ethereum.toml
└── password
```

主要修改以太坊的账号信息，跨链合约设置：

- 账户配置

  将你在以太坊私链上有足够余额的账号私钥文件替换 `account.key` 文件，文件内容格式如下：

```shell
{"address":"20f7fac801c5fc3f7e20cfbadaa1cdb33d818fa3","crypto":{"cipher":"aes-128-ctr","ciphertext":"8b60c0de32b4f06cc14932cdff70b3c01ecc21c25fcbfd0ad582f2bd19af7c30","cipherparams":{"iv":"29738398b8a2df14f74a81a4b547e3f6"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"2a26869c26b03202cc3e7b136742ac3c1793675912aa33a8adab1b15749b0dcd"},"mac":"0a4a5234df68e7fe014e6c09c1c6de36c3be08081f713d39512bdf82c9a6500c"},"id":"1f4f2381-aff1-40e0-9819-6c93f0645a6b","version":3}
```

​	如果你的账号有设置密码才能解冻账号，在password文件中写入密码即可

```
# 填入你的密码
123
```

- 修改 ethereum.toml 文件

  ```toml
  [Ether]
  # 私链的IP地址和提供websocket服务的端口
  addr = "ws://localhost:8546" 
  # 自定义的应用链名称
  name = "ether"
  # 部署在私链上的Broker合约地址
  contract_address = "0xD3880ea40670eD51C3e3C0ea089fDbDc9e3FBBb4"
  # Broker合约对应的ABI文件
  abi_path = "broker.abi"
  # 有足够余额的账号秘钥文件名称
  key_path = "account.key"
  # 解冻该账户的密码
  password = "password"
  ```

### 5. 启动跨链网关

运行下面的命令：

```shell
# 转到二进制下载解压目录并执行
# 启动Fabric的跨链网关

# 先向中继链注册应用链，其中${APPCHAIN_NAME} 为自定义的名称，法律链等都可以
pier --repo=$HOME/.pier1 appchain register --name=${APPCHAIN_NAME} --type=fabric --validators=$HOME/.pier1/fabric/fabric.validators --desc="fabric appchain for test" --version=1.4.3
# 向中继链注册验证规则
pier --repo=$HOME/.pier1 rule deploy --path=$HOME/.pier1/validating.wasm

$ pier --repo $HOME/.pier1 start 

# 启动以太坊的跨链网关
# 先向中继链注册应用链，其中${APPCHAIN_NAME} 为自定义的名称，法律链等都可以
pier --repo=$HOME/.pier2 appchain register --name=${APPCHAIN_NAME} --type=ethereum --validators=$HOME/.pier2/ether/ether.validators --desc="ethereum appchain for test" --version=1.9.3
# 向中继链注册验证规则
pier --repo=$HOME/.pier2 rule deploy --path=$HOME/.pier2/validating.wasm

$ pier --repo $HOME/.pier2 start 
```

根据跨链网关打印的相应日志信息可以判断跨链网关的运行情况。

## 发送跨链交易

按照上面的步骤，在你的两条Fabric上应该已经部署了一个 `broker` 合约，一个`transfer `合约。

`broker `合约管理所有从应用链上发出的跨链交易，并抛出跨链事件。transfer合约是应用合约，负责具体的应用逻辑。该合约实现了一个简单的模拟账户，能够记录账号名和对应的账户余额，作为一个简单的应用合约。

由于不是所有的应用合约都能随便调用broker合约的接口进行跨链，因此 broker 要对进行跨链的应用合约进行权限控制，我们利用broker来审核应用合约地址的方式进行控制。

### broker 审核 应用合约

在chaincode的broker合约中，也提供了一个 `audit` 方法进行审核，函数的参数依次为应用合约（在此样例中就是 `transfer` 合约）所在的通道名，应用合约 chaincode名称，`status` 是最后的审核状态，1表示审核通过，2表示拒绝。如 `audit(mychannel,  transfer, 1) `。

```go
func (broker *Broker) audit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	channel := args[0]
	chaincodeName := args[1]
	status := args[2]	
	...
}
```

调用此方法即可进行审核。

### 跨链准备

进行跨链转账之前，我们先在样例合约中设置一个账户，并给定余额。

在chaincode 的 transfer 合约中，我们提供了 `setBalance` 方法设置账户余额。`name` 是账户名称，`amount` 是账户余额。如 `setBalance(Bob, 10000) `。

```go
func (t *Transfer) setBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	...
	name := args[0]
	amount := args[1]
	...
}
```

同样的，也提供了 `getBalance` 方法来查询账户余额，id 是账户名称。

```go
func (t *Transfer) getBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	...
	name := args[0]
	...
}
```

设置账户余额之后，再调用该接口查询一下设置的账户余额值，以确保操作成功。

### 调用跨链转账接口

要进行跨链操作的话，需要通过应用合约调用 broker 合约相应跨链接口。我们提供的样例合约 transfer 中已经写好了方法来进行跨链转账。

```go
func (t *Transfer) transfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	...
	destChainID := args[0]
	destAddr := args[1]
	sender := args[2]
	receiver := args[3]
	amount := args[4]
}
```

其中 `destChainID` 是跨链目的链的 ID，可以通过下面的命令，根据配置目录获相应取应用链的ID：

```shell
$ pier --repo ~/.pier1 id # 如果你在其他路径下初始化的跨链网关，--repo指定的路径相应替换
```

`destAddr ` 是目的链上的合约唯一ID，对于Fabric上的 chaincode（没有合约地址的概念），使用 "{channel}&​{chaincodename}"（比如将 transfer的chaincode部署在你的 mychannel上，chaincode部署的名称是 transfer，那么这个chaincode的唯一ID为(“mychannel&transfer" )；

`sender` 为来源链上账户的名称（按照上面的设置为Alice或者Bob）;

`receiver` 为目的链上的账户名称（按照上面的设置为Alice或者Bob）;

`amount` 为要转账的金额。

**FabricA -> FabricB的跨链转账：**

调用 chaincode 的 transfer合约的transfer方法，例如：

```shell
#                destChainID(FabricB应用链的ID              destAddr        sender receiver amount
transfer(0x9f5cf4b97965ababe19fcf3f1f12bb794a7dc279, "mychannel&transfer", Alice,    Bob,    1)
```

**提示：具体的用来调用Fabric合约的工具我们不做要求，可以根据你的使用习惯来指定。Fabric部署和调用合约来说，可以参考Fabric官方教程，也可以使用其他开源的工具。**

