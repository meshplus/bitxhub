# 共识算法插件方案
## 项目结构

该项目为BitXHub提供共识算法的插件化，具体项目结构如下：

```text
./
├── Makefile //编译文件
├── README.md
├── build
│ └── rbft.so //编译后的共识算法二进制插件
├── go.mod
├── go.sum
├── order.toml //共识配置文件
└── rbft //共识算法代码
    ├── config.go
    ├── node.go
    └── stack.go
```

其中注意在`go.mod`中需要引用BitXHub项目源码，需要让该插件项目与BitXHub在同一目录下（建议在$GOPATH路径下）。

```none
replace github.com/meshplus/bitxhub => ../bitxhub/
```

## 编译Plugin

我们采用GO语言提供的插件模式，实现`BitXHub`对于Plugin的动态加载。

编写`Makefile`编译文件：

```shell
SHELL := /bin/bash
CURRENT_PATH = $(shell pwd)
GO  = GO111MODULE=on go
plugin:
   @mkdir -p build
   $(GO) build --buildmode=plugin -o build/rbft.so rbft/*.go
```

运行下面的命令，能够得到 `rbft.so`文件。

```shell
$ make plugin
```

修改节点的`bitxhub.toml`

```none
[order]
  plugin = "plugins/rbft.so"
```

将你编写的动态链接文件和`order.toml`文件，分别放到节点的plugins文件夹和配置文件下。

```text
./
├── api
├── bitxhub.toml
├── certs
│ ├── agency.cert
│ ├── ca.cert
│ ├── node.cert
│ └── node.priv
├── key.json
├── logs
├── network.toml
├── order.toml //共识算法配置文件
├── plugins
│ ├── rbft.so //共识算法插件
├── start.sh
└── storage
```

结合我们提供的`BitXHub`中继链，就能接入到跨链平台来。