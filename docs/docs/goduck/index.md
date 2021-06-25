# Goduck运维小工具

## 1 安装

### 1.1 获取源码
下载源码并切到稳定版本release-1.0
```
git clone git@github.com:meshplus/goduck
cd goduck
git checkout release-1.0
```

### 1.2 编译安装

```
make install
```

### 1.3 初始化

```
goduck init
```

使用之前一定要先初始化

## 2 使用

### 2.1 命令格式

```
goduck [global options] command [command options] [arguments...]
```

**command**

- `deploy`          远程部署bitxhub和pier
- `version`          查看组件版本信息
- `init`          初始化配置
- `status`          列举实例化组件状态
- `key`          创建并展示密钥信息
- `bitxhub`          启动或关闭bithxub节点
- `pier`          有关pier的操作
- `playground`          一键启动跨链组件
- `info`          展示跨链基本信息
- `prometheus`          启动或关闭prometheus
- `help, h`

这些命令中，比较重要的是init（使用前一定要初始化）、status（查看当前运行组件状态）、bitxhub、pier。

**global options**

- `--repo value`          goduck配置文件默认存储路径
- `--help, -h`

### 2.2 关于BitXHub的操作

```
goduck bitxhub command [command options] [arguments...]
```

#### 2.2.1 启动BitXHub节点

```
goduck bitxhub start
```

该命令会初始化并启动BitXHub节点，如果有已启动的BitXHub节点会执行失败。执行成功后提示如下：

```
1 BitXHub nodes at /Users/fangbaozhu/.goduck/bitxhub are initialized successfully
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh up v1.4.0 solo binary 4
Start bitxhub solo by binary
===> Start bitxhub solo successful
```

这是默认启动方式，也可以携带参数自定义启动，参数设置如下：

- `--type value`          配置类型，binary或docker （默认：“binary”）
- `--mode value`          配置模式，solo或cluster （默认：“solo”）
- `--num value`          节点个数，只在cluster 模式下作用（默认：4）
- `--tls`          是否启动TLS, 只在v1.4.0+版本有效 (default: false)
- `--version value`          BitXHub版本 (default: "v1.4.0")
- `--help, -h`

#### 2.2.2 为BitXHub节点生成配置文件

```
goduck bitxhub config
```

该命令默认初始化4个BitXHub节点。执行成功后当前文件夹会生成相关证书、私钥文件以及四个节点的文件夹，成功提示如下：

```
initializing 4 BitXHub nodes at .
4 BitXHub nodes at . are initialized successfully
```

You can see the following in the current directory：

```
.
├── agency.cert
├── agency.priv
├── ca.cert
├── ca.priv
├── key.priv
├── node1/
├── node2/
├── node3/
└── node4/
```

可以携带参数自定义配置情况：

- `--num value`          节点个数，只在cluster 模式下作用（默认：4）
- `--type value`          配置类型，binary或docker （默认：“binary”）
- `--mode value`          配置模式，solo或cluster （默认：“solo”）
- `--ips value`          节点IP, 默认所有节点为127.0.0.1, e.g. --ips "127.0.0.1" --ips "127.0.0.2" --ips "127.0.0.3" --ips "
  127.0.0.4"
- `--target value`          节点的配置文件位置（默认：当前目录）
- `--tls`          是否启动TLS, 只在v1.4.0+版本有效 (default: false)
- `--version value`          BitXHub版本 (default: "v1.4.0")
- `--help, -h`

#### 2.2.3 关闭BitXHub节点

```
goduck bitxhub stop
```

该命令会关闭所有启动的BitXHub节点。执行成功后会提示里会给出关闭节点的id：

```
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh down
===> Stop bitxhub
node pid:65246 exit
```

#### 2.2.4 清除BitXHub节点

```
goduck bitxhub clean
```

该命令会清除bitxhub节点的配置文件。如果bitxhub节点没有关闭，会先关闭节点再清除配置文件。

当bitxhub solo节点成功在关闭后执行此命令，会打印出提示如下：

```
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh clean
===> Stop bitxhub
===> Clean bitxhub
remove bitxhub configure nodeSolo
```

当bitxhub solo节点成功在未关闭的情况下执行此命令，会打印出提示如下：

```
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh clean
===> Stop bitxhub
node pid:65686 exit
===> Clean bitxhub
remove bitxhub configure nodeSolo
```

### 2.3 关于pier的操作

```
GoDuck pier command [command options] [arguments...]
```

#### 2.3.1 启动pier

```
goduck pier start
```

