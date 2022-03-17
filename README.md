<p align="center">
  <img src="https://raw.githubusercontent.com/meshplus/bitxhub/master/docs/logo.png" />
</p>

![build](https://github.com/meshplus/bitxhub/workflows/build/badge.svg)
[![codecov](https://codecov.io/gh/meshplus/bitxhub/branch/master/graph/badge.svg)](https://codecov.io/gh/meshplus/bitxhub)
[![Go Report Card](https://goreportcard.com/badge/github.com/meshplus/bitxhub)](https://goreportcard.com/report/github.com/meshplus/bitxhub)

BitXHub is committed to building a scalable, robust, and pluggable inter-blockchain
reference implementation, that can provide reliable technical support for the formation
of a blockchain internet and intercommunication of value islands.

**For more details please visit our [documentation](https://docs.bitxhub.cn/) and [whitepaper](https://upload.hyperchain.cn/BitXHub%20Whitepaper.pdf) | [白皮书](https://upload.hyperchain.cn/BitXHub%E7%99%BD%E7%9A%AE%E4%B9%A6.pdf).**

## Start

BitXHub start script relies on [golang](https://golang.org/) and [tmux](https://github.com/tmux/tmux/wiki). Please
install the software before start.

Use commands below to clone the project:

```shell
git clone git@github.com:meshplus/bitxhub.git
```

BitXHub also relies on some small tools, use commands below to install:

```shell
cd bitxhub
bash scripts/prepare.sh 
```

Finally, run the following commands to start a four nodes relay-chain.

```shell
make cluster
```

**Noting:** `make cluster` will use `tmux` to split the screen. Thus, during commands processing, better not switch the terminal.

## Playground
Simply go to [BitXHub Document](https://meshplus.github.io/bitxhub/bitxhub/quick_start/) and follow the tutorials.


## Contributing

See [CONTRIBUTING.md](https://github.com/meshplus/bitxhub/blob/master/CONTRIBUTING.md).

## Contact

Email: bitxhub@hyperchain.cn

Wechat: If you‘re interested in BitXHub, please add the assistant to join our community group.

<img src="https://raw.githubusercontent.com/meshplus/bitxhub/master/docs/wechat.png" width="200" /><img src="https://raw.githubusercontent.com/meshplus/bitxhub/master/docs/official.png" width="206" />

## License

The BitXHub library (i.e. all code outside of the cmd and internal directory) is licensed under the GNU Lesser General Public License v3.0, also included in our repository in the COPYING.LESSER file.

The BitXHub binaries (i.e. all code inside of the cmd and internal directory) is licensed under the GNU General Public License v3.0, also included in our repository in the COPYING file.
