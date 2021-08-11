# 跨链合约调用

BitXHub跨链系统中，跨链合约有两个概念：

1. 应用链上部署的跨链管理合约broker
1. 中继链上用于处理跨链交易的内置合约

应用链上的跨链管理合约的调用可以参见[跨链合约](../dev/cross_contract.md)。本文主要讲解如何调用BitXHub内置的跨链合约。

## 合约接口
中继链的跨链合约提供了以下接口可以客户端调用：

```go
    // HandleIBTP check the received IBTP and post event for router to handle
    func HandleIBTP(ibtp *pb.IBTP) *boltvm.Response

    // Interchain returns information of the interchain count, Receipt count and SourceReceipt count
    func Interchain() *boltvm.Response
    
    // GetInterchain returns information of the interchain count, Receipt count and SourceReceipt count by id
    func GetInterchain(id string) *boltvm.Response
    
    // GetIBTPByID get the transaction hash which contains the IBTP by IBTP id
    func GetIBTPByID(id string) *boltvm.Response
```

## 接口说明

- `HandleIBTP`

当跨链网关捕获应用链抛出当跨链事件，并将其封装成IBTP以后，就需要向中继链发送交易调用该接口。

该接口主要作用是验证IBTP的来源链、目的链状态和IBTP index，更新来源链和目的链在中继链上的interchain数据，记录当前IBTP ID和当前交易HASH的对应关系，并抛出事件。

该接口接收的参数为IBTP。

- `Interchain`

该接口用于获取发起交易的应用链的interchain数据，其中包含该应用链向其他各条应用链发起跨链的数量和收到的回执的数量，以及其他应用链向该应用链发起跨链的数量。

该接口不需要参数。

- `GetInterchain`

该接口用于获取指定应用链的interchain数据，其中包含该应用链向其他各条应用链发起跨链的数量和收到的回执的数量，以及其他应用链向该应用链发起跨链的数量。

该接口接收的参数为应用链ID。

- `GetIBTPByID`

该接口用于获取包含指定IBTP的交易的HASH。（该IBTP ID和交易HASH的对应关系在HandleIBTP时记录）

该接口接收的参数为IBTP ID。

