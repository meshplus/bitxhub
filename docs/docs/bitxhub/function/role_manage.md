# 统一身份管理
中继链提供了关于中继链上身份的统一管理功能

## 1 概述  
中继链平台上可能出现的身份有三种：中继链治理管理员、中继链审计管理员、应用链管理员（下文中不作特殊说明的中继链管理员都是指治理管理员）
- 中继链治理管理员：参与中继链上的投票治理
- 中继链审计管理员：不参与中继链上的投票治理，与nvpNode绑定并对节点上同步的数据进行审计
- 应用链管理员：应用链上的管理员，中继链并不直接对应用链管理员进行管理，仅做身份的记录

其中中继链治理管理员又可以分为超级治理管理员和普通治理管理员，中继链初始状态至少有一个超级管理员，超级管理员后续不可以注册、冻结或注销，一些高优先级的提案（比如与管理员相关的提案、与冻结或注销之类会造成严峻后果的提案）需要超级管理员投票才能生效  

中继链的统一身份管理功能主要提供管理员身份的注册、更新、冻结、激活及注销。  

## 2 身份注册
### 2.1 功能介绍
中继链管理员可以向中继链上注册治理管理员或审计管理员，需要注意的是注册审计管理员时需要绑定一个已经存在的审计节点（即需要现在注册审计节点再注册审计管理员）。注册过程中管理员的状态转换如下：  
`unavailable` --> `registering` --> `available`  

### 2.2 使用方法
#### 第一步：中继链管理员注册
- 治理管理员
中继链管理员注册新的治理管理员的命令如下：
```shell script
// --repo：指定中继链管理员私钥的路径
// --address：指定新管理员的地址
// --type：指定新管理员的类型，默认为治理管理员
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role register --address 0x936A953274bcd0d42bf0b95308040Bb469b13BA6 --type governanceAdmin
proposal id is 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0
```
根据上述命令执行的打印信息可以看到注册治理管理员的提案号为0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0

- 审计管理员
中继链管理员注册新的审计管理员的命令如下：
```shell script
// --repo：指定中继链管理员私钥的路径
// --address：指定新管理员的地址
// --type：指定新管理员的类型
// --nodePid：指定审计管理员绑定的nvpNode节点id
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role register --address 0x936A953274bcd0d42bf0b95308040Bb469b13BA7 --type auditAdmin --nodePid QmPSzhXo2MQxWReUXXPRhytYS7HMh1bjRLupjJ1LEGaLzg
proposal id is 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1
```
根据上述命令执行的打印信息可以看到注册审计管理员的提案号为0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1

#### 第二步：中继链管理员投票
- 治理管理员
中继链管理员进一步进行投票治理，默认四个管理员的情况下需要三个管理员投赞成票提案可通过，命令如下
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定治理提案的id，0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0
// --info：指定投票内容是approve或是reject
// --reason：指定投票理由，可自定义
$ bitxhub --repo ~/work/bitxhub/scripts/build/node4 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0 --info apprpve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node2 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-0 --info approve --reason reason
```

- 审计管理员
中继链管理员进一步进行投票治理，由于前面新注册成功了一个治理管理员，治理管理员总数为5，此时需要至少4个管理员投票（即刚刚注册的管理员也可以参与治理），命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定治理提案的id，0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1
// --info：指定投票内容是approve或是reject
// --reason：指定投票理由，可自定义
$ bitxhub --repo ~/work/bitxhub/scripts/build/node4 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1 --info apprpve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node2 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/role5 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-1 --info approve --reason reason
```

