# 验证规则管理
中继链提供对应用链验证规则进行管理的功能。  

## 1 概述

验证规则是应用链在中继链上自主部署的智能合约，用于中继链验证应用链跨链请求的真伪。  

中继链的验证规则管理主要包含**验证规则注册**、**主验证规则切换**、**验证规则注销**。

## 2 验证规则注册
### 2.1 功能介绍
应用链可以向中继链注册多条验证规则，但只能有一个主验证规则，中继链的验证引擎根据主验证规则验证应用链的跨链请求。注册过程中验证规则的状态转换情况如下：
- 若当前没有主验证规则：`unavailable` --> `binding` --> `available`
- 若当前已有主验证规则：`unavailable` --> `bindable`  

如果当前应用链没有主验证规则，那么中继链会默认将新注册的验证规则绑定为主验证规则，该过程需要投票治理，故会经过一个绑定中的状态。如果应用链已经有一个主验证规则，那么中继链只会存储新规则，不会改变应用链的主验证规则。

### 2.2 使用方法


#### 第一步：应用链部署验证规则  
应用链可以通过pier向中继链部署验证规则，部署时默认向中继链发起注册,命令如下：
```shell script
// --repo：指定pier启动路径
// --path：指定验证规则所在路径
// --method：指定应用链的method信息
// --admin-key：指定应用链管理员key的路径
$ pier --repo ~/.pier_ethereum rule deploy --path ~/.pier_ethereum/ethereum/validating.wasm --method appchain --admin-key ~/.pier_ethereum/key.json
```
- 情景1：如果是首次部署验证规则（即当前应用链没有主验证规则），注册命令执行后打印信息如下：
```shell script
INFO[2021-07-23T09:48:44.608] Establish connection with bitxhub localhost:60013 successfully  module=rpcx
Deploy rule to bitxhub for appchain appchain successfully: 0x0c216b651E435d35d94196573BE7a89eDC883295
Register rule to bitxhub for appchain did:bitxhub:appchain:. successfully, the bind request was submitted successfully, wait for proposal 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-1 to finish.
```
接下来需要中继链管理员投票，进入第二步

- 情景2：如果当前应用链已经有一条主验证规则，再次注册命令执行后打印信息如下：
```shell script
INFO[2021-07-23T10:11:32.767] Establish connection with bitxhub localhost:60012 successfully  module=rpcx
Deploy rule to bitxhub for appchain appchain successfully: 0xa7d4E414FDB74fb2Bf8D851933d2bBbeF0B570ff
Register rule to bitxhub for appchain did:bitxhub:appchain:. successfully.
```
此时验证规则注册成功，进入第三步


#### 第二步：中继链管理员投票  
上一步情景1需要中继链管理员进一步进行投票治理，默认四个管理员的情况下需要三个管理员投赞成票提案可通过，命令如下
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定治理提案的id，上述验证规则注册的治理提案id从结果1的打印信息中可以看到是0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-1
// --info：指定投票内容是approve或是reject
// --reason：指定投票理由，可自定义
$ bitxhub --repo ~/work/bitxhub/scripts/build/node4 client governance vote --id 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-1 --info apprpve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node2 client governance vote --id 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-1 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance vote --id 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-1 --info approve --reason reason
```



#### 第三步：查看应用链验证规则  
中继链提供查看应用链验证规则的功能，命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定应用链id
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance rule all --id did:bitxhub:appchain:.
```
- 情景1执行结果如下，可以看到刚刚部署（注册）的验证规则已经成为主验证规则
```shell script
ChainId                 RuleAddress                                 Status     Master
-------                 -----------                                 ------     ------
did:bitxhub:appchain:.  0x0c216b651E435d35d94196573BE7a89eDC883295  available  true
```
- 情景2执行结果如下，可以看到刚刚部署（注册）的验证规则成为了一条可绑定状态的验证规则，而不是主验证规则。如果想将起设置为主验证规则，可以参考3.3节操作
```shell script
ChainId                 RuleAddress                                 Status     Master
-------                 -----------                                 ------     ------
did:bitxhub:appchain:.  0x0c216b651E435d35d94196573BE7a89eDC883295  available  true
did:bitxhub:appchain:.  0xa7d4E414FDB74fb2Bf8D851933d2bBbeF0B570ff  bindable   false
```

