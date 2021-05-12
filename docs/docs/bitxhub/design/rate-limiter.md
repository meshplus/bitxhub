# API流控设计

## 实体定义

### 漏桶算法

漏桶算法的原理可以类比为往一个固定大小的桶里盛水，同时，水从桶底漏洞以固定速度流出，如果加水过快，则直接溢出，如下图所示。它可以应用于网络传输限流，计算机每发送一个数据包，如果桶内未满，则把数据包放入桶里，如果桶内已满，则丢弃数据包，与此同时，以固定速度从桶内取出数据包，发送到网络，从而达到**强行限制数据平均传输速率**的目的。

![](../../../assets/faucet.png)

漏桶算法常用于将突发或不稳定流量整形为以固定速度在网络中传输的流量。

### 令牌桶算法

对于要求允许某种程度的突发传输，漏桶算法显然无法满足需求，而令牌桶可以做到这一点。令牌桶算法同样定义了一个固定大小的桶，桶里最多可容纳 b 个令牌，每当有数据包需要发送时，要从桶里取出对应数量的令牌才能发送，如果桶里没有足够令牌，则无法发送。与此同时，以固定速度 r 往桶里添加新令牌，当桶里令牌数已经达到 b 个时，丢弃新令牌。

![](../../../assets/token_bucket.png)

令牌桶算法非常适合于针对系统外部请求的限流，当桶内有足够多令牌时，系统在某一时刻可以同时接收并处理多个请求，充分利用到系统资源。

总结来说，令牌桶限流允许突发流量，对于请求的限流、网络带宽限流，更能充分利用系统资源和网络资源，是适用于区块链底层平台系统流控的一种限流方法。



# 2.详细设计

区块链节点的入口流量大体分为两种：

一种为客户端发送过来的请求，请求可能为区块链数据查询、发送新交易、合约操作等（下文将简称为“客户端请求”）。节点接收到客户端请求后，首先需要从网络IO流中读取到请求的字节内容，然后反序列化字节内容为结构化内容，最后根据结构化请求体调用对应的API逻辑；

另一种为其他区块链节点发过来的网络消息，区块链系统底层是由多个共识节点组成的共识网络，节点间通过计算机网络进行信息传输（下文将简称为“节点间网络消息”）。节点接收到对端节点发送过来的网络消息后，根据消息类型，抛给对应的模块去处理。因此，不仅需要对客户端请求进行流量控制，防止大量突发外部请求都往同一个节点发送，耗尽目标节点资源导致目标节点服务瘫痪。还要对节点接收到的网络消息进行限流，防止节点在高负载下，前面的消息涉及的系统逻辑还未处理完，还源源不断地接收和缓存后面到来的消息，甚至导致节点内存溢出。





## 2.1 RPC流控

**交易拦截器限流**

- 限制请求速率：通过令牌桶限流算法控制请求速率，并限制节点最多可同时接收并处理的请求数。

- 节点高负载下拒绝新交易：当节点交易池已满或者处于异常、异常恢复状态无法进行正常共识时，拒绝来自客户端发送过来的新交易，避免交易参数检查、交易验签带来的CPU消耗。

令牌桶算法：https://github.com/juju/ratelimit

GRPC提供Interceptor用于拦截请求：[__https://github.com/grpc-ecosystem/go-grpc-middleware/blob/master/ratelimit/ratelimit.go__](https://github.com/grpc-ecosystem/go-grpc-middleware/blob/master/ratelimit/ratelimit.go)

```go
// Limiter defines the interface to perform request rate limiting.
// If Limit function return true, the request will be rejected.
// Otherwise, the request will pass.
type Limiter interface {
	Limit() bool
}

// UnaryServerInterceptor returns a new unary server interceptors that performs request rate limiting.
func UnaryServerInterceptor(limiter Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if limiter.Limit() {
			return nil, status.Errorf(codes.ResourceExhausted, "%s is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod)
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a new stream server interceptor that performs rate limiting on the request.
func StreamServerInterceptor(limiter Limiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if limiter.Limit() {
			return status.Errorf(codes.ResourceExhausted, "%s is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod)
		}
		return handler(srv, stream)
	}
}

```

结合令牌桶和GRPC的interceptor实现对于API的流控：

```go
type Limitor struct {
  TokenBucket
}

// returns a new token bucket that fills at the rate of one token every fillInterval,
// up to the given maximum capacity.Both arguments must be positive. 
// The bucket is initially full.
func NewLimitor(fillInterval time.Duration, capacity int64) *Limitor


// allows the specification of the quantum size - quantum tokens are added every fillInterval.
func NewLimitorWithQuantum(fillInterval time.Duration, capacity, quantum int64) *Limitor


func (l *Limitor) Limit() bool {
	return l.TakeAvailable() == 0
}

```



在bitxhub.toml添加相对应的配置

```text
limitor:
	interval: 10ms
	quantum: 100
    capacity: 10000
```

## 2.2 P2P流控

**带权消息分发器限流**

主要用来限制非关键模块的流量，防止带宽、CPU和内存都被非关键模块给占用。具体做法是为各个需要进行网络通信的模块分配带缓存空间的读（R）、写（W）管道，根据模块在系统中所占权重为其管道分配不同的缓存大小。

消息分发器收到一条来自底层P2P网络的网络消息，根据消息类型将消息分发给对应模块进行处理。这条消息首先分发给模块对应的 R 管道，模块再从 R 管道按照FIFO原则取出消息，执行相关逻辑，如果 R 管道消费速度慢于生产速度，导致分发消息时 R 管道已满，则说明模块内部已处于高负载，丢弃这条消息。为了保证达到系统限流目的，模块从 R 管道取出消息并处理消息的过程必须是串行的，而模块间的消息并行处理，互不干扰。

举个例子，当非关键模块处于高负载处理能力变慢时，其 R 管道虽然占满，但是不会影响共识模块消息的处理速度，同时又由于不同模块根据权重 R 管道大小不同，一定程度上防止节点一直处理非关键模块消息占用过多系统资源而导致共识模块消息无法得到及时处理。

带权消息分发一定程度上降低了各模块由于处理能力差异而相互干扰，提高系统网络消息并行处理能力，保证核心网络消息不被非核心网络消息占去全部系统资源，同时，系统高负载下自动丢弃新接收到的网络消息，防止系统负载过高而崩溃。
# 















































