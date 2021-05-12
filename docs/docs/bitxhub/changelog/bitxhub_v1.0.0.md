# BitXHub v1.0.0
## 中继链V1.0.0
#### 新功能
- 接入区块链并适配 Ethereum；
- 添加Prometheus监控模块并且丰富相关埋点；
- 添加Solo版本的Dockerfile；
- 实现Docker Compose启动集群功能；
- 内置合约添加查询跨链索引的功能；
- 添加Rust验证规则模版；
- 完善区块头交易Merkle树的构造。

### 开发工具
- 新增了运维工具项目GoDuck，后续将持续进行优化；
- 添加快速启动Fabric网络和Ethereum网络的功能；
- 添加Prometheus+Grafana快速启动docker-compose的配置文件；
- 加入BitXHub和Pier配置快速生成功能；
- 实现二进制管理功能。

### 缺陷修复
- 修复验证规则无法解析的Bug；
- 修复应用链注册时ID未找到的Bug；
- 修复Dockerfile未编译raft共识算法插件的Bug；

## 跨链网关V1.0.0
- 重构跨链网关拉取中继链上的跨链交易时Wrapper的数据结构；
- 添加p2p ID 的命令行子命令；

### 缺陷修复
- 修复跨链网关和插件依赖库版本冲突的Bug；
- 修复验证来自中继链的跨链交易为空时处理的Bug。