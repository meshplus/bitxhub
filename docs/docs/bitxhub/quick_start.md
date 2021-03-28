# 快速开始
我们提供了脚本来快速启动中继链、两条Fabric应用链A和B和跨链网关。


## 启动BitXHub
BitXHub依赖于[golang](https://golang.org/)和[tmux](https://github.com/tmux/tmux/wiki)，需要提前进行安装。

使用下面的命令克隆项目：

```shell
git clone git@github.com:meshplus/bitxhub.git
```

BitXHub还依赖于一些小工具，使用下面的命令进行安装：

```shell
cd bitxhub
git checkout v1.0.0-rc1
bash scripts/prepare.sh 
```

最后，运行下面的命令即可运行一个四节点的BitXHub中继链：

```shell
make cluster
```

**注意：** `make cluster`启动会使用`tmux`进行分屏，所以在命令执行过程中，最好不要进行终端切换。


## 部署Fabric网络

在运行跨链网络之前，必要的软件如Golang和Docker可以根据官网自行安装。确保 $GAPTH， $GOBIN等环境变量已经正确设置。

以上的软件依赖安装之后，我们提供了脚本来安装启动两个简单的Fabric网络。

下载部署fabric网络脚本（ffn.sh）：

```shell
wget https://github.com/meshplus/goduck/raw/master/scripts/quick_start/ffn.sh
```

启动fabric网络：

**注意：** 脚本运行过程中按照提示进行确认即可。

```shell
bash ffn.sh down // 如果本地已经有Fabric网络运行，需要先关闭，如果没有可以不运行该命令
bash ffn.sh up //启动两条fabric应用链A和B
```

**注意：** 命令执行完，会在当前目录生成`crypto-config`和`crypto-configB`文件夹，后面的`chaincode.sh`和 `fabric_pier.sh`需要在执行目录下存在上述文件。


## 部署跨链合约
下载操作`chaincode`的脚本（chaincode.sh）：

```shell
wget https://raw.githubusercontent.com/meshplus/goduck/master/scripts/quick_start/chaincode.sh
```

拷贝`crypto-config`和`crypto-configB`到当前目录，执行以下命令：

```shell
// -c：指定fabric cli的配置文件，默认为config.yaml
// 应用链A部署chaincode
bash chaincode.sh install 

//应用链B部署chaincode
bash chaincode.sh install -c 'configB.yaml' 
```

该命令会在指定fabric网络中部署`broker`, `transfer` 和`data_swapper`三个`chaincode`

部署完成后，通过以下命令检查是否部署成功：

```shell
// 上一步的脚本默认初始化了一个有10000余额的账号Alice（transfer chaincode）
// -c：指定fabric cli的config.yaml配置文件
// 查看应用链A中Alice的余额
bash chaincode.sh get_balance -c 'config.yaml' 
****************************************************************************************************
***** |  Response[0]: //peer0.org2.example.com
***** |  |  Payload: 10000
***** |  Response[1]: //peer1.org2.example.com
***** |  |  Payload: 10000
****************************************************************************************************

//查看应用链B中Alice的余额
bash chaincode.sh get_balance -c 'configB.yaml' 
****************************************************************************************************
***** |  Response[0]: //peer0.org2.example1.com
***** |  |  Payload: 10000
***** |  Response[1]: //peer1.org2.example1.com
***** |  |  Payload: 10000
****************************************************************************************************
```



## 启动跨链网关

下载相关的脚本（fabric_pier.sh）：

```shell
wget https://github.com/meshplus/goduck/raw/master/scripts/quick_start/fabric_pier.sh
```

执行以下命令即可启动跨链网关：

```shell
//启动跨链网关连接应用链A和BitXHub
// -r: 跨链网关启动目录，默认为.pier目录
// -c: fabric组织证书目录，默认为crypto-config
// -g: 指定fabric cli连接的配置文件，默认为config.yaml
// -p: 跨链网关的启动端口，默认为8987
// -b: 中继链GRPC地址，默认为localhost:60011
// -o: pprof端口，默认为44555
bash fabric_pier.sh start -r '.pier' -c 'crypto-config'  -g 'config.yaml' -p 8987 -b 'localhost:60011' -o 44555

//启动跨链网关连接应用链B和BitXHub
bash fabric_pier.sh start -r '.pierB' -c 'crypto-configB'  -g 'configB.yaml' -p 8988 -b 'localhost:60011' -o 44556
```

在该目录下通过以下命令可以得到该跨链网关对应应用链的ID：

```shell
//应用链A的ID
bash fabric_pier.sh id -r '.pier'

//应用链B的ID
bash fabric_pier.sh id -r '.pierB'
```

**注意：** 后面跨链交易命令需要该值

## 跨链转账
使用**部署跨链合约**章节中下载的`chaincode.sh`进行相关`chaincode`调用。

1. 查询Alice余额

```shell
// 查询应用链A中Alice的余额
// -c：指定fabric cli的config.yaml配置文件
bash chaincode.sh get_balance -c 'config.yaml'
****************************************************************************************************
***** |  Response[0]: //peer0.org2.example.com
***** |  |  Payload: 10000
***** |  Response[1]: //peer1.org2.example.com
***** |  |  Payload: 10000
****************************************************************************************************

// 查询应用链B中Alice的余额
bash chaincode.sh get_balance -c 'configB.yaml'
****************************************************************************************************
***** |  Response[0]: //peer0.org2.example1.com
***** |  |  Payload: 10000
***** |  Response[1]: //peer1.org2.example1.com
***** |  |  Payload: 10000
****************************************************************************************************
```

2. 发送一笔跨链转账

下面的命令会将应用链A中Alice的一块钱转移到应用链B中Alice：

```shell
// -c：指定fabric cli的config.yaml配置文件
// -t: 目的链的ID（应用链B的ID）
bash chaincode.sh interchain_transfer -c 'config.yaml' -t <target_appchain_id>
```
3. 查询余额

分别在两条链上查询Alice余额：

```shell
// 查询应用链A中Alice的余额，发现余额少了一块钱
// -c：指定fabric cli的config.yaml配置文件
bash chaincode.sh get_balance -c 'config.yaml'
****************************************************************************************************
***** |  Response[0]: //peer0.org2.example.com
***** |  |  Payload: 9999
***** |  Response[1]: //peer1.org2.example.com
***** |  |  Payload: 9999
****************************************************************************************************

// 查询应用链B中Alice的余额，发现余额多了一块钱
bash chaincode.sh get_balance -c 'configB.yaml'
****************************************************************************************************
***** |  Response[0]: //peer0.org2.example1.com
***** |  |  Payload: 10001
***** |  Response[1]: //peer1.org2.example1.com
***** |  |  Payload: 10001
****************************************************************************************************
```

**注意：** `chaincode.sh`调用不同的fabric，需要不同的`config.yaml`，注意区分。