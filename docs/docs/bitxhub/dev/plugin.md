# 应用链插件编写

在本教程中，你将构建一个完整功能的 Plugin。过程中能学习基本的概念和具体使用细节。该示例将展示如何快速、轻松地接入自己的区块链到跨链平台中来。

如果你需要接入自己开发的区块链到BitXHub跨链平台来的话，可以根据你们自己的区块链来定制开发Plugin，通过跨链网关来加入到跨链平台。

## 开发要求

- 安装 [__go1.13+__](https://golang.org/doc/install)

- 设置好$GOPATH等环境

## 教程章节

1. 重要概念

2. Plugin接口

3. 程序目标

4. 开始编写程序

5. 编译你的Plugin

## 重要概念

在解释具体的接口之前，先明确几个概念：

**跨链请求**：如果有两条区块链A和B，A链需要向B链发起任何操作，需要按照IBTP规则向中继链发出一个请求包，我们称之为跨链请求A->B。

**IBTP包**：满足IBTP的一个package，跨链请求都需要通过IBTP包进行。

**来源链**：在跨链请求A->B中，A即为来源链。

**目的链**：在跨链请求A->B中，B即为目的链。

## Plugin接口

为了更加便捷的开发Plugin接入到Pier中来，我们规定了下面一些必要的接口。

```go
type Client interface {
	
	// 初始化Plugin服务
    Initialize(configPath string, pierID string, extra []byte) error
    
    // 启动Plugin服务的接口
    Start() error

    // 停止Plugin服务的接口
    Stop() error

    // Plugin负责将区块链上产生的跨链事件转化为标准的IBTP格式，Pier通过GetIBTP接口获取跨链请求再进行处理
    GetIBTP() chan *pb.IBTP

    // Plugin 负责执行来源链过来的跨链请求，Pier调用SubmitIBTP提交收到的跨链请求。
    SubmitIBTP(*pb.IBTP) (*pb.SubmitIBTPResponse, error)

    // GetOutMessage 负责在跨链合约中查询历史跨链请求。查询键值中to指定目的链，idx指定序号，查询结果为以Plugin负责的区块链作为来源链的跨链请求。
    GetOutMessage(to string, idx uint64) (*pb.IBTP, error)

    // GetInMessage 负责在跨链合约中查询历史跨链请求。查询键值中from指定来源链，idx指定序号，查询结果为以Plugin负责的区块链作为目的链的跨链请求。
    GetInMessage(from string, idx uint64) ([][]byte, error)

    // GetInMeta 是获取跨链请求相关的Meta信息的接口。以Plugin负责的区块链为目的链的一系列跨链请求的序号信息。如果Plugin负责A链，则可能有多条链和A进行跨链，如B->A:3; C->A:5。返回的map中，key值为来源链ID，value对应该来源链已发送的最新的跨链请求的序号，如{B:3, C:5}。
    GetInMeta() (map[string]uint64, error)

    // GetOutMeta 是获取跨链请求相关的Meta信息的接口。以Plugin负责的区块链为来源链的一系列跨链请求的序号信息。如果Plugin负责A链，则A可能和多条链进行跨链，如A->B:3; A->C:5。返回的map中，key值为目的链ID，value对应已发送到该目的链的最新跨链请求的序号，如{B:3, C:5}。
    GetOutMeta() (map[string]uint64, error)

    // GetCallbackMeta 是获取跨链请求相关的Meta信息的接口。以Plugin负责的区块链为来源链的一系列跨链请求的序号信息。如果Plugin负责A链，则A可能和多条链进行跨链，如A->B:3; A->C:5；同时由于跨链请求中支持回调操作，即A->B->A为一次完整的跨链操作，我们需要记录回调请求的序号信息，如A->B->:2; A->C—>A:4。返回的map中，key值为目的链ID，value对应到该目的链最新的带回调跨链请求的序号，如{B:2, C:4}。（注意 CallbackMeta序号可能和outMeta是不一致的，这是由于由A发出的跨链请求部分是没有回调的）
    GetCallbackMeta() (map[string]uint64, error)

    // CommitCallback 执行完IBTP包之后进行一些回调操作。
    CommitCallback(ibtp *pb.IBTP) error

    // GetReceipt 获取一个已被执行IBTP的回执
    GetReceipt(ibtp *pb.IBTP) (*pb.IBTP, error)

    // Name 描述Plugin负责的区块链的自定义名称，一般和业务相关，如司法链等。
    Name() string

    // Type 描述Plugin负责的区块链类型，比如Fabric
    Type() string
}
```

## 程序目的

本教程以开发一个简单的连接Fabric区块链网络的Plugin为例，最终的程序能够实现从负责的区块链获取`Hello World`信息并返回到跨链平台中。

## 开始编写你的程序

首先选择你的工程目录，按照正常的GO程序的流程建立项目

```shell
$ go version // 确认你安装的GO版本
$ mkdir ${YOUR_PROJECT}
$ cd ${YOUR_PROJECT}
$ go mod init exmple/fabric-plugin
```

### Client对象

首先创建一个`client.go`文件，这个文件是Plugin的核心和入口。

在该文件中，应该定义你的Plugin如何获取client 实例，以及如何启动和停止Plugin服务。

现在我们需要创建一个自定义的Client 结构，跨链网关最终拿到的应该是这个结构的一个实例，先来看看这个结构中都需要什么。

首先来看看`Client自定义`具体结构

```go
type Client struct {
   meta     *ContractMeta
   consumer *Consumer
   eventC   chan *pb.IBTP
   pierId   string
   name     string
}

type ContractMeta struct {
   EventFilter string `json:"event_filter"`
   Username    string `json:"username"`
   CCID        string `json:"ccid"`
   ChannelID   string `json:"channel_id"`
   ORG         string `json:"org"`
}
```

- meta：Plugin直接和跨链合约交互，需要保存你的合约的一些基础信息。由于我们需要连接一个Fabric网络，这些Meta信息包括 **Fabric中跨链事件的名称、Fabric中的用户名称、Chaincode合约的名称、你的组织名称Org以及组织所在的channel。**

- consumer：可以理解为Fabric上跨链事件的“监听器”，这个监听器也是一个自定义的结构，具体的结构在后面会详细介绍。

- eventC：为跨链网关提供读取监听到的跨链事件的通道。

- name：自定的区块链的名称。

- pierId：跨链网关注册在跨链平台中后产生的唯一ID，作为应用链的标识。

然后应该提供一个Client的实例化的接口（类似于构造函数），具体代码如下：

```go
func (c *Client) Initialize(configPath, pierId string, extra []byte) error {
    eventC := make(chan *pb.IBTP)
    // read config from files
    fabricConfig, err := UnmarshalConfig(configPath)
    if err != nil {
       return nil, fmt.Errorf("unmarshal config for plugin :%w", err)
    }

    // some basic configs about your chaincode 
    contractmeta := &ContractMeta{
          EventFilter: fabricConfig.EventFilter,
          Username:    fabricConfig.Username,
          CCID:        fabricConfig.CCID,
          ChannelID:   fabricConfig.ChannelId,
          ORG:         fabricConfig.Org,
    }

    // handler for listening on inter-chain events posted on your Fabric
    mgh, err := newFabricHandler(contractmeta.EventFilter, eventC, pierId)
    if err != nil {
        return err
    }

    done := make(chan bool)
    csm, err := NewConsumer(configPath, contractmeta, mgh, done)
    if err != nil {
        return err
    }

    c.consumer = csm
    c.eventC = eventC
    c.meta = contractmeta
    c.pierId = pierId
    c.name = fabricConfig.Name
    c.outMeta = m
    c.ticker = time.NewTicker(2 * time.Second)
    c.done = done
    return nil
}
```

### consumer

consumer 负责监听区块链上的由跨链合约抛出的跨链事件以及和调用chaincode。

我们新建 `./consumer.go` 文件

```go
type Consumer struct {
   eventClient     *event.Client
   meta            *ContractMeta
   msgH            MessageHandler
   channelProvider context.ChannelProvider
   ChannelClient   *channel.Client
   registration    fab.Registration
   ctx             chan bool
}
```

- eventClient：fabric gosdk提供的事件Client

- meta Fabric：相关的参数信息

- msgH：事件handler，在监听到指定事件之后负责处理的函数

- channelProvider：fabric gosdk提供的和chaincode交互

- ChannelClient：fabric gosdk 提供的和调用chaincode的对象

- registeration：fabric gosdk 提供的订阅特定事件的对象

- ctx：用来结束consumer的goroutine

### Event

由于在Fabric上抛出的事件内容是可以自定义的，而跨链请求要在跨链平台上传递的话，需要使用IBTP包，所以我们需要一定的代码来执行这种转换。

我们新建 `./event.go` 文件

```go
type Event struct {
   Index         uint64 `json:"index"`
   DstChainID    string `json:"dst_chain_id"`
   SrcContractID string `json:"src_contract_id"`
   DstContractID string `json:"dst_contract_id"`
   Func          string `json:"func"`
   Args          string `json:"args"`
   Argscb        string `json:"argscb"`
   Rollback      string `json:"rollback"`
   Argsrb        string `json:"argsrb"`
   Callback      string `json:"callback"`
   Proof         []byte `json:"proof"`
   Extra         []byte `json:"extra"`
}
```

Event结构也是自定义的，需要和在你的跨链合约中抛出的事件结构一致。一个跨链交易事件，一般来说需要指定目标应用链的ID `DstChainID`，目标应用链上智能合约的地址或者ID（Fabric上的chaincode没有合约地址）`DstContractID`，这次跨链交易的发起者的合约地址`SrcContractID`，跨链调用的函数名 `Func`，该函数的参数 `Args`，是否有跨链调用之后要执行的回调函数 `Callback`，为了该应用链上对于该事件的证明 `Proof`，用户可自定义的部分 `Extra`。

### 读取配置

Plugin的配置文件路径是通过Initialize的方法动态传入的，这意味着你可以方便的修改关于你的区块链的参数信息。我们新建文件 `./config.go` 文件，负责配置读取的所有操作。

这里使用的是 `github.com/spf13/viper`库和TOML文件作为配置，当然你也可以使用任何你熟悉的工具来读取配置。

```go
package main

import (
   "path/filepath"
   "strings"

   "github.com/spf13/viper"
)

const (
   ConfigName = "fabric.toml"
)

type Fabric struct {
   Addr        string `toml:"addr" json:"addr"`
   Name        string `toml:"name" json:"name"`
   EventFilter string `mapstructure:"event_filter" toml:"event_filter" json:"event_filter"`
   Username    string `toml:"username" json:"username"`
   CCID        string `toml:"ccid" json:"ccid"`
   ChannelId   string `mapstructure:"channel_id" toml:"channel_id" json:"channel_id"`
   Org         string `toml:"org" json:"org"`
}

func DefaultConfig() *Fabric {
   return &Fabric{
      Addr:        "localhost:10053",
      Name:        "fabric",
      EventFilter: "CrosschainEventName",
      Username:    "Admin",
      CCID:        "Broker-001",
      ChannelId:   "mychannel",
      Org:         "org2",
   }
}

func UnmarshalConfig(configPath string) (*Fabric, error) {
   viper.SetConfigFile(filepath.Join(configPath, ConfigName))
   viper.SetConfigType("toml")
   viper.AutomaticEnv()
   viper.SetEnvPrefix("FABRIC")
   replacer := strings.NewReplacer(".", "_")
   viper.SetEnvKeyReplacer(replacer)
   if err := viper.ReadInConfig(); err != nil {
      return nil, err
   }

   config := DefaultConfig()

   if err := viper.Unmarshal(config); err != nil {
      return nil, err
   }

   return config, nil
}
```

### SubmitIBTP

该接口主要负责将其他链发送过来的IBTP包解析并构造成当前目的链的交易，发送到目的链的跨链合约中。
如果来源链要求将本链调用合约的结果返回的话，还需要构造相应的IBTP回执发回来源链。

```go
func (c *Client) SubmitIBTP(ibtp *pb.IBTP) (*pb.SubmitIBTPResponse, error) {
	pd := &pb.Payload{}
	ret := &pb.SubmitIBTPResponse{}
	if err := pd.Unmarshal(ibtp.Payload); err != nil {
		return ret, fmt.Errorf("ibtp payload unmarshal: %w", err)
	}
	content := &pb.Content{}
	if err := content.Unmarshal(pd.Content); err != nil {
		return ret, fmt.Errorf("ibtp content unmarshal: %w", err)
	}

	if ibtp.Category() == pb.IBTP_UNKNOWN {
		return nil, fmt.Errorf("invalid ibtp category")
	}

	logger.Info("submit ibtp", "id", ibtp.ID(), "contract", content.DstContractId, "func", content.Func)
	for i, arg := range content.Args {
		logger.Info("arg", strconv.Itoa(i), string(arg))
	}

	if ibtp.Category() == pb.IBTP_RESPONSE && content.Func == "" {
		logger.Info("InvokeIndexUpdate", "ibtp", ibtp.ID())
		_, resp, err := c.InvokeIndexUpdate(ibtp.From, ibtp.Index, ibtp.Category())
		if err != nil {
			return nil, err
		}
		ret.Status = resp.OK
		ret.Message = resp.Message

		return ret, nil
	}

	var result [][]byte
	var chResp *channel.Response
	callFunc := CallFunc{
		Func: content.Func,
		Args: content.Args,
	}
	bizData, err := json.Marshal(callFunc)
	if err != nil {
		ret.Status = false
		ret.Message = fmt.Sprintf("marshal ibtp %s func %s and args: %s", ibtp.ID(), callFunc.Func, err.Error())

		res, _, err := c.InvokeIndexUpdate(ibtp.From, ibtp.Index, ibtp.Category())
		if err != nil {
			return nil, err
		}
		chResp = res
	} else {
		res, resp, err := c.InvokeInterchain(ibtp.From, ibtp.Index, content.DstContractId, ibtp.Category(), bizData)
		if err != nil {
			return nil, fmt.Errorf("invoke interchain for ibtp %s to call %s: %w", ibtp.ID(), content.Func, err)
		}

		ret.Status = resp.OK
		ret.Message = resp.Message

		// if there is callback function, parse returned value
		result = util.ToChaincodeArgs(strings.Split(string(resp.Data), ",")...)
		chResp = res
	}

	// If is response IBTP, then simply return
	if ibtp.Category() == pb.IBTP_RESPONSE {
		return ret, nil
	}

	proof, err := c.getProof(*chResp)
	if err != nil {
		return ret, err
	}

	ret.Result, err = c.generateCallback(ibtp, result, proof, ret.Status)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
```



## 编译你的Plugin

我们采用GO语言提供的插件模式，实现Pier对于你编写的Plugin的动态加载。

MacOS和Linux平台：

运行下面的命令，能够得到 `your_plugin.so`文件。

```shell
$ cd ${YOUR_PROJECT_PATH}
$ go build --buildmode=plugin -o your_plugin.so ./*.go
```

将你编写的动态链接文件，放到Pier配置文件夹下，配合我们提供的Pier，就能接入到跨链平台来。