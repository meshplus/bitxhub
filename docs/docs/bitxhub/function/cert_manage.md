# 证书管理

中继链节点具有完备的证书管理体系用于保证信息和数据的完整性和安全性。

### 证书配置

##### 证书结构

中继链节点主要分三个层级的证书结构，分别是ca、agency和node，具体结构如下

~~~
.
├── agency.cert
├── ca.cert
├── node.cert
├── node.priv

0 directories, 7 files
~~~

##### 证书生成方式

**ca证书生成**

~~~shell
# 生成ca相关的证书和私钥
bitxhub cert ca
~~~

**agency证书生成**

~~~shell
# 根据ca颁布agency的证书
bitxhub cert priv gen --name=agency --target=./
bitxhub cert csr --key=agency.priv --org=agency --target=./
bitxhub cert issue --csr=agency.csr --is_ca=true --key=ca.priv --cert=ca.cert --target=./
~~~

**node证书生成**

~~~shell
bitxhub cert priv gen --name=node --target=./
bitxhub cert csr --key=node.priv --org=node --target=./
bitxhub cert issue --csr=node.csr --is_ca=false --key=agency.priv --cert=agency.cert --target=./
~~~

### 节点私钥配置

##### 节点私钥生成

~~~shell
bitxhub key gen --name=key --target=./
~~~

##### 节点私钥格式转换

~~~shell
bitxhub key gen --passwd=bitxhub --target=./
~~~

**说明：节点私钥会进行加密，如果密码不正确，中继链无法启动，目前默认的密码是bitxhub，如果采用其他密码，在启动中继链节点的过程中需要指定密码。以密码为hub为例，bitxhub启动命令为：bitxhub --repo ~/node1 start --passwd hub**