## 3 主验证规则切换
### 3.1 功能介绍
中继链上可能部署注册了应用链的多条验证规则（但只有一条主验证规则），应用链如果想更换主验证规则，可以使用主验证规则切换功能。  
切换主验证规则过程涉及到新旧两条主验证规则，新验证规则在切换之前必须是可绑定状态，切换过程中新旧主验证规则的状态转换如下：
- 新的主验证规则：`bindable` --> `binding` --> `available`
- 旧的主验证规则：`available` --> `unbinding` --> `bindable`

### 3.2 使用方法
#### 第一步：应用链切换主验证规则
应用链管理员可以通过pier向中继链发起切换主验证规则的请求，命令如下：
```shell script
// --repo 指定pier的启动路径
// --addr 指定新的主验证规则的地址
// --method：指定应用链的method信息
// --admin-key：指定应用链管理员key的路径
$ pier --repo ~/.pier_ethereum rule update --addr 0xa7d4E414FDB74fb2Bf8D851933d2bBbeF0B570ff --method appchain --admin-key ~/.pier_ethereum/key.json
```
该命令执行结果打印信息如下：
```shell script
INFO[2021-07-23T10:26:16.509] Establish connection with bitxhub localhost:60013 successfully  module=rpcx
Update master rule to bitxhub for appchain did:bitxhub:appchain:. successfully, wait for proposal 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-2 to finish.
```

#### 第二步：中继链管理员投票 
中继链管理员进一步进行投票治理，默认四个管理员的情况下需要三个管理员投赞成票提案可通过，命令如下
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定治理提案的id，上述主验证规则切换的治理提案id从打印信息中可以看到是0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-2
// --info：指定投票内容是approve或是reject
// --reason：指定投票理由，可自定义
$ bitxhub --repo ~/work/bitxhub/scripts/build/node4 client governance vote --id 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-2 --info apprpve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node2 client governance vote --id 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-2 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance vote --id 0x4DebB0f20B9e639827e273bc8FeDD73Da6193543-2 --info approve --reason reason
```



#### 第三步：查看应用链验证规则 
中继链提供查看应用链验证规则的功能，命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定应用链id
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance rule all --id did:bitxhub:appchain:.
```
执行结果如下，可以看到指定的验证规则已经成为新的主验证规则
```shell script
ChainId                 RuleAddress                                 Status     Master
-------                 -----------                                 ------     ------
did:bitxhub:appchain:.  0x0c216b651E435d35d94196573BE7a89eDC883295  bindable   false
did:bitxhub:appchain:.  0xa7d4E414FDB74fb2Bf8D851933d2bBbeF0B570ff  available  true
```

## 4 验证规则注销
### 4.1 功能介绍
应用链可以注销已经注册过的验证规则，但为保证应用链始终有一条可用的验证规则，主验证规则无法注销。应用链注销验证规则的过程不需要中继链管理员投票治理。  
注销过程中验证规则的状态转换如下：  
`bindable` --> `unbinding` --> `forbidden`

### 4.2 使用方法
#### 第一步：应用链注销验证规则
应用链管理员可以通过pier向中继链发起非主验证规则的注销，命令如下：
```shell script
$ pier --repo ~/.pier_ethereum rule logout --addr 0x0c216b651E435d35d94196573BE7a89eDC883295 --method appchain --admin-key ~/.goduck/pier/.pier_ethereum/key.json
```
注销成功后打印信息如下：  
```shell script
INFO[2021-07-23T10:37:53.380] Establish connection with bitxhub localhost:60014 successfully  module=rpcx
The logout request was submitted successfully
```

#### 第二步：查看应用链验证规则  
中继链提供查看应用链验证规则的功能，命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定应用链id
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance rule all --id did:bitxhub:appchain:.
```
执行结果如下，可以看到指定的验证规则已经成为新的主验证规则
```shell script
ChainId                 RuleAddress                                 Status     Master
-------                 -----------                                 ------     ------
did:bitxhub:appchain:.  0x0c216b651E435d35d94196573BE7a89eDC883295  forbidden  false
did:bitxhub:appchain:.  0xa7d4E414FDB74fb2Bf8D851933d2bBbeF0B570ff  available  true
```

## 5 其他功能
中继链还提供了其他查询验证规则信息的功能。
- 查询应用链所有验证规则，命令如下：
```shell script
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance rule all --id did:bitxhub:appchain:.
```
- 查询应用链所有可用验证规则，命令如下：
```shell script
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance rule available --id did:bitxhub:appchain:.
```
- 查询应用链某条验证规则状态，命令如下：
```shell script
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance rule status --id did:bitxhub:appchain:. --addr 0xa7d4E414FDB74fb2Bf8D851933d2bBbeF0B570ff
```