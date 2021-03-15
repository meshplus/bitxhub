# GoDuck 运维小工具
Goduck is a command-line management tool that can help to run BitXHub.
# 1 Installation

## 1.1 Get the Source Code
```
git clone git@github.com:meshplus/goduck
```

## 1.2 Compile and Install
```
cd goduck
sudo make install
```

## 1.3 Initialization
```
goduck init
```
Be sure to initialize before use.

# 2 Usage
## 2.1 Format of Command
```
goduck [global options] command [command options] [arguments...]
```
**command**
- `deploy`          Deploy BitXHub and pier
- `version`          Components version
- `init`          Init config home for GoDuck
- `status`          List the status of instantiated components
- `key`          Create and show key information
- `bitxhub`          Start or stop BitXHub nodes
- `pier`          Operation about pier
- `playground`          Set up and experience interchain system smoothly
- `info`          Show basic info about interchain system
- `prometheus`          Start or stop prometheus
- `help, h`          Shows a list of commands or help for one command

Among these commands, the relatively important ones are `init` (be sure to initialize before use), `status` (view the status of current running components), `bitxhub`, and `pier`.

**global options**
- `--repo value`          GoDuck storage repo path
- `--help, -h`

## 2.2 Operation about BitXHub
```
goduck bitxhub command [command options] [arguments...]
```
### 2.2.1 Start BitXHub Nodes
```
goduck bitxhub start
```
The command will initialize and start the bitxhub node. If there are bitxhub nodes that have been started, the execution fails. After successful execution, the prompt is as follows:
```
1 BitXHub nodes at /Users/fangbaozhu/.goduck/bitxhub are initialized successfully
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh up v1.4.0 solo binary 4
Start bitxhub solo by binary
===> Start bitxhub solo successful
```

This is the default startup mode. You can also carry parameters for custom startup. The parameter settings are as follows:
- `--type value`          configuration type, one of binary or docker (default: "binary")
- `--mode value`          configuration mode, one of solo or cluster (default: "solo")
- `--num value`          node number, only useful in cluster mode, ignored in solo mode (default: 4)
- `--tls`          whether to enable TLS, only useful for v1.4.0+ (default: false)
- `--version value`          BitXHub version (default: "v1.4.0")
- `--help, -h`          show help (default: false)

### 2.2.2 Generate Configuration for BitXHub Nodes
```
goduck bitxhub config
```
The command initializes 4 bitxhub nodes by default mode. After successful execution, the current folder will generate related certificates, private key files and folders of four nodes. The success prompt is as follows:
```
initializing 4 BitXHub nodes at .
4 BitXHub nodes at . are initialized successfully
```

You can see the following in the current directory：
```
.
├── agency.cert
├── agency.priv
├── ca.cert
├── ca.priv
├── key.priv
├── node1/
├── node2/
├── node3/
└── node4/
```

You can customize the configuration by carrying parameters.
- `--num value`          node number, only useful in cluster mode, ignored in solo mode (default: 4)
- `--type value`         configuration type, one of binary or docker (default: "binary")
- `--mode value`          configuration mode, one of solo or cluster (default: "cluster")
- `--ips value`          nodes' IP, use 127.0.0.1 for all nodes by default, e.g. --ips "127.0.0.1" --ips "127.0.0.2" --ips "127.0.0.3" --ips "127.0.0.4"
- `--target value`          where to put the generated configuration files (default: ".")
- `--tls`          whether to enable TLS, only useful for v1.4.0+ (default: false)
- `--version value`          BitXHub version (default: "v1.4.0")
- `--help, -h`          show help (default: false)

### 2.2.3 Stop BitXHub Nodes
```
goduck bitxhub stop
```

The command will shut down all started bitxhub nodes. After successful execution, a prompt will show the ID of the node to be closed as follow:

```
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh down
===> Stop bitxhub
node pid:65246 exit
```

### 2.2.4 Clean BitXHub Nodes
```
goduck bitxhub clean
```

The command will clean the configuration files of the BithxHub node. If there are any unclosed BitxHub nodes, it will close the node before deleting the configuration.

When the BitxHub SOLO node is started and closed normally, the print result of the current command is as follows:
```
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh clean
===> Stop bitxhub
===> Clean bitxhub
remove bitxhub configure nodeSolo
```

When the BitxHub SOLO node is started normally but not closed, the print result of the current command is as follows:
```
exec:  /bin/bash /Users/fangbaozhu/.goduck/playground.sh clean
===> Stop bitxhub
node pid:65686 exit
===> Clean bitxhub
remove bitxhub configure nodeSolo
```

## 2.3 Operation about Pier
```
GoDuck pier command [command options] [arguments...]
```

