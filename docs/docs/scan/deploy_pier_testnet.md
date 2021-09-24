# 跨链网关部署

跨链网关Pier能够支持业务所在区块链（以下简称 应用链）便捷、快速的接入到跨链平台BitXHub中，从而实现和其他业务区块链的跨链操作。目前中继链和跨链网关已经支持 **Fabric**、**Ethereum**、**BCOS**、**CITA** 和 **Hyperchain** 五种应用链接入并完成跨链交易，如果您有兴趣，也可以参与开发适配另外种类应用链的插件和合约。

跨链网关的部署需要提前确定应用链类型（对应不同的插件和配置），也需要提前在对应的应用链上部署跨链合约，我们将跨链网关的部署依次分为如下五个步骤：

1. **部署跨链合约**

2. **获取部署包和修改Pier配置**




## 部署跨链合约

这一章是要在应用链上部署跨链合约，**注意：在此操作之前，您需要确认已经部署有可接入的应用链**。

我们提供了针对不同应用链的跨链合约，broker合约是管理合约，transfer合约是业务交易合约，需要说明的是 transfer合约需要经过broker合约审核通过后才能发起或接受跨链交易，具体方法是：调用broker合约的audit方法，其参数依次是业务合约地址和合约状态（数字1表示审核通过，数字2表示审核失败）。

下面以Ethereum和Fabric为例进行介绍，其它类型的应用链部署跨链合约的步骤基本上是一致的。

