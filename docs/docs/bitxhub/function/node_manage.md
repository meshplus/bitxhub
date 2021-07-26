# 节点管理
中继链提供对中继链自身节点的增删管理功能

## 1 概述
中继链自身节点有共识节点和审计节点两种，中继链初始启动的节点都是共识节点。  

- 共识节点：参与共识
- 审计节点：不参与共识，只能同步数据


⚠️注意：
1. 关于共识节点，目前中继链只支持rbft共识算法的节点增删管理  
2. rbft共识只在BitXHub商业版本中支持，启动商业版本的BitXHub需要获取相应的LICENSE，并将LICENSE放在每个BitXHub节点的根目录下  
3. 关于审计节点,目前中继链支持注册注销的简单操作，但实际的审计节点有待后续实现  

（下文中的使用方法主要介绍rbft共识算法下的共识节点增删）

## 2 增加节点
### 2.1 功能介绍
目前中继链支持在使用rbft共识算法的情况下增加共识节点，新增过程中节点的一般状态转换如下：  
`unavailable` --> `registering` --> `available`  
新增节点有以下几点需要注意：  
（1）节点只能逐个的增加，即上一个节点的新增提案完成才可以继续增加下一个节点；  
（2）共识节点增加时需要提供一个vpNodeId参数，该参数按可用共识节点的序号逐个递增，比如：当前有4个可用共识节点，那么这4个共识节点的vpNodeId一定分别是1、2、3、4，新增共识节点的id必须为5。

### 2.2 使用方法

#### 第一步：启动rbft共识的BitXHub
bitxhub的默认共识类型是raft，如果要启动rbft共识的bitxhub，需要修改每个节点的bitxhub.toml配置文件，将共识类型参数改为rbft：
```shell script
[order]
  type = "rbft"
```
此外，rbft是在bitxhub商业版本中支持的，启动商业版本的bitxhub的命令如下：
```shell script
$ make cluster TAGS=ent
```

#### 第二步：启动新节点
新节点的启动需要先准备相关配置文件然后再启动，具体流程如下：  
（1）线下向CA证书的代理机构请求准入证书。bitxhub命令行提供了生成证书功能，命令如下：
```shell script
// 生成自己的私钥
// --name：指定私钥名称，默认使用node
// --target：指定生成私钥位置
$ bitxhub cert priv gen --name node --target .

// 获取私钥的csr文件
// --key：指定新节点私钥的路径
// --org：指定新节点的组织名称
// --target：指定生成文件路径
$ bitxhub cert csr --key ./node.priv --org Node --target .

// 代理结构颁发证书
// --csr：指定新节点csr文件的路径
// --is_ca：指定当前是否是生成ca根证书（不是）
// --key：指定代理机构私钥的路径
// --cert：指定代理机构证书的路径
// --target：指定生成文件路径
$ bitxhub cert issue --csr node.csr --is_ca false --key agency.priv --cert agency.cert --target .
```
（2）线下准备新节点的私钥（这个私钥与上文中用户获取证书的私钥不同，此私钥与节点network.toml中的account对应）。bitxhub命令行提供相关功能，命令如下：
```shell script
// --target：指定生成私钥位置
// --passwd：指定私钥密码
$ bitxhub key gen --target tmp --passwd bitxhub

```
（3）准备节点配置文件模板，命令如下：
```shell script
// --repo：指定新节点配置文件路径
$ bitxhub --repo ./node5 init
```
（4）将相关私钥证书拷贝到相应路径，包括新节点证书私钥、新节点证书、agency证书、ca证书、节点私钥、LICENSE，命令如下：
```shell script
$ cp ./node.cert ./node5/certs/
$ cp ./node.priv ./node5/certs/
$ cp ~/work/bitxhub/scripts/certs/node1/certs/agency.cert ./node5/certs/
$ cp ~/work/bitxhub/scripts/certs/node1/certs/ca.cert ./node5/certs/
$ cp ./key.json ./node5
$ cp LICENSE ./node5
```
（5）修改bitxhub.toml中端口信息和共识类型，需要修改的部分修改后如下：
```shell script
[port]
  jsonrpc = 8885
  grpc = 60015
  gateway = 9095
  pprof = 53125
  monitor = 40015

[order]
  type = "raft"
```
（6）在network.toml中添加新节点的网络信息，bitxhub命令行提供了根据相关私钥查看节点pid和account信息的功能
```shell script
// 查看节点pid信息
$ bitxhub cert priv pid --path ./node5/certs/node.priv
Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm

// 查看节点account信息
$ bitxhub key show --path ./node5/key.json
private key: 18fcb8f87c5ed7bc67bd0fc40248ecea7e65d7d624cc588ed5903e6b2caca339
public key: 04ffa0e8a1353aa36ef7c4e986ec87d01e4692a7fc1b511d2f7e56a8573ebc54f76f7efa3e5eca17655a2c91d57fa060b30e395500c1bff7b3933477678d78d93c
address: 0x9E887Aa2e8009C6c4b4aF7792e0afe71f0Dc1d64

// 修改network.toml，将新节点网络信息添加到network.toml末尾，并修改id和new参数，修改后netwokr.toml如下：
id = 5 # self id
n = 4 # the number of primary vp nodes
new = true # track whether the node is a new node

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

[[nodes]]
account = "0x9E887Aa2e8009C6c4b4aF7792e0afe71f0Dc1d64"
hosts = ["/ip4/127.0.0.1/tcp/4005/p2p/"]
id = 5
pid = "Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm"
```
（7）启动新节点，命令如下：
```shell script
$ bitxhub --repo ./node5 start
```
新节点后启动后会一直尝试连接bitxhub节点，但由于节点没有注册会一直连接失败，原节点也会一直拒绝新节点的连接。  
- 新节点会打印出如下日志：
```shell script
ERRO[2021-07-23T14:36:13.300] Connect failed                                error="dial backoff" module=p2p node=1
ERRO[2021-07-23T14:36:13.326] Connect failed                                error="dial backoff" module=p2p node=4
ERRO[2021-07-23T14:36:13.339] Connect failed                                error="dial backoff" module=p2p node=2
ERRO[2021-07-23T14:36:14.005] Connect failed                                error="dial backoff" module=p2p node=3
```
- 原节点会打印出如下日志：
 ```shell script
INFO[2021-07-23T14:35:12.804] Intercept a connection with an unavailable node, peer.Pid: Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm, peer.Id: 5, peer.status: registering  module=p2
```

