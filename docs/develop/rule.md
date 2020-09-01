# 验证引擎规则编写规范

## 智能合约

BitXHub提供wasm虚拟机来运行验证规则的智能合约，应用链方可以灵活自主实时的部署和更新验证规则的智能合约。

验证规则的智能合约的主要目的是：

1. 解析IBTP的Proof字段，不同的应用链在Pier发送IBTP时会对自身的验证信息做不同形式的封装，智能合约需要对封装的Proof进行解析，得到需要验签的参数；

1. 对得到的验签的参数采用对应的签名算法进行验签。

BitXHub采用wasmer作为wasm的虚拟机，因此验证规则智能合约需要遵循wasi的规范，验证规则合约需要使用指定的方法名。规则合约的编写者可以通过以下两种方式来进行合约的编写，入口函数为verify：

3. 编写智能合约时可以引入BitXHub提供给wasm的签名函数。

```rust

// BitXHub provides ecdsa signature verify method

// Take this as an example

extern {

	fn ecdsa_verify(...) -> ... ;

}



#[no_mangle]

fn verify(...) -> ... {

	...

}

```

4. 应用链方在合约中自己编写验签方法，并编译成wasm智能合约。

## 规则部署

应用链在注册到BitXHub以后，需要将自身对跨链交易的背书验证规则注册到BitXHub上，不然跨链交易的请求是无法通过的。所以应用链在注册到BitXHub后想要进行跨链请求就必须将自身的验证规则合约部署到BitXHub上。

首先应用链需要通过部署合约的方式将验证规则的合约部署到BitXHub上，部署完后WASM虚拟机会返回一个合约的地址，这个地址就是验证引擎需要调用的地址。客户端在拿到部署合约的结果（验证规则的地址）后。需要调用BitXHub的内置合约，把验证规则的地址和自己应用链的ID关联起来。

之后每一次跨链交易产生，验证引擎就会通过这个合约地址拿到验证规则合约，将合约载入到虚拟机中对验证信息进行校验。


## 规则调用

当BitXHub处理一条跨链交易的时候，需要通过验证引擎验证交易的有效性，验证引擎会根据来源链的ID找到对应的验证规则，加载到WASM虚拟机中，WASM虚拟机会通过验证的入口函数（Verify）调用验证规则的智能合约。运行验证规则，返回验证结果。


## 证书签名

一般来说，验证签名需要证书或者公钥，验证引擎的背书公钥和证书是在应用链注册到BitXHub中的时候上传到BitXHub中的。

## 合约编写

### 参数传递

由于wasm虚拟机对传入传出的数据类型有限制，现行支持的类型不足以支持将验证信息直接传入到wasm虚拟机中，为了使wasm合约编写者能对验证信息直接读写，BitXHub提供了一套验证引擎合约的模板，模板对各个入口进行封装，用户只需要在入口函数中添加自己的验证逻辑即可。

wasm现阶段不支持对字符串或者是byte数组的输入输出，所以如何将证明信息传入到wasm虚拟机需要使用特定的方法。BitXHub的模板提供了对虚拟机内存访问的接口，通过该接口可以在虚拟机外部直接读写虚拟机的内存，从而变相的达到传入传出字符串的目的。

```rust

#[no_mangle]

pub extern fn allocate(size: usize) -> *mut c_void {

    let mut buffer = Vec::with_capacity(size);

    let pointer = buffer.as_mut_ptr();

    mem::forget(buffer);



    pointer as *mut c_void

}



#[no_mangle]

pub extern fn deallocate(pointer: *mut c_void, capacity: usize) {

    unsafe {

        let _ = Vec::from_raw_parts(pointer, 0, capacity);

    }

}

```

### 引用BitXHub提供的方法

由于wasm交叉编译对一些动态链接库的支持性低，许多合约编写者可能无法直接调用高级语言的算法库进行合约编写。BitXHub提供了一些基础的密码学的库方便智能合约的编写。

原理介绍：

wasm虚拟机允许外部import函数让wasm来调用，本质上是通过cgo的方式，将执行函数的入口指针提供给wasm虚拟机，这样wasm就可以调用Go的函数了。

虚拟机的内存是Go程序内存的一部分，虚拟机内部无法访问到Go程序的内存，所以wasm函数返回的指针都是基于虚拟机内存的偏移量，这就造成了Go的函数无法通过wasm函数返回的指针直接访问到指针对应的变量。我们这边的解决方案是将虚拟机内存的起始地址作为所有import函数的上下文，这样所有import函数都可以通过这个上下文和wasm函数返回的指针来定位变量的地址了。

```go

  data := ctx.Data().(map[int]int)

  memory := ctx.Memory()

  signature := memory.Data()[sig_ptr : sig_ptr+70]



```

## 验证引擎与WASM虚拟机

### WASM虚拟机实例

当BitXHub的Executor创建时，验证引擎的实例也就被创建了。当每一次有跨链交易的请求到达时，验证引擎会根据交易来源链的ID拿到存储在链上的验证规则合约，然后会实例化一个WASM的虚拟机，将该验证规则合约加载到虚拟机中。

由于BitXHub本身提供很多密码学的方法，当WASM虚拟机实例化之前需要将这些方法import进来，从而可以提供给合约进行调用。

```go

imports, _ := wasm.NewImports().Append("*", *, C.*)

instance, _ := wasm.NewInstanceWithImports(bytes, imports)

```

### 验证引擎传参

验证引擎需要将IBTP解析出的Proof字段传输到WASM中以便合约能够执行，由于WASM虚拟机对传入数据的类型有非常严格的限制，现在只支持整形和浮点参数的传入，所以字符串和byte数组的信息无法直接被合约方法调用。

验证引擎首先需要根据传入参数的大小长度在WASM的虚拟机中划分出一块内存作为输入读取内存，并获得输入读取内存的指针，并且划分出一块内存作为输出读取内存，同样的获得输出内存指针。指针都可以用i32的整型数据变量来表示。

内存的划分和释放方法由BitXHub提供的合约模板提供，入口函数为allocate和deallocate。


### 验证引擎验证

根据验证引擎传参章节的描述，验证引擎在虚拟机划出输入输出的内存以后，就会调用验证规则的入口函数start，将输入输出的内存指针作为穿参让入口函数执行。

合约模板已经帮合约编写者处理好了参数在wasm中的输入输出问题。合约编写者可以直接拿到Proof字段和Validator的信息，只需要对这些信息进行验证规则的编写即可。

```rust

pub fn verify<'a>(proof: &[u8], validators: &Vec<&'a str>) -> bool {

	// Code your rule here with variables: proof and validators

	...

}

```