#### 第三步：查看中继链管理员身份信息 
中继链提供查看身份信息的功能，命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --type：指定中继链管理员类型，不填的话默认为治理管理员
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role all --type governanceAdmin
```
治理管理员类型执行结果如下，可以看到新注册的治理管理员已经是可用状态，且其为普通治理管理员，而初始的几个管理员均为超级治理管理员：  
```shell script
RoleId                                      type             Weight  NodePid  Status
------                                      ----             ------  -------  ------
0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8  governanceAdmin  2                available
0x936A953274bcd0d42bf0b95308040Bb469b13BA6  governanceAdmin  1                available
0x97c8B516D19edBf575D72a172Af7F418BE498C37  governanceAdmin  2                available
0x79a1215469FaB6f9c63c1816b45183AD3624bE34  governanceAdmin  2                available
0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013  governanceAdmin  2                available
```
审计管理员类型执行结果如下，可以看到新注册的审计管理员已经是可用状态：
```shell script
RoleId                                      type        Weight  NodePid                                         Status
------                                      ----        ------  -------                                         ------
0x936A953274bcd0d42bf0b95308040Bb469b13BA7  auditAdmin  1       QmPSzhXo2MQxWReUXXPRhytYS7HMh1bjRLupjJ1LEGaLzg  available
```


## 3 身份更新
### 3.1 功能介绍
中继链管理员身份的更新指审计管理员更新其绑定的审计节点pid，新的审计节点需要提前注册好，且新的审计节点pid不可以与原审计节点pid相同。更新过程中审计管理员的状态转换如下：  
`available` --> `updating` --> `available`  

### 3.2 使用方法
#### 第一步：中继链管理员更新审计管理员
中继链管理员更新审计管理员绑定节点的命令如下：
```shell script
// --repo：指定中继链管理员私钥的路径
// --id：指定待更新审计管理员的地址
// --type：指定新管理员的类型，默认为治理管理员
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role update --id 0x936A953274bcd0d42bf0b95308040Bb469b13BA7 --nodePid QmQaBr8oak4F66AcTqZJo2oYZfQN7cJ6o9aV4UeKzmHTpz
proposal id is 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-4
```
根据上述命令执行的打印信息可以看到更新审计管理员的提案号为0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-4

#### 第二步：中继链管理员投票
中继链管理员进一步进行投票治理，五个管理员的情况下需要四个管理员投赞成票提案可通过，命令如下
```shell script
// --repo：指定中继链管理员key的路径
// --id：指定治理提案的id，0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-4
// --info：指定投票内容是approve或是reject
// --reason：指定投票理由，可自定义
$ bitxhub --repo ~/work/bitxhub/scripts/build/node4 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-4 --info apprpve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node2 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-4 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-4 --info approve --reason reason
$ bitxhub --repo ~/work/bitxhub/scripts/build/role5 client governance vote --id 0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013-4 --info approve --reason reason
```

#### 第三步：查看审计管理员身份信息 
中继链提供查看身份信息的功能，命令如下：
```shell script
// --repo：指定中继链管理员key的路径
// --type：指定中继链管理员类型，不填的话默认为治理管理员
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role all --type governanceAdmin
```
执行结果如下，可以看到审计管理员的nodePid已经更新了：  
```shell script
RoleId                                      type        Weight  NodePid                                         Status
------                                      ----        ------  -------                                         ------
0x936A953274bcd0d42bf0b95308040Bb469b13BA7  auditAdmin  1       QmQaBr8oak4F66AcTqZJo2oYZfQN7cJ6o9aV4UeKzmHTpz  available
```

## 4 身份冻结
### 4.1 功能介绍
管理员身份被冻结后将无法正常工作，比如治理管理员被冻结后将无法正常发提案或投票，需要注意的是超级治理管理员不可被冻结，治理管理员不可以自己发提案冻结自己。冻结过程的状态转换如下：    
`available` --> `freezinging` --> `frozen`  

### 4.2 使用方法
中继链管理员冻结另一个管理员的命令如下：
```shell script
// --repo：指定中继链管理员私钥的路径
// --id：指定待冻结管理员的地址
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role freeze --id 0x936A953274bcd0d42bf0b95308040Bb469b13BA6
```
冻结过程需要中继链管理员进行投票治理，治理过程与上文类似，此处省略。

## 5 身份激活
### 5.1 功能介绍
管理员身份被冻结后可以通过激活恢复正常工作，激活过程的状态转换如下：    
`frozen` --> `activating` --> `available`

### 5.2 使用方法
中继链管理员激活另一个被冻结的管理员的命令如下：
```shell script
// --repo：指定中继链管理员私钥的路径
// --id：指定待激活管理员的地址
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role activate --id 0x936A953274bcd0d42bf0b95308040Bb469b13BA6
```
激活过程需要中继链管理员进行投票治理，治理过程与上文类似，此处省略。

## 6 身份注销
### 6.1 功能介绍
普通治理管理员和审计管理员都可以被注销，身份注销是不可逆操作，注销过程的状态转换如下：    
`*` --> `logouting` --> `forbidden`
注销是一种高优先级的提案，即管理员身份注册成功后处于任意状态时都可以被注销，故状态转换的起始状态是不确定的

### 6.2 使用方法
中继链管理员注销管理员的命令如下：
```shell script
// --repo：指定中继链管理员私钥的路径
// --id：指定待注销管理员的地址
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role logout --id 0x936A953274bcd0d42bf0b95308040Bb469b13BA6
```
注销过程需要中继链管理员进行投票治理，治理过程与上文类似，此处省略。

## 7 其他功能
中继链还提供了其他查询身份信息的功能。
- 查询中继链所有指定类型管理员，默认查询治理管理员，命令如下：
```shell script
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role all
```
- 查询中继链某个管理员的状态，命令如下：
```shell script
$ bitxhub --repo ~/work/bitxhub/scripts/build/node1 client governance role status --id 0x936A953274bcd0d42bf0b95308040Bb469b13BA6
```