### 2.3.1 Start Pier
```
goduck pier start
```
The command will start the pier. You can define the type of application chain to which the pier connects and the startup type of pier by carrying parameters. The parameter settings are as follows:

- `--chain value`          specify appchain type, ethereum(default) or fabric (default: "ethereum")
- `--cryptoPath value`          path of crypto-config, only useful for fabric chain, e.g $HOME/crypto-config
- `--pier-type value`          specify pier up type, docker(default) or binary (default: "docker")
- `--version value`          pier version (default: "v1.4.0")
- `--tls value`          whether to enable TLS, true or false, only useful for v1.4.0+ (default: "false")
- `--http-port value`          peer's http port, only useful for v1.4.0+ (default: "44544")
- `--pprof-port value`          pier pprof port, only useful for binary (default: "44550")
- `--api-port value`          pier api port, only useful for binary (default: "8080")
- `--overwrite value`          whether to overwrite the configuration if the pier configuration file exists locally (default: "true")
- `--appchainIP value`          the application chain IP that pier connects to (default: "127.0.0.1")
- `--help, -h`          show help (default: false)

Before using this command, you need to start a same version of BitxHub and an application chain that requires the pier connection. If the application chain or the bitxhub does not exist, the pier will fail to start and the print prompt is as follows:
```
exec:  /bin/bash run_pier.sh up -m ethereum -t docker -r .pier_ethereum -v v1.4.0 -c  -p 44550 -a 8080
===> Start pier of ethereum-v1.4.0 in docker...
===> Start a new pier-ethereum container
===> Wait for ethereum-node container to start for seconds...
136d323b1418a026101515313dbbdafee240ac0f0c0d63b4f202304019e13e24
===> Start pier fail
```

If the same version of BitxHub and the chain of applications to be connected has been started and the PIER has started successfully, the print prompt is as follows:
```
exec:  /bin/bash run_pier.sh up -m ethereum -t docker -r .pier_ethereum -v v1.0.0-rc1 -c  -p 44550 -a 8080
===> Start pier of ethereum-v1.0.0-rc1 in docker...
===> Start a new pier-ethereum container
===> Wait for ethereum-node container to start for seconds...
351bd8e8eb8d5a1803690ac0cd9b77c274b775507f30cb6271164fb843442bfd
===> Start pier successfully
```

### 2.3.2 Stop Pier
```
goduck pier stop
```
The command will close the pier. The type of application chain (ethereum or fabric) to which the pier is specifically closed can be determined by the parameter `-- chain value`.

### 2.3.3 Clean Pier
```
goduck pier clean
```

The command will clean the configuration files of the pier. If there are any unclosed pier, it will close the pier before deleting the configuration.


### 2.3.4 Generate configuration for Pier
```
goduck pier config
```

- `--mode value`          configuration mode, one of direct or relay (default: "direct")
- `--type value`          configuration type, one of binary or docker (default: "binary")
- `--bitxhub value`       BitXHub's address, only useful when in relay mode
- `--validators value`    BitXHub's validators, only useful in relay mode, e.g. --validators "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013" --validators "0x79a1215469FaB6f9c63c1816b45183AD3624bE34" --validators "0x97c8B516D19edBf575D72a172Af7F418BE498C37" --validators "0x97c8B516D19edBf575D72a172Af7F418BE498C37"
- `--port value`          pier's port, only useful when in direct mode (default: 5001)
- `--peers value`          peers' address, only useful in direct mode, e.g. --peers "/ip4/127.0.0.1/tcp/4001/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5ngmL"
- `--connectors value`     address of peers which need to connect, only useful in union mode for v1.4.0+, e.g. --connectors "/ip4/127.0.0.1/tcp/4001/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5ngmL" --connectors "/ip4/127.0.0.1/tcp/4002/p2p/Qma1oh5JtrV24gfP9bFrVv4miGKz7AABpfJhZ4F2Z5abcD"
- `--providers value`      the minimum number of cross-chain gateways that need to be found in a large-scale network, only useful in union mode for v1.4.0+ (default: "1")
- `--appchain-type value`  appchain type, one of ethereum or fabric (default: "ethereum")
- `--appchain-IP value`    appchain IP address (default: "127.0.0.1")
- `--target value`         where to put the generated configuration files (default: ".")
- `--tls value`            whether to enable TLS, only useful for v1.4.0+ (default: "false")
- `--http-port value`      peer's http port, only useful for v1.4.0+ (default: "44544")
- `--pprof-port value`     peer's pprof port (default: "44550")
- `--api-port value`       peer's api port (default: "8080")
- `--cryptoPath value`     path of crypto-config, only useful for fabric chain, e.g $HOME/crypto-config (default: "$HOME/.goduck/crypto-config")
- `--version value`        pier version (default: "v1.4.0")
- `--help, -h` 