=== "Ethereum"
    在Ethereum上部署合约的工具有很多，您可以使[Remix](https://remix.ethereum.org/)进行合约的编译和部署，这里关键的是跨链合约的获取。可以下载pier-client-ethereum项目：	`git clone https://github.com/meshplus/pier-client-ethereum.git && git checkout v1.6.2`
    
    **说明：**
    1. 合约文件就在项目的example目录下，broker.sol是跨链管理合约，transfer.sol是示例业务合约；
    2. 首先部署broker合约，然后将返回的合约地址填入transfer合约中的`BrokerAddr`字段，这样业务合约才能正确跨链调用。

=== "Fabric"
    在Fabric上部署跨链合约工具一般是fabric-cli（可以参考[官方项目的使用说明](https://github.com/hyperledger/fabric-cli)）， 在Fabric上部署跨链合约的过程和部署其它合约没有区别，只是合约名称和代码文件需要替换，以下操作的命令可供参考，默认应用链是使用的fabric-sample项目的v1.4.3版本部署。
    
    Step1: 安装部署合约的工具fabric-cli
    ```
    go get github.com/securekey/fabric-examples/fabric-cli/cmd/fabric-cli
    ```
    
    Step2: 获取需要部署的合约文件并解压
    ```
    git clone https://github.com/meshplus/pier-client-ethereum.git && git checkout v1.6.2
    # 需要部署的合约文件就在example目录下
    #解压即可
    unzip -q contracts.zip
    ```
    
    Step3: 部署broker、transfer合约
    ```
    #安装和示例化broker合约
    fabric-cli chaincode install --gopath ./contracts --ccp broker --ccid broker --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
    fabric-cli chaincode instantiate --ccp broker --ccid broker --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
    
    #安装和示例化transfer合约
    fabric-cli chaincode install --gopath ./contracts --ccp transfer --ccid transfer --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
    fabric-cli chaincode instantiate --ccp transfer --ccid transfer --config "${CONFIG_YAML}" --orgid org2 --user Admin --cid mychannel
    
    #业务合约需要broker管理合约审计后，才能进行跨链交易
    fabric-cli chaincode invoke --cid mychannel --ccid=broker \
    --args='{"Func":"audit", "Args":["mychannel", "transfer", "1"]}' \
    --user Admin --orgid org2 --payload --config "${CONFIG_YAML}"
    ```

## 获取部署包和修改Pier配置
这一章是要获取部署包和修改Pier的配置，为启动pier节点作准备，主要分为pier本身和应用链插件的配置修改。可以通过源码编译和二进制下载的方式获取部署包。
### 获取部署包
=== "Ethereum"
    **源码下载编译**

    ```
    # 编译跨链网关本身
    cd $HOME
    git clone https://github.com/meshplus/pier.git
    cd pier && git checkout v1.11.1
    make prepare && make build

    # 编译Ethereum 插件
    cd $HOME
    git clone https://github.com/meshplus/pier-client-ethereum.git
    cd pier-client-ethereum && git checkout v1.11.1
    make eth
    
    # 说明：1.ethereum插件编译之后会在插件项目的build目录生成eth-client文件；
    2.pier编译之后会在跨链网关项目bin目录生成同名的二进制文件。
    ```
    
    **二进制下载**
    
    除了源码编译外，我们也提供了直接下载Pier及其插件二进制的方式，下载地址链接如下：[Pier二进制包下载](https://github.com/meshplus/pier/releases/tag/v1.11.1) 和 [ethereum插件二进制包下载](https://github.com/meshplus/pier-client-ethereum/releases/tag/v1.6.2)链接中已经包含了所需的二进制程序和依赖库，您只需跟据操作系统的实际情况进行选择和下载即可。

=== "Fabric"
    **源码下载编译**
    
    ```
    # 编译跨链网关本身
    cd $HOME
    git clone https://github.com/meshplus/pier.git
    cd pier && git checkout v1.11.1
    make prepare && make build
    
    # 编译Fabric插件
    cd $HOME
    git clone https://github.com/meshplus/pier-client-fabric.git
    cd pier-client-fabric && git checkout v1.11.1
    make fabric1.4
    
    # 说明：1.fabric插件编译之后会在插件项目的build目录生成fabric-client-1.4文件；
    2.pier编译之后会在跨链网关项目bin目录生成同名的二进制文件。
    ```
    
    **二进制下载**
    
    除了源码编译外，我们也提供了直接下载Pier及其插件二进制的方式，下载地址链接如下：[Pier二进制包下载](https://github.com/meshplus/pier/releases/tag/v1.11.1) 和 [fabric插件二进制包下载](https://github.com/meshplus/pier-client-fabric/releases/tag/v1.6.2)链接中已经包含了所需的二进制程序和依赖库，您只需跟据操作系统的实际情况进行选择和下载即可。

经过以上的步骤，相信您已经编译出了部署Pier和fabric/ethereum应用链插件的二进制文件，Pier节点运行还需要外部依赖库，均在项目build目录下（Macos使用libwasmer.dylib，Linux使用libwasmer.so）,建议将得到的二进制和适配的依赖库文件拷贝到同一目录，然后使用 `export LD_LIBRARY_PATH=$(pwd)`命令指定依赖文件的路径，方便之后的操作。

### 修改Pier自身的配置
在进行应用链注册、验证规则部署等步骤之前，需要初始化跨链网关的配置目录，以用户目录下的pier为例：
```
pier --repo=~/.pier init
```
该命令会生成跨链网关的一些基础配置文件模板，使用 tree 命令可查看目录信息：
```text
tree -L 1 ~/.pier
├── api
├── certs
├── key.json
├── node.priv
└── pier.toml    
1 directory, 4 files
```

pier.toml是描述链跨链网关启动的主要配置，一般需要修改的是端口信息、中继链的信息、应用链的信息。

- 修改端口信息

```none
[port]
# 如果不冲突的话，可以不用修改
http  = 44544
pprof = 44555
```

- 修改中继链信息（一般只修改addrs字段，指定bitxhub的节点地址）

```none
[mode]
type = "relay" # relay or direct
[mode.relay]
addrs = ["localhost:60011", "localhost:60012", "localhost:60013", "localhost:60014"]
quorum = 2
validators = [
    "0x000f1a7a08ccc48e5d30f80850cf1cf283aa3abd",
    "0xe93b92f1da08f925bdee44e91e7768380ae83307",
    "0xb18c8575e3284e79b92100025a31378feb8100d6",
    "0x856E2B9A5FA82FD1B031D1FF6863864DBAC7995D",
]
```

- 修改应用链信息（针对不同应用链类型进行配置）

=== "Ethereum"
    ```
    [appchain]
    # ethereum插件文件的名称
    plugin = "eth-client"
    # ethereum配置文件夹在跨链网关配置文件夹下的相对路径
    config = "ether"
    ```
=== "Fabric"
    ```
    [appchain]
    # fabric插件文件的名称
    plugin = "fabric-client-1.4"
    # ethereum配置文件夹在跨链网关配置文件夹下的相对路径
    config = "fabric"
    ```

### 修改应用链插件的配置

应用链插件的配置目录即是pier.toml中的config字段，它的模板在`pier-client-ethereum`或`pier-client-ethereum`项目（之前拉取跨链合约时已经clone），直接在GitHub上下载代码即可

=== "Ethereum"
    ```shell
    # 将ethereum插件拷贝到plugins目录下
    cp ether-client ~/.pier/plugins/
    # 切换到pier-client-ethereum项目路径下
    cd pier-client-ethereum
    cp ./config $HOME/.pier/ether
    ```
    其中重要配置如下：
    ```shell
    ├── account.key
    ├── broker.abi
    ├── ether.validators
    ├── ethereum.toml
    ├── password
    └── validating.wasm
    ```
    **说明**：account.key和password建议换成应用链上的真实账户，且须保证有一定金额（ethereum上调用合约需要gas费），broker.abi可以使用示例，也可以使用您自己编译/部署broker合约时返回的abi，ether.validators和validating.wasm一般不需要修改。ethereum.toml是需要重点修改的，需要根据应用链实际情况填写ethereum网络地址、broker合约地址及abi，账户的key等，示例如下：

    ```
    [ether]
    addr = "wss://kovan.infura.io/ws/v3/cc512c8c74c94938aef1c833e1b50b9a"
    name = "ether-kovan"
    ## 此处合约地址需要替换成变量代表的实际字符串
    contract_address = "$brokerAddr-kovan"
    abi_path = "broker.abi"
    key_path = "account.key"
    password = "password"
    ```

=== "Fabric"
    ```shell
    # 将fabric插件拷贝到plugins目录下
    cp fabric-client-1.4 ~/.pier/plugins/
    # 切换到pier-client-fabric项目路径下
    cd pier-client-fabric
    cp ./config $HOME/.pier/fabric
    ```
    其中重要配置如下：
    ```shell
    ├── crypto-config/
    ├── config.yaml
    ├── fabric.toml
    ├── fabric.validators
    └── validating.wasm
    ```
    接下来主要修改Fabric网络配置，验证证书，跨链合约设置：

    1. **fabric证书配置**
    
    启动Fabric网络时，会生成所有节点（包括Order、peer等）的证书信息，并保存在 crypto-config文件夹中，Fabric插件和Fabric交互时需要用到这些证书。
    ```
    # 复制您所部署的Fabric所产生的crypto-config文件夹
    cp -r /path/to/crypto-config $HOME/.pier/fabric/
    
    # 复制Fabric上验证人证书
    cp $HOME/.pier/fabric/crypto-config/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/msp/signcerts/peer1.org2.example.com-cert.pem $HOME/.pier/fabric/fabric.validators
    ```
    2. **修改Plugin配置文件config.yaml**
    
    `config.yaml`文件记录的Fabric网络配置（用您的网络拓扑配置文件替换这个样例文件），需要使用绝对路径，把所有的路径都修改为 `crypto-config`文件夹所在的绝对路径。
    ```
    {CONFIG_PATH}/fabric/crypto-config =>～/.pier/fabric/crypto-config
    # 替换为您部署的Fabric网络的拓扑设置文件即可，同时需要修改所有的Fabric 的IP地址，如：
    url: grpcs://localhost:7050 => url: grpcs://10.1.16.48:7050
    ```
    3. **修改Plugin配置文件 fabric.toml**
    
    示例是以官方部署脚本进行配置：
    ```
    addr = "localhost:7053" // 若Fabric部署在服务器上，该为服务器地址
    event_filter = "interchain-event-name"
    username = "Admin"
    ccid = "broker" // 若部署跨链broker合约名字不是broker需要修改
    channel_id = "mychannel"
    org = "org2"
    ```

**至此，对接ethereum和fabric应用链的pier及其插件的配置已经完成，接下来需要在测试网进行应用链注册和验证规则部署后，再启动pier节点。**