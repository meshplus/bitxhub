# Premo使用文档

Premo是BitXHub跨链系统测试框架，目前支持系统集成测试、接口测试和压力测试

## 安装

#### 获取源码

```shell
git clone git@github.com:meshplus/premo.git
```

#### 编译

进入premo工程目录:

```shell
cd premo
make install
```

## 初始化

```shell
premo init
```

## 基本使用

```text
premo [global options] command [command options] [arguments...]
```

#### COMMANDS:

- `init` init config home for premo
- `version` Premo version
- `test` test bitxhub function
- `pier` Start or stop the pier
- `bitxhub` Start or stop the bitxhub cluster
- `appchain` Bring up the appchain network
- `interchain` Start or Stop the interchain system
- `status` List the status of instantiated components
- `help, h` Shows a list of commands or help for one command

#### GLOBAL OPTIONS:

- `--repo value`  Premo storage repo path
- `--help, -h`    show help (default: false)

## 集成测试

进入premo的工程目录

```shell
cd premo
make bitxhub-tester
```

注意：集成测试默认前置条件是本机已启动bitxhub四节点集群（可在bitxhub工程目录下通过`make cluster`命令启动）

## 接口测试

进入premo的工程目录

```shell
cd premo
make http-tester
```

注意：集成测试默认前置条件是本机已启动bitxhub四节点集群（可在bitxhub工程目录下通过`make cluster`命令启动）

## 压力测试

test命令用于压测bitxhub的TPS性能。使用下面的命令获取使用帮助信息：

```text
premo test --help
```

帮助信息如下：

```text
NAME:
   premo test - test bitxhub function

USAGE:
   premo test [command options] [arguments...]

OPTIONS:
   --concurrent value, -c value           concurrent number (default: 100)
   --tps value, -t value                  all tx number (default: 500)
   --duration value, -d value             test duration (default: 60)
   --key_path value, -k value             Specific key path
   --remote_bitxhub_addr value, -r value  Specific remote bitxhub address (default: "localhost:60011")
   --type value                           Specific tx type: interchain, data, transfer (default: "transfer")
   --help, -h                             show help (default: false)

```

`--concurrent`或者`-c`指定并发量；

`--tps`或者`-t`指定每秒交易数量；

`--duration`或者`-d`指定压测时间；

`--key_path`或者`-k`指定私钥路径；

`--remote_bitxhub_addr`或者`-r`指定bitxhub的地址；

`--type`指定交易类型，其中`transfer`是普通转账交易，`data`是调用BVM交易，`interchain`是跨链交易；

压测完成后会打印压测的实际情况：

```text
$ premo test -c 50 -t 3000 -d 1000
INFO[0000] Premo configuration                           concurrent=50 duration=1000 tps=3000 type=transfer
INFO[0000] generate all bees                             number=50
2020-08-10 13:51:11 [INFO] [$(GOPATH)/src/meshplus/premo/internal/bitxhub/bitxhub.go:92] starting broker
INFO[0000] start all bees                                number=50
INFO[0001] current tps is 834.000000
INFO[0002] current tps is 1346.000000
INFO[0003] current tps is 2469.000000
INFO[0004] current tps is 1732.000000
INFO[0005] current tps is 2221.000000
INFO[0006] current tps is 2068.000000
INFO[0007] current tps is 1145.000000
INFO[0008] current tps is 1626.000000
INFO[0009] current tps is 2425.000000
INFO[0010] current tps is 1703.000000
INFO[0011] current tps is 1772.000000
INFO[0012] current tps is 1823.000000
INFO[0013] current tps is 1213.000000
INFO[0014] current tps is 1974.000000
INFO[0015] current tps is 1965.000000
INFO[0016] current tps is 2001.000000
INFO[0017] current tps is 975.000000
INFO[0018] current tps is 1505.000000
INFO[0019] current tps is 2338.000000
INFO[0020] current tps is 1704.000000
INFO[0021] current tps is 1270.000000
INFO[0022] current tps is 2418.000000
INFO[0023] current tps is 1673.000000
INFO[0024] current tps is 997.000000
INFO[0025] current tps is 1935.000000
INFO[0026] current tps is 1840.000000
INFO[0027] current tps is 710.000000
INFO[0028] current tps is 1041.000000
INFO[0029] current tps is 837.000000
INFO[0030] current tps is 1403.000000
received interrupt signal, shutting down...
INFO[0030] finish testing                                duration=30.468557927 number=50880 tps=1669.9181872450927 tx_delay=968.0890066430818
```

