# BitXHub v1.4.0
## 中继链V1.4.0
#### 新功能
- 中继链接入了强鲁棒拜占庭容错算法RBFT，其失效数据恢复、动态节点增删等机制保证了中继链天然的高可用性；
- 增加Sync Protocol节点恢复机制，保证中继链节点状态的一致性；
- 中继链DID内置合约开发及优化；
- 新增事务管理模块，提供多链消息表和中继多签资产交换两种跨链事务。使用多链消息表技术实现一对一和一对多跨链；
- 中继链的GRPC通信增加TLS加密功能；
- 中继链mempool实现，添加交易池的超时处理机制

#### 缺陷修复

- 修复创建私钥必须在已有目录的bug；
- 修复中继链Solo模式下不支持交易池的bug；
- 修复bitxhub client validators命令返回错误的Bug；
- 修复raft选举异常的bug

## 跨链网关V1.4.0

#### 新功能

- 跨链网关的GRPC通信增加TLS加密功能；
- 重构跨链网关执行、监听和同步等模块；
- 跨链网关主备模块化；
- 适配BCOS；

#### 缺陷修复

- 修复Pier中继模式启动依赖应用链的Bug；
- 修复fabric插件中轮询chaincode交易过多的bug；
- 修复删除跨链网关存储重启导致Get out Message not found的Bug；
- 修复Pier重启恢复过程中生成IBTP回执时IBTP参数设置错误的Bug

## 运维工具

#### 新功能

- Goduck命令格式调整优化；
- Goduck添加远程部署pier和bitxhub的结果检查；
- Goduck的status命令开发，添加docker组件信息；

#### 缺陷修复

- 修复Goduck pier config命令中id参数无效的bug；
- 修复Goduck info命令中判断已退出但配置目录仍存在的组件状态错误的bug；
- 修复Goduck远程重复部署pier时出现的文件名不一致的bug；
- 修复Goduck pier直连模式peers信息错误的bug；

## 其它

无



