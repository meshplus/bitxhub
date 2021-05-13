# 环境准备

环境准备是部署和使用BitXHub跨链平台的第一步，主要是说明BitXHub及相关组件运行的硬件配置和软件依赖，您需要在部署BitXHub平台之前确认机器满足下述的要求。

## 硬件

配置| 推荐配置 | 最低配置 
---|---|---
CPU | 2.4GHz *8核或以上 |1.5GHz *4核
 内存 | 16GB或以上                 | 8GB         
 存储 | 500G或以上（需要支持扩容） |100G
带宽 | 10Mb |2Mb

## 操作系统支持

目前BitXHub支持的操作系统以及对应版本号如下：

操作系统| 发行版本 | 系统架构 
---|---|---
RHEL | 6或更新 |amd64，386
CentOS | 6或更新| amd64，386
SUSE  |11SP3或更新|amd64，386
Ubuntu |14.04或更新|amd64，386
MacOS |10.8或更新|amd64，386

**说明：为了更好的部署安装体验，我们建议您选用CentOS 8.2、Ubuntu 16.04和MacOS 10.15来进行部署安装。**

## 软件依赖

#### Go环境

BitXHub作为golang项目，需要安装和配置Go环境，您可以在这里下载适用于您的平台的最新版本Go二进制文件：[下载](https://golang.org/dl/) -（请下载 1.13.x 或更新的稳定版本），也可以下载Go源代码并从源代码进行安装，这里不再赘述。

下载完成后您需要安装Go，可以参考官方文档：[安装Go](https://golang.org/doc/install#install) ，推荐使用默认配置安装即可，

- 对于Mac OS X 和 Linux操作系统，默认情况下Go会被安装到/usr/local/go/，并且将环境变量GOROOT设置为该路径/usr/local/go.
```shell
export GOROOT=/usr/local/go
```


- 同时，由于我们可能将在Go中进行一系列编译操作，还需要设置GOPATH等，您可以将以下内容添加到您的~/.bashrc文件中：
```shell
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin:$GOROOT/bin
```

**说明：以上配置均是参考，您可以根据自己的实际情况进行安装配置。**

#### Docker

如果您想使用容器来部署bitxhub平台，则需要提前安装好Docker，推荐安装18.03或更新的稳定版本，具体的安装方法可以参考官方文档：[安装Docker](https://docs.docker.com/engine/install/)



恭喜您！环境确认和准备完成，接下来可以愉快地开始部署BitXHub平台啦！！！