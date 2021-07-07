# 快速开始
我们提供了Goduck运维小工具来快速体验跨链流程。

## 1 环境准备

> Goduck快速开始依赖于Docker和Docker-Compose，需要提前准备好docker环境。

## 2 下载 Goduck

> **！！！注意：** 执行上述命令的路径下应该没有goduck同名目录

下载Goduck可执行二进制文件：

```shell
curl https://raw.githubusercontent.com/meshplus/goduck/release-1.0/scripts/goduck.sh -L -o - | bash
```

## 3 初始化
初始化goduck配置文件，命令如下：
```shell
goduck init
```

## 4 启动跨链网络
在本地启动一个solo版本的BitXHub节点、两条以太坊私有链以及相应的两个跨链网关，启动命令如下：

```shell
goduck playground start
```
该命令执行的具体操作包括以下步骤：

### 获取相关镜像
镜像拉取成功后将会打印日志如下：

```shell
Creating network "quick_start_default" with the default driver
Creating ethereum-1      ... done
Creating bitxhub_solo ... done
Creating ethereum-2   ... done
Creating pier-ethereum-2 ... done
Creating pier-ethereum-1 ... done
Attaching to bitxhub_solo, ethereum-2, ethereum-1, pier-ethereum-1, pier-ethereum-2
```

### 启动两条以太坊私有链
以太坊私有链ethereum-1和ethereum-2启动成功后，最终会打印出日志如下：
```shell
ethereum-2         | INFO [04-02|02:41:57.348] Sealing paused, waiting for transactions
ethereum-1         | INFO [04-02|02:41:57.349] Sealing paused, waiting for transactions
```

### 启动中继链
启动一条solo模式的中继链，启动成功后将会打印出BitXHub的logo如下：

```shell
bitxhub_solo       | =======================================================
bitxhub_solo       |     ____     _    __    _  __    __  __            __
bitxhub_solo       |    / __ )   (_)  / /_  | |/ /   / / / /  __  __   / /_
bitxhub_solo       |   / __  |  / /  / __/  |   /   / /_/ /  / / / /  / __ \
bitxhub_solo       |  / /_/ /  / /  / /_   /   |   / __  /  / /_/ /  / /_/ /
bitxhub_solo       | /_____/  /_/   \__/  /_/|_|  /_/ /_/   \__,_/  /_.___/
bitxhub_solo       |
bitxhub_solo       | =======================================================
```

### 注册应用链
创建两个与应用链对应的跨链网关，通过跨链网关向中继链注册应用链信息，注册成功后会打印出应用链id如下：

```shell
pier-ethereum-1    | appchain register successfully, id is 0xb132702a7500507411f3bd61ab33d9d350d41a37
pier-ethereum-2    | appchain register successfully, id is 0x9f5cf4b97965ababe19fcf3f1f12bb794a7dc279
```

### 部署验证规则
跨链网关向中继链部署应用链的验证规则，部署成功后将打印日志如下：

```shell
pier-ethereum-1    | Deploy rule to bitxhub successfully
pier-ethereum-2    | Deploy rule to bitxhub successfully
```

### 启动跨链网关
跨链网关完成注册应用链、部署验证规则两步后即可启动，启动成功后跨链网关会打印出日志如下：

```shell
pier-ethereum-1    | time="02:42:02.287" level=info msg="Exchanger started" module=exchanger
pier-ethereum-2    | time="02:42:02.349" level=info msg="Exchanger started" module=exchanger
```
中继链会打印出日志如下：

```
bitxhub_solo       | time="02:42:02.291" level=info msg="Add pier" id=0xb132702a7500507411f3bd61ab33d9d350d41a37 module=router
bitxhub_solo       | time="02:42:02.353" level=info msg="Add pier" id=0x9f5cf4b97965ababe19fcf3f1f12bb794a7dc279 module=router
```


**！！！注意：** 如遇网络问题，运行下面命令：

```shell
goduck playground clean
```


## 5 跨链交易

分别在两条以太坊应用链上发起跨链交易，执行命令如下：

```shell
goduck playground transfer
```
该命令会调用以太坊上的合约（以太坊镜像上已经部署好合约）发起两笔跨链交易：
- 从ethereum-1的Alice账户转账1到ethereum-2的Alice账户
- 从ethereum-2的Alice账户转账1到ethereum-1的Alice账户


信息打印出如下：
```shell
// 查询账户余额
1. Query original accounts in appchains
// 以太坊1的Alice账户余额为10000
Query Alice account in ethereum-1 appchain

======= invoke function getBalance =======
call result: 10000
// 以太坊2的Alice账户余额为10000
Query Alice account in ethereum-2 appchain

======= invoke function getBalance =======
call result: 10000

// 从以太坊1的Alice账户转账1到以太坊2的Alice账户
2. Send 1 coin from Alice in ethereum-1 to Alice in ethereum-2
======= invoke function transfer =======

=============== Transaction hash is ==============
0xff42eb87410f7ed4c8bf394716b7f202d4f307191e5ac2ef3cfb77fabd8211a0

// 查询账户余额
3. Query accounts after the first-round invocation
// 以太坊1的Alice账户余额为9999
Query Alice account in ethereum-1 appchain

======= invoke function getBalance =======
call result: 9999
// 以太坊2的Alice账户余额为10001
Query Alice account in ethereum-2 appchain

======= invoke function getBalance =======
call result: 10001

// 从以太坊2的Alice账户转账1到以太坊1的Alice账户
4. Send 1 coin from Alice in ethereum-2 to Alice in ethereum-1

======= invoke function transfer =======

=============== Transaction hash is ==============
0x0c8042a3539cd49ce5d0570afbdb625697aadf71a040203f6bc03e3a43fb71b5

// 查询账户余额
5. Query accounts after the second-round invocation
// 以太坊1的Alice账户余额为10000
Query Alice account in ethereum-1 appchain

======= invoke function getBalance =======
call result: 10000
// 以太坊2的Alice账户余额为10000
Query Alice account in ethereum-2 appchain

======= invoke function getBalance =======
call result: 10000
```