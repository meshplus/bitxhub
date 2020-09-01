# BitXHub使用文档

## 准备工作

BitXHub使用[**golang**](https://golang.org/doc/install)开发，版本要求1.13以上。

如果你想在本地或者服务器上运行BitXHub，你需要在机器上安装有以下的依赖：

[__packr__](https://github.com/gobuffalo/packr)、[__gomock__](https://gist.github.com/thiagozs/4276432d12c2e5b152ea15b3f8b0012e#installation)、[__mockgen__](https://gist.github.com/thiagozs/4276432d12c2e5b152ea15b3f8b0012e#installation)

可以通过下面的命令，快速安装依赖：

```bash
bash scripts/prepare.sh
```

## 编译

编译bitxhub：

```bash
make build
```

编译完后，会在`bin`目录下生成一个`bitxhub`二进制文件。

使用`version`命令进行验证：

```bash
$ ./bin/bitxhub version
BitXHub version: 1.0.0-master-2bb82e8
App build date: 2020-03-31T00:02:19
System version: darwin/amd64
Golang version: go1.13.8
```

如果想直接安装`bitxhub`到可执行环境，可以使用`install`命令：

```bash
make install
```

## 启动solo模式bitxhub

使用下面的命令即可启动一个单节点的bitxhub：

```bash
cd scripts
bash solo.sh
```

启动成功会打印出BitXHub的ASCII字体：
<p>
    <img src="https://user-images.githubusercontent.com/29200902/77936225-30cc6400-72e5-11ea-9499-1ead165c5495.png" width="50%" />
</p>

## 本地快速启动4节点

本地快速启动脚本依赖于[tmux](https://github.com/tmux/tmux/wiki)，需要提前进行安装。

使用下面的命令克隆项目：

```shell
git clone git@github.com:meshplus/bitxhub.git
```

BitXHub还依赖于一些小工具，使用下面的命令进行安装：

```shell
cd bitxhub
bash scripts/prepare.sh 
```

最后，运行下面的命令即可运行一个四节点的BitXHub中继链：

```shell
make cluster
```

启动成功会在四个窗格中分别打印出BitXHub的ASCII字体。

**注意：** `make cluster`启动会使用`tmux`进行分屏，所以在命令执行过程中，最好不要进行终端切换。

## 自定义启动

首先，使用项目提供的配置脚本快速生成配置文件：

```shell
cd bitxhub/scripts
bash config.sh <number> // number是节点数量，
```

上面命令以生成4个节点配置作为例子，会在当前目录下的build文件夹下生成如下文件：

```shell
.
├── addresses
├── agency.cert
├── agency.priv
├── bitxhub
├── ca.cert
├── ca.priv
├── node1
├── node2
├── node3
├── node4
├── pids
└── raft.so
```

node1-node4下的文件信息如下：

```shell
.
├── api
├── bitxhub.toml
├── certs
├── network.toml
├── order.toml
├── plugins
└── start.sh
```

### 修改端口信息

端口的配置主要在`bitxhub.toml`文件中。

`port.grpc` 修改节点的grpc端口

`port.gateway` 修改节点的grpc gateway端口

`port.pprof` 修改节点的pprof端口

### 修改初始化账号

`addresses`文件中记录了各节点的地址，将里面的地址填写到`genesis.addresses`中即可。

### 修改网络信息
网络配置修改在`network.toml`中。

`N`字段修改为节点数量，默认是4个。

`id`字段代表了节点的顺序id，范围在1-N，节点间不能重复。

`addr`和`id`分别是各节点的地址和id，其中`/ip4/`后填写节点所在服务器的ip地址，`/tcp/`后填写节点的libp2p端口（端口不重复即可），`/p2p/`后填写Libp2p的id，具体的值从pids文件中按照顺序获取。

### 启动

启动前，需要将build目录下的bitxhub二进制拷贝到node1-node4目录下，将build目录下的raft.so插件拷贝到node1-node4下的plugins下。

分别进入到node1-node4目录下，执行以下命令即可启动：

```shell
bash start.sh
```
