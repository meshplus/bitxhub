# 硬件环境
## 服务器

| 服务器| 配置要求
---|---
CPU | 64位8核及其以上
存储 | 大于等于500G（需要支持扩容）
内存 | 大于等于16GB


## 软件环境
**操作系统**

目前BitXHub支持的操作系统以及对应版本号如下：

| 操作系统| 系统版本|系统架构
---|---|---
RHEL | 6或更新 |amd64，386
CentOS | 6或更新| amd64，386
SLES  |11SP3或更新|amd64，386
Ubuntu |14.04或更新|amd64，386
MacOS |10.8或更新|amd64，386

## 安装go
Go为Mac OS X、Linux和Windows提供二进制发行版。如果您使用的是不同的操作系统，您可以下载Go源代码并从源代码安装。

在这里下载适用于您的平台的最新版本Go：[下载](https://golang.org/dl/) - 请下载 1.13.x 或更新

请按照对应于您的平台的步骤来安装Go环境：[安装Go](https://golang.org/doc/install#install) ，推荐使用默认配置安装。

- 对于Mac OS X 和 Linux操作系统，默认情况下Go会被安装到/usr/local/go/，并且将环境变量GOROOT设置为该路径/usr/local/go.
```shell
export GOROOT=/usr/local/go
```


- 同时，请添加路径 GOROOT/bin 到环境变量PATH中，可以使Go工具正常执行。
```shell
export PATH=$PATH:$GOROOT/bin
```


## 设置GOPATH
您的Go工作目录 (GOPATH) 是用来存储您的Go代码的地方，您必须要将他跟您的Go安装目录区分开 (GOROOT)。
以下命令是用了设置您的GOPATH环境变量的，您也可以参考Go官方文档，来获得更详细的内容: [https://golang.org/doc/code.html](https://golang.org/doc/code.html).

对于 Mac OS X 和 Linux 操作系统 将 GOPATH 环境变量设置为您的工作路径：
```shell
export GOPATH=$HOME/go
```


同时添加路径 GOPATH/bin 到环境变量PATH中，可以使编译后的Go程序正常执行。
```shell
export PATH=$PATH:$GOPATH/bin
```


由于我们将在Go中进行一系列编码，您可以将以下内容添加到您的~/.bashrc文件中：
```shell
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin:$GOROOT/bin
```