参数设置如下：

- `--chain value`          应用链类型，ethereum 或 fabric（默认：ethereum）
- `--cryptoPath value`          crypto-config路径, 只对fabric链有效, e.g $HOME/crypto-config
- `--pier-type value`          pier类型，docker或者binary (默认: "docker")
- `--version value`          pier版本 (默认: "v1.4.0")
- `--tls value`          是否启动TLS, true or false, 只对v1.4.0+版本有效 (默认: "false")
- `--http-port value`          pier的http端口号, 只对v1.4.0+版本有效 (默认: "44544")
- `--pprof-port value`          pier的pprof端口号, 只对binary有效 (默认: "44550")
- `--api-port value`          pier的api端口号, 只对binary有效 (默认: "8080")
- `--overwrite value`          当本地默认路径存在pier配置文件时是否重写配置 (默认: "true")
- `--appchainIP value`          pier连接的应用链ip (默认: "127.0.0.1")
- `--help, -h`

在使用此命令之前，您需要启动一个相同版本的BitxHub和一个需要pier连接的应用链。如果应用链或bitxhub不存在，pier将无法启动，打印提示如下:

```
exec:  /bin/bash run_pier.sh up -m ethereum -t docker -r .pier_ethereum -v v1.4.0 -c  -p 44550 -a 8080
===> Start pier of ethereum-v1.4.0 in docker...
===> Start a new pier-ethereum container
===> Wait for ethereum-node container to start for seconds...
136d323b1418a026101515313dbbdafee240ac0f0c0d63b4f202304019e13e24
===> Start pier fail
```

如果要连接的应用链和BitxHub已经启动，且PIER已经启动成功，打印提示如下:

```
exec:  /bin/bash run_pier.sh up -m ethereum -t docker -r .pier_ethereum -v v1.0.0-rc1 -c  -p 44550 -a 8080
===> Start pier of ethereum-v1.0.0-rc1 in docker...
===> Start a new pier-ethereum container
===> Wait for ethereum-node container to start for seconds...
351bd8e8eb8d5a1803690ac0cd9b77c274b775507f30cb6271164fb843442bfd
===> Start pier successfully
```

#### 2.3.2 关闭pier

```
goduck pier stop
```

该命令可以关闭pier，可以通过携带参数指定关闭那种类型应用链的pier

#### 2.3.3 清除Pier

```
goduck pier clean
```

该命令可以清除pier的配置文件，如果pier还没有关闭，该命令会先关闭pier再清除其配置文件。

#### 2.3.4 生成Pier的配置文件

```
goduck pier config
```

- `--mode value`          配置模式, 直连模式或者中继模式(默认: "direct")
- `--type value`          配置类型, binary或者docker (默认: "binary")
- `--bitxhub value`       BitXHub的地址，只在中继模式有效
- `--validators value`    BitXHub的验证人地址，只在中继模式有效, 例如 --validators "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"
  --validators "0x79a1215469FaB6f9c63c1816b45183AD3624bE34" --validators "0x97c8B516D19edBf575D72a172Af7F418BE498C37"
  --validators "0x97c8B516D19edBf575D72a172Af7F418BE498C37"
- `--port value`          pier的端口号，只在直连模式有效 (默认: 5001)
- `--peers value`          连接节点的地址，只在直连模式有效, 例如 --peers "
  /ip4/127.0.0.1/tcp/4001/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5ngmL"
- `--connectors value`     待连接节点的地址，只在v1.4.0+版本的union模式有效, 例如 --connectors "
  /ip4/127.0.0.1/tcp/4001/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5ngmL" --connectors "
  /ip4/127.0.0.1/tcp/4002/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5abcD"
- `--appchain-type value`  应用链类型, ethereum或者fabric (默认: "ethereum")
- `--appchain-IP value`    应用链IP地址 (默认: "127.0.0.1")
- `--target value`         生成配置文件路径 (默认: ".")
- `--tls value`            是否开启TLS，只在v1.4.0+版本有效 (默认: "false")
- `--http-port value`      pier的http端口号, 只在v1.4.0+版本有效 (默认: "44544")
- `--pprof-port value`     pier的pprof端口号 (默认: "44550")
- `--api-port value`       pier的api端口号 (默认: "8080")
- `--cryptoPath value`     crypto-config文件的路径，只对fabric链有效，例如 $HOME/crypto-config (default: "
  $HOME/.goduck/crypto-config")
- `--version value`        pier版本 (默认: "v1.4.0")
- `--help, -h` 