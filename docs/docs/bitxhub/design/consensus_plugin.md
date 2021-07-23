# 共识算法插件
在本教程中，你将构建一个完整功能的共识服务。过程中能学习基本的概念和具体使用细节。该示例将展示如何快速、轻松地**接入自己的共识算法到BitXHub中来**。

## 1. 开发要求
- 安装 [__go1.13+__](https://golang.org/doc/install)

- 设置好$GOPATH等环境

## 2. 准备

为了更加便捷的开发共识服务接入到`BitXHub`中来，我们提供了一些接口和参数。

### 2.1 Order接口

我们规定了下面一些必要的接口，这些接口是`BitXHub`与共识服务的交互接口。

```go
type Order interface {
//开启共识服务
Start() error

//停止共识服务，关闭共识服务资源
Stop()

//交易发送到共识服务，共识服务将处理并打包该交易
Prepare(tx *pb.Transaction) error

//返回打包的区块
Commit() chan *pb.Block

//从网络接收到的共识消息
Step(ctx context.Context, msg []byte) error

//集群中产生了新Leader，系统通过该接口判断共识服务是否正常
Ready() bool

//系统会通知该接口已经持久化的区块
ReportState(height uint64, hash types.Hash)

//集群中可以正常工作的最少节点数量，如在raft中要求正常节点数是N/2+1
Quorum() uint64

//获取账户最新的nonce
GetPendingNonceByAccount(account string) uint64

//从共识删除指定的节点
DelNode(delID uint64) error
}
```

### 2.2 Config参数

我们规定了下面一些参数可以从BitXHub传递给共识服务。

```go
type Config struct {
   Id                 uint64 //节点ID
   RepoRoot           string //根路径
   StoragePath        string //存储文件路径
   PluginPath         string //插件文件路径
   PeerMgr            peermgr.PeerManager //网络模块组件
   PrivKey            crypto.PrivateKey //节点私钥
   Logger             logrus.FieldLogger //日志组件
   Nodes              map[uint64]types.Address //集群节点网络地址
   Applied            uint64 //当前区块高度
   Digest             string //当前区块哈希
   GetTransactionFunc func(hash types.Hash) (*pb.Transaction, error) // 获取已经持久化的交易函数
   GetBlockByHeight func(height uint64) (*pb.Block, error) // 获取指定高度的区块
   GetAccountNonce  func(address *types.Address) uint64 // 获取指定账户的nonce
}
```


## 3. 程序目的

本教程以开发一个简单Solo版本的共识算法为例。

### 3.1 开始编写你的程序

首先选择你的工程目录，按照正常的GO程序的流程建立项目

```shell
$ go version // 确认你安装的GO版本
$ mkdir ${YOUR_PROJECT}
$ cd ${YOUR_PROJECT}
$ go mod init
```



### 3.2 Node对象

首先创建一个`node.go`文件，这个文件是共识Plugin的核心和入口，来看看`Node`具体结构

```go
type Node struct {
    ID        uint64
    commitC   chan *pb.CommitEvent         // block channel
    logger    logrus.FieldLogger           // logger
    mempool   mempool.MemPool              // transaction pool
    proposeC  chan *raftproto.RequestBatch // proposed listenReadyBlock, input channel
    stateC    chan *mempool.ChainState
    txCache   *mempool.TxCache // cache the transactions received from api
    batchMgr  *etcdraft.BatchTimer
    lastExec  uint64              // the index of the last-applied block
    packSize  int                 // maximum number of transaction packages
    blockTick time.Duration       // block packed period
    peerMgr   peermgr.PeerManager // network manager
    
    ctx    context.Context
    cancel context.CancelFunc
    sync.RWMutex
}
```

然后应该提供一个`Order`的实例化的接口（类似于构造函数），具体代码如下：

```go
func NewNode(opts ...order.Option) (order.Order, error) {
    //处理Order参数
    config, err := order.GenerateConfig(opts...)
    if err != nil {
        return nil, fmt.Errorf("generate config: %w", err)
    }
    ctx, cancel := context.WithCancel(context.Background())
   
    //读取mempool相关配置，batchTimeout出块时间，memConfig配置
    batchTimeout, memConfig, err := generateSoloConfig(config.RepoRoot)
    mempoolConf := &mempool.Config{
        ID:              config.ID, //节点ID
        ChainHeight:     config.Applied, //当前区块高度
        Logger:          config.Logger,  //日志组件
        StoragePath:     config.StoragePath, //存储路径
        GetAccountNonce: config.GetAccountNonce, // 获取账户nonce
        
        BatchSize:      memConfig.BatchSize, //区块最大交易数
        PoolSize:       memConfig.PoolSize, //交易池容纳最大交易数
        TxSliceSize:    memConfig.TxSliceSize, //单次广播最大交易数  
        TxSliceTimeout: memConfig.TxSliceTimeout, //单次广播交易间隔
    }
    batchC := make(chan *raftproto.RequestBatch)
    
    //实例化mempool，mempool的作用是交易排序
    mempoolInst, err := mempool.NewMempool(mempoolConf)
    if err != nil {
        return nil, fmt.Errorf("create mempool instance: %w", err)
    }
    
    //实例化txCache交易池
    txCache := mempool.NewTxCache(mempoolConf.TxSliceTimeout, mempoolConf.TxSliceSize, config.Logger)
    batchTimerMgr := etcdraft.NewTimer(batchTimeout, config.Logger)
    soloNode := &Node{
        ID:       config.ID,
        commitC:  make(chan *pb.CommitEvent, 1024),
        stateC:   make(chan *mempool.ChainState),
        lastExec: config.Applied,
        mempool:  mempoolInst,
        txCache:  txCache,
        batchMgr: batchTimerMgr,
        peerMgr:  config.PeerMgr,
        proposeC: batchC,
        logger:   config.Logger,
        ctx:      ctx,
        cancel:   cancel,
    }
    soloNode.logger.Infof("SOLO lastExec = %d", soloNode.lastExec)
    soloNode.logger.Infof("SOLO batch timeout = %v", batchTimeout)
    return soloNode, nil
}
```

## 3.3 Node主要方法

通过描述Node的主要方法，介绍pending的交易是如何被打包到区块中以及如何与`BitXHub`系统进行交互。

### 3.3.1 Start方法

功能：定时在交易池中扫描交易并出块。

```go
func (n *Node) Start() error {
	//交易池打包区块
    go n.txCache.ListenEvent()
    
    //solo监听新区块
    go n.listenReadyBlock()
    return nil
}

// Schedule to collect txs to the listenReadyBlock channel
func (n *Node) listenReadyBlock() {
	go func() {
		for {
			select {
			case proposal := <-n.proposeC:
				n.logger.WithFields(logrus.Fields{
					"proposal_height": proposal.Height,
					"tx_count":        len(proposal.TxList),
				}).Debugf("Receive proposal from mempool")

				if proposal.Height != n.lastExec+1 {
					n.logger.Warningf("Expects to execute seq=%d, but get seq=%d, ignore it", n.lastExec+1, proposal.Height)
					return
				}
				n.logger.Infof("======== Call execute, height=%d", proposal.Height)
				block := &pb.Block{
					BlockHeader: &pb.BlockHeader{
						Version:   []byte("1.0.0"),
						Number:    proposal.Height,
						Timestamp: time.Now().UnixNano(),
					},
					Transactions: proposal.TxList,
				}
				localList := make([]bool, len(proposal.TxList))
				for i := 0; i < len(proposal.TxList); i++ {
					localList[i] = true
				}
				executeEvent := &pb.CommitEvent{
					Block:     block,
					LocalList: localList,
				}
				n.commitC <- executeEvent
				n.lastExec++
			}
		}
	}()

	for {
		select {
		case <-n.ctx.Done():
			n.logger.Info("----- Exit listen ready block loop -----")
			return

		case txSet := <-n.txCache.TxSetC:
			// start batch timer when this node receives the first transaction
			if !n.batchMgr.IsBatchTimerActive() {
				n.batchMgr.StartBatchTimer()
			}
			if batch := n.mempool.ProcessTransactions(txSet.TxList, true, true); batch != nil {
				n.batchMgr.StopBatchTimer()
				n.proposeC <- batch
			}

      case state := <-n.stateC:
         //每十个区块做一个Check Point
			if state.Height%10 == 0 {
				n.logger.WithFields(logrus.Fields{
					"height": state.Height,
					"hash":   state.BlockHash.String(),
				}).Info("Report checkpoint")
			}
			n.mempool.CommitTransactions(state)

		case <-n.batchMgr.BatchTimeoutEvent():
			n.batchMgr.StopBatchTimer()
			n.logger.Debug("Batch timer expired, try to create a batch")
			if n.mempool.HasPendingRequest() {
				if batch := n.mempool.GenerateBlock(); batch != nil {
					n.postProposal(batch)
				}
			} else {
				n.logger.Debug("The length of priorityIndex is 0, skip the batch timer")
			}
		}
	}
}
```

### 3.3.2 Stop方法

功能：停止共识，释放共识相关资源。

```go
func (n *Node) Stop() {
   n.cancel()
}
```

### 3.3.3 Prepare方法

功能：从`BitXHub`系统中传入交易，收集进交易池。

```go
func (n *Node) Prepare(tx *pb.Transaction) error {
   //判断当前共识是否正常
	if err := n.Ready(); err != nil {
		return err
   }
   //交易进入交易池
	n.txCache.RecvTxC <- tx
	return nil
}

```



### 3.3.4 Commit方法

功能：返回新区块的`channel`。

```go
func (n *Node) Commit() chan *pb.Block {
   return n.commitC
}
```



### 3.3.5 Step方法

功能：通过该接口接收共识的网络消息。

```go
//由于示例是Solo的版本，故具体不实现该方法
func (n *Node) Step(ctx context.Context, msg []byte) error {
   return nil
}
```

### 3.3.6 Ready方法

功能：判断共识是否完成，Leader是否完成选举。

```go
//由于示例是Solo的版本，单节点直接返回True
func (n *Node) Ready() bool {
   return true
}
```

### 3.3.7 ReportState方法

功能：新区块被持久化后，`BitXHub`会调用该接口通知共识服务

```go
func (n *Node) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {
	state := &mempool.ChainState{
		Height:     height,
		BlockHash:  blockHash,
		TxHashList: txHashList,
   }
   //结合listenReadyBlock()，账本落盘区块后通知共识清理相关资源
	n.stateC <- state
}
```

### 3.3.8 Quorum方法

功能：集群中可以正常工作的最少节点数量（比如在raft中要求正常节点数是N/2+1）。

```go
//由于示例是Solo的版本，直接返回1
func (n *Node) Quorum() uint64 {
   return 1
}
```

### 3.3.9 GetPendingNonceByAccount方法

功能：获取指定账户最新的nonce值。

```go
func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	// 直接从交易池中获取当前账户的最新nonce，由于交易池中可能有该账号的多笔交易，
	// 需要把符合条件的nonce刷选出。
    return n.mempool.GetPendingNonceByAccount(account)
}
```

### 3.3.10 DelNode方法

功能：删除指定的共识节点。

```go
//由于示例是Solo的版本，直接返回nil
func (n *Node) DelNode(delID uint64) error {
    return nil
}
```



## 4. 接入共识算法

### 4.1 配置文件

可以通过配置`order.toml`文件，自定义你的共识算法。

```none
[solo]
batch_timeout = "0.3s"  # Block packaging time period.

   [solo.mempool]
        batch_size          = 200   # How many transactions should the primary pack.
        pool_size           = 50000 # How many transactions could the txPool stores in total.
        tx_slice_size       = 10    # How many transactions should the node broadcast at once
        tx_slice_timeout    = "0.1s"  # Node broadcasts transactions if there are cached transactions, although set_size isn't reached yet
```


### 4.2 项目结构

该项目为BitXHub提供共识算法的插件化，具体项目结构如下：

```text
./
├── Makefile //编译文件
├── README.md
├── build
│ └── rbft.so //编译后的共识算法二进制插件
├── go.mod
├── go.sum
├── order.toml //共识配置文件
└── rbft //共识算法代码
    ├── config.go
    ├── node.go
    └── stack.go
```

其中注意在`go.mod`中需要引用BitXHub项目源码，需要让该插件项目与BitXHub在同一目录下（建议在$GOPATH路径下）。

```none
replace github.com/meshplus/bitxhub => ../bitxhub/
```

### 4.3 编译Plugin

我们采用GO语言提供的插件模式，实现`BitXHub`对于Plugin的动态加载。

编写`Makefile`编译文件：

```shell
SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
GO  = GO111MODULE=on go
plugin:
   @mkdir -p build
   $(GO) build --buildmode=plugin -o build/rbft.so rbft/*.go
```

运行下面的命令，能够得到 `rbft.so`文件。

```shell
$ make plugin
```

修改节点的`bitxhub.toml`

```none
[order]
  plugin = "plugins/rbft.so"
```

将你编写的动态链接文件和`order.toml`文件，分别放到节点的plugins文件夹和配置文件下。

```text
./
├── api
├── bitxhub.toml
├── certs
│ ├── agency.cert
│ ├── ca.cert
│ ├── node.cert
│ └── node.priv
├── key.json
├── logs
├── network.toml
├── order.toml //共识算法配置文件
├── plugins
│ ├── rbft.so //共识算法插件
├── start.sh
└── storage
```

结合我们提供的`BitXHub`中继链，就能接入到跨链平台来。