# 应用链管理

中继链支持对应用链的管理，包括注册、更新、冻结、注销应用链等功能。

## 1. 应用链注册
### 1.1 命令
对于需要加入跨链网络使用中继链进行跨链的应用链，需要首先由应用链管理员向中继链注册应用链，命令如下：

```shell
pier [--repo <repository>] appchain register --name <appchain name> --type <appchain type> --desc <description> --version <appchain version> --validators <path of appchain validators file> --consensusType <appchain consensus type> [--addr <bitxhub node address>]
```
参数解释：

* --repo：可选参数，指定pier配置文件所在目录，如果不指定，默认使用$HOME/.pier目录。
* --name：必选参数，指定应用链名称。
* --type：必选参数，指定应用链类型，如hyperchain、fabric等。
* --desc：必选参数，对应用链的描述信息。
* --version：必选参数，指定应用链版本信息。
* --validators：必选参数，指定应用链的验证人信息所在的文件路径。
* --consensusType：必选参数，指定p应用链的共识类型，如rbft、raft等。
* --addr：可选参数，指定要连接的中继链节点地址，如果不指定，默认使用$repo目录下pier.toml中指定的BitXHub节点地址。

该命令向中继链发送一笔应用链注册的交易，中继链以交易的from（即当前pier公钥的地址）作为应用链的ID，生成一个应用链注册的提案。

中继链管理员需要对提案进行投票，命令如下：
```shell
bitxhub [--repo <repository>] client governance vote --id <proposal id> --info <voting information>  --reason <reason to vote>
```
参数解释：

* --repo：可选参数，指定bitxhub节点配置文件所在目录，如果不指定，默认使用$HOME/.bitxhub目录。
* --id：必选参数，指定提案id。
* --info：必选参数，指定投票信息，approve或者reject。
* --reason：必选参数，指定投票的原因。

### 1.2 举例
比如进行fabric应用链的注册，命令执行如下：
```shell
$ pier appchain register --name chainA --type fabric --desc chainA-desc --version 1.4.3 --validators config/fabric.validators --consensusType raft 
INFO[11:01:25.884] Establish connection with bitxhub localhost:60011 successfully  module=rpcx
the register request was submitted successfully, chain id is 0x8D093dd84717042b23546cdA396cEBB2F48D8442, proposal id is 0x8D093dd84717042b23546cdA396cEBB2F48D8442-0
```
如上例所示，应用链管理员提交应用链注册请求，应用链ID为0x8D093dd84717042b23546cdA396cEBB2F48D8442，提案号为0x8D093dd84717042b23546cdA396cEBB2F48D8442-0。中继链管理员需要对该提案进行审核并进行投票，命令如下：

如管理员对该提案审核后，认为该应用链提交对信息无误，投票通过，命令执行如下：
```shell
$ bitxhub --repo ./scripts/build/node1 client governance vote --id 0x8D093dd84717042b23546cdA396cEBB2F48D8442-0 --info approve --reason "fabric appchain register"
vote successfully!
$ bitxhub --repo ./scripts/build/node2 client governance vote --id 0x8D093dd84717042b23546cdA396cEBB2F48D8442-0 --info approve --reason "fabric appchain register"
vote successfully!
$ bitxhub --repo ./scripts/build/node3 client governance vote --id 0x8D093dd84717042b23546cdA396cEBB2F48D8442-0 --info approve --reason "fabric appchain register"
vote successfully!
$ bitxhub --repo ./scripts/build/node1 client governance proposals --id 0x8D093dd84717042b23546cdA396cEBB2F48D8442-0 
Id                                            Type         Status   ApproveNum  RejectNum  ElectorateNum  ThresholdNum  Des
--                                            ----         ------   ----------  ---------  -------------  ------------  ---
0x8D093dd84717042b23546cdA396cEBB2F48D8442-0  AppchainMgr  approve  3           0          4              3             register
```
可以看到该提案已经投票通过，应用链注册成功。

## 2. 更新应用链
### 2.1 命令
如果应用链发生了更改，比如验证人信息发生了变化，需要更新其在中继链上的应用链信息。命令如下：
```shell
pier --repo <repository> appchain update --name <appchain name> --type <appchain type> --desc <description> --version <appchain version> --validators <path of appchain validators file> --consensusType <appchain consensus type> --addr <bitxhub node address>
```

该命令参数含义与应用链注册命令的参数一致，不再赘述。

### 2.2 举例
比如进行fabric应用链的验证人信息发生变化，需要更新应用链，命令执行如下：
```shell
$ pier appchain update --name chainA --type fabric --desc chainA-desc --version 1.4.3 --validators config/fabric-new.validators --consensusType raft 
INFO[11:01:25.884] Establish connection with bitxhub localhost:60011 successfully  module=rpcx
the update request was submitted successfully, proposal id is 0x8D093dd84717042b23546cdA396cEBB2F48D8442-1
```

管理员进行审核并投票，与注册应用链部分一致，不再赘述。

## 3. 冻结应用链
### 3.1 命令
如果应用链出现问题，应用链管理员可以申请冻结应用链。命令如下：
```shell
pier --repo <repository> appchain freeze
```

### 3.2 举例
比如对之前已经注册过的应用链进行冻结，命令执行如下：
```shell
$ pier appchain freeze
INFO[11:01:25.884] Establish connection with bitxhub localhost:60011 successfully  module=rpcx
the freeze request was submitted successfully, proposal id is 0x8D093dd84717042b23546cdA396cEBB2F48D8442-2
```

管理员进行审核并投票，与注册应用链部分一致，不再赘述。


## 4. 激活应用链
### 4.1 命令
如果冻结的应用链恢复正常，应用链管理员可以申请激活应用链。命令如下：
```shell
pier --repo <repository> appchain activate
```

### 4.2 举例
比如对之前已经冻结的应用链进行激活，命令执行如下：
```shell
$ pier appchain freeze
INFO[11:01:25.884] Establish connection with bitxhub localhost:60011 successfully  module=rpcx
the activate request was submitted successfully, proposal id is 0x8D093dd84717042b23546cdA396cEBB2F48D8442-3
```

管理员进行审核并投票，与注册应用链部分一致，不再赘述。

## 5. 注销应用链
### 5.1 命令
如果应用链退出跨链系统，不再进行跨链，应用链管理员可以向中继链提交注销应用链的提案。命令如下：
```shell
pier --repo <repository> appchain logout
```

### 5.2 举例
比如对之前激活的应用链进行注销，命令执行如下：
```shell
$ pier appchain logout 
INFO[11:01:25.884] Establish connection with bitxhub localhost:60011 successfully  module=rpcx
the logout request was submitted successfully, proposal id is 0x8D093dd84717042b23546cdA396cEBB2F48D8442-4
```

管理员进行审核并投票，与注册应用链部分一致，不再赘述。