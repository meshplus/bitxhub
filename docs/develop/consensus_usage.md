# 共识算法插件化使用文档

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

	//系统会通知该接口已经持久化的区块，
	ReportState(height uint64, hash types.Hash)

	//集群中可以正常工作的最少节点数量，如在raft中要求正常节点数是N/2+1
	Quorum() uint64
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
   GetTransactionFunc func(hash types.Hash) (*pb.Transaction, error) //获取已经持久化的交易函数
   GetChainMetaFunc   func() *pb.ChainMeta //获取链的元信息函数，其中包括当前区块高度和区块哈希
}
```

### 2.3 Filter过滤器

为了更加方便过滤重复交易，我们提供了布隆过滤器的组件。

```go
const (
   filterDbKey = "bloom_filter"
   m = 10000000 //filter的字节位数
   k = 4        //计算hash的次数
)

type ReqLookUp struct {
   filter       *bloom.BloomFilter //bloom过滤器
   storage storage.Storage //leveldb存储
   b       bytes.Buffer //filter缓存
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
   sync.RWMutex         
   height             uint64             // 当前区块高度
   pendingTxs         *list.List         //交易池
   commitC            chan *pb.Block     //区块channel
   logger             logrus.FieldLogger //日志
   reqLookUp          *order.ReqLookUp   //bloom过滤器
   getTransactionFunc func(hash types.Hash) (*pb.Transaction, error) //获取持久化的交易函数
   packSize  int           //出块的最大交易数量
   blockTick time.Duration //出块间隔

   ctx    context.Context 
   cancel context.CancelFunc 
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

   //创建leveldb，用于存储bloom filter
   storage, err := leveldb.New(config.StoragePath)
   if err != nil {
      return nil, fmt.Errorf("new leveldb: %w", err)
   }

   //创建bloom filter，用于过滤重复交易
   reqLookUp, err := order.NewReqLookUp(storage)
   if err != nil {
      return nil, fmt.Errorf("new bloom filter: %w", err)
   }

   //创建node的上下文
   ctx, cancel := context.WithCancel(context.Background())
   return &Node{
      height:             config.Applied,//区块的当前高度
      pendingTxs:         list.New(),
      commitC:            make(chan *pb.Block, 1024),
      packSize:           500, //出块的最大交易数量，可配置在order.toml中
      blockTick:          500 * time.Millisecond, //出块时间间隔，可配置在order.toml中
      reqLookUp:          reqLookUp,
      getTransactionFunc: config.GetTransactionFunc,
      logger:             config.Logger,
      ctx:                ctx,
      cancel:             cancel,
   }, nil
}
```

### 3.3 Node主要方法

通过描述Node的主要方法，介绍pending的交易是如何被打包到区块中以及如何与`BitXHub`系统进行交互。

#### 3.3.1 Start方法

功能：定时在交易池中扫描交易并出块。

```go
func (n *Node) Start() error {
   go n.execute()
   return nil
}

func (n *Node) execute(){
   //开启定时器，定时扫描交易池中是否存在pending的交易
   ticker := time.NewTicker(n.blockTick)
   defer ticker.Stop()

   for {
      select {
      case <-ticker.C:
         n.Lock()
         l := n.pendingTxs.Len()
         if l == 0 {
            n.Unlock()
            continue
         }

         var size int
         if l > n.packSize {
            size = n.packSize
         } else {
            size = l
         }

	 //打包交易
         txs := make([]*pb.Transaction, 0, size)
         for i := 0; i < size; i++ {
            front := n.pendingTxs.Front()
            tx := front.Value.(*pb.Transaction)
            txs = append(txs, tx)
            n.pendingTxs.Remove(front)
         }

	//区块高度+1
         n.height++
         n.Unlock()
	 //区块出块
         block := &pb.Block{
            BlockHeader: &pb.BlockHeader{
               Version:   []byte("1.0.0"),
               Number:    n.height,
               Timestamp: time.Now().UnixNano(),
            },
            Transactions: txs,
         }
         n.commitC <- block
      case <-n.ctx.Done():
         ticker.Stop()
      }
   }
    return nil
}
```

#### 3.3.2 Stop方法

功能：停止共识，释放共识相关资源。

```go
func (n *Node) Stop() {
   n.cancel()
}
```

#### 3.3.3 Prepare方法

功能：从`BitXHub`系统中传入交易，收集进交易池。

```go
func (n *Node) Prepare(tx *pb.Transaction) error {
   hash := tx.TransactionHash
   //检查交易是否存在，防止交易二次打包
   if ok := n.reqLookUp.LookUp(hash.Bytes()); ok {
      if tx, _ := n.getTransactionFunc(hash); tx != nil {
         return nil
      }
   }

   //交易进入交易池
   n.pushBack(tx)
   return nil
}
```



#### 3.3.4 Commit方法

功能：返回新区块的`channel`。

```go
func (n *Node) Commit() chan *pb.Block {
   return n.commitC
}
```



#### 3.3.5 Step方法

功能：通过该接口接收共识的网络消息。

```go

//由于示例是Solo的版本，故具体不实现该方法
func (n *Node) Step(ctx context.Context, msg []byte) error {
   return nil
}
```

#### 3.3.6 Ready方法

功能：判断共识是否完成，Leader是否完成选举。

```go

//由于示例是Solo的版本，单节点直接返回True
func (n *Node) Ready() bool {
   return true
}
```

#### 3.3.7 ReportState方法

功能：新区块被持久化后，`BitXHub`会调用该接口通知共识服务

```go
func (n *Node) ReportState(height uint64, hash types.Hash) {
   //每一个新区块被存储后，持久化bloom filter
   if err := n.reqLookUp.Build(); err != nil {
      n.logger.Errorf("bloom filter persistence error：", err)
   }
   //每十个区块做一个Check Point
   if height%10 == 0 {
      n.logger.WithFields(logrus.Fields{
         "height": height,
         "hash":   hash.ShortString(),
      }).Info("Report checkpoint")
   }
}
```

#### 3.3.8 Quorum方法

功能：集群中可以正常工作的最少节点数量（比如在raft中要求正常节点数是N/2+1）。

```go
//由于示例是Solo的版本，直接返回1
func (n *Node) Quorum() uint64 {
   return 1
}
```



## 4. 接入共识算法

### 4.1 配置文件

可以通过配置`order.toml`文件，自定义你的共识算法。

```none
[solo]
[solo.tx_pool]
   pack_size = 500 # 出块最大交易数
   block_tick = "500ms" # 出块间隔
```

### 4.2 插件化
[共识算法插件化设计文档](https://github.com/meshplus/bitxhub/wiki/%E5%85%B1%E8%AF%86%E7%AE%97%E6%B3%95%E6%8F%92%E4%BB%B6%E5%8C%96%E8%AE%BE%E8%AE%A1%E6%96%87%E6%A1%A3)


