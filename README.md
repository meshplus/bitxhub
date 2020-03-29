![BitXHub-Logo](https://raw.githubusercontent.com/meshplus/bitxhub/master/docs/logo.png)

BitXHub is committed to building a scalable, robust, and pluggable inter-blockchain
reference implementation, that can provide reliable technical support for the formation
of a blockchain internet and intercommunication of value islands.


## Dependencies

This project uses [golang](https://golang.org/), [tmux](https://github.com/tmux/tmux/wiki). Go check them out if you don't have them locally installed.

This project also depends on [packr](https://github.com/gobuffalo/packr/), [golangci-lint](github.com/golangci/golangci-lint), [gomock](github.com/golang/mock) and [mockgen](github.com/golang/mock), Installing them by follow command:

```bash
bash scripts/prepare.sh
```


## Documentation

If you want to run bitxhub in cluster mode, please use the follow command:

```bash
make cluster
```

## Deploy
BitXHub needs link dynamic library. User should download the [libwasmer_runtime_c_api.so](https://github.com/wasmerio/wasmer/releases/download/0.11.0/libwasmer_runtime_c_api.so) file and set the LD_LIBRARY_PATH to ensure BitXHub run correctly.

```bash
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:《your_lib_path》
```

[Whitepaper](https://upload.hyperchain.cn/bitxhub_whitepaper.pdf) | [白皮书](https://upload.hyperchain.cn/BitXHub%E7%99%BD%E7%9A%AE%E4%B9%A6.pdf)


## License