# 验证规则
BitXHub的验证引擎的规则除了内置的规则以外主要是通过WASM字节码来运行逻辑的。该文档将会以rust为例说明验证规则的编写方法。

## 1. 下载规则模板

在GitHub的BitXHub项目已经提供了规则编写[__模板__](https://github.com/meshplus/bitxhub/tree/master/example/rule_example)，用户可以下载这个模板然后对模板进行改造来形成自己所希望运行的规则逻辑。

## 2. 规则模板的目录结构

```text
.
├── Cargo.toml
└── src
    ├── app
    │   ├── contract.rs
    │   └── mod.rs
    ├── crypto
    │   ├── ecdsa.rs
    │   ├── fabric.rs
    │   └── mod.rs
    ├── lib.rs
    ├── memory.rs
    └── model
        ├── mod.rs
        └── transaction.rs
```

从上面的目录结构图中，我们可以看到除去rust项目自己本身自带的Cargo.toml文件以外了，目录的源码目录src下面有三个文件夹和两个rs文件，其中lib.rs文件存放着能够被wasm虚拟机export识别的函数，在规则编写中用户不需要理会这个文件，这个是由模板自己预先编写好的能够读取外部输入参数的函数。还有就是memory.rs这个文件是模板用来处理wasm虚拟机的字符串和字节数组输入输出使用的，也是用户不需要理会的文件。

除了上述的两个rs文件，还有三个文件夹，分别是app, crypto以及model。model文件夹可以让用户存放一些他们规则逻辑中需要使用的数据原型，在模板当中我们使用了fabric的ChaincodeAction的proto作为数据原型作为例子来展现如何引用或者定义自己的数据模型。再其次是crypto库，在BitXHub的规划中会逐渐丰富能够提供给用户的方便使用的密码库，这样能够让用户更方便的调用加密函数来编写自己的验证逻辑。

除了上述的目录和文件，整个合约模板最重要的就是app这个目录，用户主要需要编写的文件也是在这个目录中，其中app这个目录下的contract.rs的文件就是用户编写验证规则逻辑的文件。

```rust

pub fn verify(proof: &[u8], validator: &[u8], payload: &[u8]) -> bool {
  return true
}
```

在模板中我们可以看到这个函数，这个就是用户需要进行逻辑编写的地方，其中该函数的输入由三个参数组成：proof, validator和payload，都是字节数组，用户直接可以用这三个参数完成自己的验证逻辑。如何用proof, validator和payload进行验证就看应用链自己是如何来规定自己的验证逻辑的。

以最简单的椭圆曲线验证为例，在模板库里面提供了一个ecdsa的函数：

```rust
fn ecdsa_verify(sig_ptr: i64, digest_ptr: i64, pubkey_ptr: i64, opt: i32) -> i32;
```

如果我们的验证逻辑是proof就是ecdsa中的签名，payload就是ecdsa中的digest，然后validator就是公钥，那么验证逻辑就可以简单写成：

```rust

pub fn verify(proof: &[u8], validator: &[u8], payload: &[u8]) -> bool {
  return ecdsa::verify(
     &proof,
     &payload,
     &validator,
     ecdsa::EcdsaAlgorithmn::P256,
  );
}
```

更加复杂的逻辑就留给用户自己来实现。

## 3. 编译代码成为WASM字节码

下面来介绍如何用rust将代码编译成简洁的wasm字节码。

首先我们需要添加rust的nightly版本：

```shell
rustup toolchain install nightly
```

更新rust：

```shell
rustup update
```

添加需要的目标工具链：

```shell
rustup target add wasm32-unknown-unknown --toolchain nightly
```

接下来就可以编译我们的项目了：

```text
cargo  +nightly build --target wasm32-unknown-unknown --release
```

我们编译出来的wasm字节码将会在target/wasm32-unknown-unknown/release下面，注意这个时候我们编译出来的wasm字节码的大小是非常大的，我们需要将其精简一下，首先下载工具wasm-gc

```text
cargo install --git https://github.com/alexcrichton/wasm-gc
```

然后使用wasm-gc进行精简：

```text
wasm-gc xxx.wasm small-xxx.wasm
```

获取的small-xxx.wasm就是可以部署到中继链的验证规则。