#### 第三步：注册新节点
中继链管理员注册新节点的命令如下：
```shell script
// --repo：指定中继链管理员私钥的路径
// --pid：指定新节点pid信息
// --account：指定新节点account信息
// --type：指定新节点类型
// --id：指定共识节点的id
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance node register --pid Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm --account 0x9E887Aa2e8009C6c4b4aF7792e0afe71f0Dc1d64 --type vpNode --id 5
proposal id is 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0
```
根据上述命令执行的打印信息可以看到注册新节点的提案号为0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0

#### 第四步：中继链管理员投票
中继链管理员进行投票治理，默认四个管理员的情况下需要三个管理员投赞成票提案可通过，命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定治理提案的id，0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0
// --info：指定投票内容是approve或是reject
// --reason：指定投票理由，可自定义
$ bitxhub --repo ~/work/bitxhub/scripts/build/node4 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0 --info apprpve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node2 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0 --info approve --reason reason
```

投票通过注册成功后，新节点将会自动与原节点连接成功，等待片刻（新节点需要同步区块）后新节点将会成功打印出bitxhub的logo:
```shell script
=======================================================
    ____     _    __    _  __    __  __            __
   / __ )   (_)  / /_  | |/ /   / / / /  __  __   / /_
  / __  |  / /  / __/  |   /   / /_/ /  / / / /  / __ \
 / /_/ /  / /  / /_   /   |   / __  /  / /_/ /  / /_/ /
/_____/  /_/   \__/  /_/|_|  /_/ /_/   \__,_/  /_.___/

=======================================================
```

#### 第五步：查看中继链节点信息 
中继链提供查看节点信息的命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --type：指定中继链节点类型
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance node all --type vpNode
```
执行结果如下，可以看到新增节点已经是可用状态
```shell script
NodePid                                         type    VpNodeId  Account                                     Status
-------                                         ----    --------  -------                                     ------
QmQW3bFn8XX1t4W14Pmn37bPJUpUVBrBjnPuBZwPog3Qdy  vpNode  4         0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8  available
QmbmD1kzdsxRiawxu7bRrteDgW1ituXupR8GH6E2EUAHY4  vpNode  2         0x79a1215469FaB6f9c63c1816b45183AD3624bE34  available
QmXi58fp9ZczF3Z5iz1yXAez3Hy5NYo1R8STHWKEM9XnTL  vpNode  1         0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013  available
QmQUcDYCtqbpn5Nhaw4FAGxQaSSNvdWfAFcpQT9SPiezbS  vpNode  3         0x97c8B516D19edBf575D72a172Af7F418BE498C37  available
Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm  vpNode  5         0x9E887Aa2e8009C6c4b4aF7792e0afe71f0Dc1d64  available
```

