# 整体说明

这篇文档是对BitXHub跨链系统部署的整体说明，主要是介绍BitXHub跨链系统的一般部署流程和部署架构。

## 1. 环境准备

环境准备是部署和使用BitXHub跨链平台的第一步，主要是说明BitXHub及相关组件运行的硬件配置和软件依赖，您需要在部署BitXHub平台之前确认服务器满足硬件和软件的要求，具体信息请查看《环境说明》。

## 2. 单中继链部署架构

一般来说，单中继链架构适用于大多数部署场景，建议您使用此种部署架构来体验BitXHub跨链系统。如下图所示，部署完bitxhub节点集群（也可以是solo模式的单机节点），两条或多条应用链上部署好跨链合约，然后通过各自的跨链网关接入到中继链中，完成跨链系统的搭建。

<img src="../../assets/single_bitxhub.png" alt="single_bitxhub" style="zoom:50%;" />

在明晰了部署架构之后，这里再说明下部署的一般流程，如下图所示。

<img src="../../assets/deploy_flow.png" alt="deploy_flow" style="zoom:50%;" />



## 3. 跨链网关直连部署架构

跨链网关直连部署架构是指不使用中继链，两方的应用链通过跨链网关与对方直接连接，部署结构如下图所示，除了无需部署中继链节点之外，部署的流程上与上一章基本一致。

<img src="../../assets/direct_pier.png" alt="direct_pier" style="zoom:50%;" />

