![build](https://github.com/axiomesh/axiom/workflows/build/badge.svg)
[![codecov](https://codecov.io/gh/axiomesh/axiom/branch/main/graph/badge.svg)](https://codecov.io/gh/axiomesh/axiom)
[![Go Report Card](https://goreportcard.com/badge/github.com/axiomesh/axiom)](https://goreportcard.com/report/github.com/axiomesh/axiom)

Axiom is high performance open permission blockchain.

## Start

Axiom start script relies on [golang](https://golang.org/) and [tmux](https://github.com/tmux/tmux/wiki). Please
install the software before start.

Use commands below to clone the project:

```shell
git clone git@github.com:axiomesh/axiom.git
```

Axiom also relies on some small tools, use commands below to install:

```shell
make prepare
```

Finally, run the following commands to start a four nodes relay-chain.

```shell
make cluster
```

**Noting:** `make cluster` will use `tmux` to split the screen. Thus, during commands processing, better not switch the terminal.

## Contributing

See [CONTRIBUTING.md](https://github.com/axiomesh/axiom/blob/main/CONTRIBUTING.md).

## License

The Axiom library (i.e. all code outside of the cmd and internal directory) is licensed under the GNU Lesser General Public License v3.0, also included in our repository in the COPYING.LESSER file.

The Axiom binaries (i.e. all code inside of the cmd and internal directory) is licensed under the GNU General Public License v3.0, also included in our repository in the COPYING file.