## 3 删除节点
### 3.1 功能介绍
节点删除时的一般状态转换如下：  
`available` --> `logouting` --> `unavailable`  
删除节点有以下几点注意：  
（1）节点删除后可以重新注册，不做类似应用链注销后不可重复注册的限制；  
（2）初始节点不可以删除；  
（3）删除共识节点时目前只支持删除按vpNodeId排序的最后一个节点，比如当前有5个可用共识节点，这5个共识节点的id一定分别是1、2、3、4、5，那么只能删除id为5的共识节点（这一点后续可能会改进）  
（4）cluster模式删除节点时需要保证共识节点个数不少于4个

### 3.2 使用方法

#### 第一步：中继链管理员删除节点
中继链管理员除节点的命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --pid：待删除节点的pid
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance node logout --pid Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm
proposal id is 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1
```
根据上述命令执行的打印信息可以看到删除节点的提案号为0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1  

#### 第二步：中继链管理员投票
中继链管理员进行投票治理，默认四个管理员的情况下需要三个管理员投赞成票提案可通过，命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定治理提案的id，0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1
// --info：指定投票内容是approve或是reject
// --reason：指定投票理由，可自定义
$ bitxhub --repo ~/work/bitxhub/scripts/build/node4 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1 --info apprpve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node2 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1 --info approve --reason reason
```

投票通过后，被删除的节点会自动被停掉，打印出的日志如下：
```shell script
INFO[2021-07-23T15:10:48.190] Delete node [ID: 5, peerInfo: id:5 pid:"Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm" account:"0x9E887Aa2e8009C6c4b4aF7792e0afe71f0Dc1d64" hosts:"/ip4/127.0.0.1/tcp/4005/p2p/" ]   module=p2p
INFO[2021-07-23T15:10:48.451] Replica 5 need to process seqNo 1 as a null request  module=order
INFO[2021-07-23T15:10:48.451] ======== THIS NODE WILL STOP IN 3 SECONDS     module=order
INFO[2021-07-23T15:10:48.468] Replica 5 persist view=9/N=4 after updateN    module=order
INFO[2021-07-23T15:10:48.468] ======== Replica 5 finished updateN, primary=2, n=4/f=1/view=9/h=0  module=order
INFO[2021-07-23T15:10:51.460] Transaction cache stopped!                    module=order
INFO[2021-07-23T15:10:51.460] RBFT stopping...                              module=order
INFO[2021-07-23T15:10:51.461] ======== RBFT stopped!                        module=order
INFO[2021-07-23T15:10:51.466] Disconnect peer [ID: 1, Pid: QmXi58fp9ZczF3Z5iz1yXAez3Hy5NYo1R8STHWKEM9XnTL]  module=p2p
INFO[2021-07-23T15:10:51.466] Disconnect peer [ID: 2, Pid: QmbmD1kzdsxRiawxu7bRrteDgW1ituXupR8GH6E2EUAHY4]  module=p2p
INFO[2021-07-23T15:10:51.466] Disconnect peer [ID: 3, Pid: QmQUcDYCtqbpn5Nhaw4FAGxQaSSNvdWfAFcpQT9SPiezbS]  module=p2p
INFO[2021-07-23T15:10:51.466] Disconnect peer [ID: 4, Pid: QmQW3bFn8XX1t4W14Pmn37bPJUpUVBrBjnPuBZwPog3Qdy]  module=p2p
INFO[2021-07-23T15:10:51.466] ======== THIS NODE HAS BEEN DELETED!!!        module=order
```

## 4 其他功能
中继链还提供了其他查询节点信息的功能。
- 查询中继链所有指定类型节点，默认查询共识节点，命令如下：
```shell script
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance node all
```
- 查询中继链某个节点状态，命令如下：
```shell script
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance node status --pid Qmb6ZaaCw6dYPL7ifLAG2gugXf61jo9q4eEfQ3USNBaubm
```