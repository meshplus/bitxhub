# BitXHub v1.3.0

## 中继链V1.3.0

#### 新功能

- 重构跨链验证
- 中继链PendingPool功能优化，提升性能；
- 中继链实现交易执行并行化；
- 中继链跨链验证去除IBTP的Proof，减少存储压力；

#### 缺陷修复

无

## 跨链网关V1.3.0

#### 新功能

- 适配中继链的PendingPool模式，维护发送跨链交易的Nonce并且顺序递增；
- 优化跨链网关直连模式的性能，实现IBTP异步发送和接收；
- 实现初版pier-client-fake以支持Pier直连模式测试；

#### 缺陷修复

- 修复Pier在中继链和应用链跨链合约数据不一致情况下一直向中继链发交易的Bug

## 运维工具

#### 新功能

- Goduck添加根据版本远程部署BITXHUB和Pier的功能；
- Goduck支持prometheus监控启动；
- Goduck支持按版本启动playground

#### 缺陷修复

- 修复linux系统上docker模式启动pier失败的bug；
- 修复key命令path参数无效的bug
- 修复info命令无响应的bug

## 其它

无