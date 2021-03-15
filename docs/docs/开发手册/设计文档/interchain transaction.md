# 跨链事务方案
BitXHub提供两种不同的事务方案，用户可以根据实际需求选择。

## 1. 本地消息表事务方案

若用户不想更改业务合约或者所用的区块链系统不支持跨合约调用，可以选择本地消息表方案。该方案在跨链交易失败情况下，需要由业务系统负责实现事务回滚操作，业务系统需实现接收跨链消息的函数对应的回滚函数。

当来源链发起跨链交易时，该跨链交易需要注册到跨链事务合约，以KV形式记录全局事务ID和事务信息，全局事务ID由来源链ID || 跨链交易index组成，事务信息表示如下：

```go
type txInfo struct {
	globalState string
	// 子事务信息，key为子事务ID，即各目的链的地址，value为子事务状态
	childTxInfo map<string, string>
}
```

其中，全局事务状态和子事务初始状态为BEGIN。之后，跨链事务合约将跨链消息发送至各目的链的Pier。Pier在提交跨链交易之前，需要发送交易到事务代理合约，由事务代理合约通知业务系统将全局事务ID以及该跨链交易相关该合约通知业务系统跨链事件发生，业务系统应该记录全局事务ID以及该跨链交易相关信息，以便在接收到回滚消息时做相应的回滚操作。

当目的链处理完跨链交易，需要向跨链事务管理合约，反馈子事务情况：

- 如果交易执行成功，则将对应的子事务状态设置为SUCCESS，当所有子事务状态均为SUCCESS时，跨链事务管理合约将该全局事务ID对应的事务状态更新为SUCCESS

- 如果交易执行失败，则将对应的子事务状态设置为FAILURE，并将该全局事务ID对应的事务状态更新为FAILURE，同时通知来源链和所有相关的目的链，各应用链通过事务代理合约通知业务系统回滚该事务对应的交易，通知信息中包含全局事务ID

跨链事务管理合约接口设计如下：

```go
// 由来源链的Pier调用，开始跨链事务，该接口将设置事务状态为BEGIN，并向各目的链发送子事务
Begin(globalId string， targets []byte)
// 由目的链执行完子事务后由目的链Pier调用，result为执行结果
Prepared(globalId string, result bool)
```

跨链代理合约接口设计如下：

```go
// Pier调用该方法，通过事件通知业务系统跨链事务开始
NotifyBizStart(globalId string， txHash string)
// Pier调用该方法，通过事件通知业务系统跨链事务需要回滚
NotifyBizRollback(globalId string)
```

中继链模式下，跨链事务合约部署在中继链；直连模式下，跨链事务合约部署在应用链。事务代理合约可以和broker合约结合。

## 2. 二阶段事务方案

若用户想要完善的事务保证，并且可以修改业务合约，则可以选择二阶段事务方案。

### 1. 资源锁定

二阶段事务方案需要锁定业务合约中的资源，该方式要求更改业务合约，在具体的业务上下文增加资源锁定和解锁的逻辑，以及预执行相关的代码逻辑

### 2. 中继链模式

分为prepare和commit两个阶段，由中继链担任协调者，使用内置的跨链事务管理合约协调多个应用链的跨链交易。

内置的跨链事务管理合约接口设计如下：

```go
// 由来源链的Pier调用，开始跨链事务，该接口将向各目的链发起prepare事件，各目的链进行交易执行
Begin(globalId string, targets []byte, height uint64)
// 由目的链的Pier调用，status为预执行结果， undoLog为回滚需要调用的合约方法和参数
Prepared(globalId string, status bool, undoLog []byte)
// 由来源链和目的链的Pier调用
Committed(globalId string)
// 由来源链和目的链的Pier调用
Rollbacked(globalId string)
// 如果中继链收到目的链prepare失败的消息或者目的链超时，中继链触发rollback；
// 各参与方也可以在一定情况下主动发起rollback
Rollback(globalId string)
```

跨链事务的状态变换如下：

- 成功：BEGIN -> PREPARED -> COMMITTED

- 失败：BEGIN -> ABORTED -> ROLLBACKED

流程：

![](http://teambitiondoc.hyperchain.cn:8099/storage/011u3ff3f9226a3870b7f7f6ef9ca4c63093?Signature=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBcHBJRCI6IjU5Mzc3MGZmODM5NjMyMDAyZTAzNThmMSIsIl9hcHBJZCI6IjU5Mzc3MGZmODM5NjMyMDAyZTAzNThmMSIsIl9vcmdhbml6YXRpb25JZCI6IiIsImV4cCI6MTYxNTk2MzgyMCwiaWF0IjoxNjE1MzU5MDIwLCJyZXNvdXJjZSI6Ii9zdG9yYWdlLzAxMXUzZmYzZjkyMjZhMzg3MGI3ZjdmNmVmOWNhNGM2MzA5MyJ9.EUNnGBvL5G7D8BkJLgmMpfR4peINZAWll2-tLpexsFg&download=%E4%BA%8C%E9%98%B6%E6%AE%B5%E4%BA%8B%E5%8A%A1.png "")

1. 来源链预执行交易，如果成功，则锁定资源并发起跨链转账，将预执行结果写入事件，预执行失败则结束

1. 来源链Pier将事件提交给中继链，调用中继链的跨链事务管理合约Begin方法，跨链事务状态为BEGIN

1. 中继链将跨链交易分发到各个目的链

1. 各目的链Pier接收事件，并提交交易给目的链

1. 各目的链执行跨链交易，如果成功，则锁定资源，如果失败，则回滚交易，将执行结果返回给Pier

1. Pier将执行结果提交给中继链，调用合约的Prepared方法

1. 如果所有目的链都prepare成功，事务状态更新为PREPARED，并触发commit事件，继续下一步；如果中继链收到prepare失败的交易，事务状态更新为ABORTED，并对除prepare失败的其他参与链触发rollback事件，继续步骤12

1. 各参与链Pier接收事件，并提交交易给目的链

1. 各参与链执行commit操作，并释放资源

1. 各参与链Pier在目的链commit完成后调用中继链合约的Committed方法

1. 如果参与链的commit都提交了，中继链将该事务状态更新为COMMITTED，继续步骤16

1. 各参与链Pier接收事件，并提交交易给目的链

1. 各参与执行rollback操作，回滚交易的结果，解锁资源

1. 各参与链Pier在目的链rollback完成后调用中继链合约的Rollbacked方法

1. 如果参与链都rollback了，中继链将该事务状态更新为ROLLBACKED

1. 跨链事务结束



### 3. 直连模式

直连模式中协调者由来源链担任，需要在应用链上部署跨链事务管理合约。流程和中继链模式类似



## 3. 改进的二阶段事务方案

该方案用于只有一个目的链的场景。跨链交易在目的链不需要锁定资源，直接执行，若成功，则将结果写入账本，若失败，则直接回滚。

