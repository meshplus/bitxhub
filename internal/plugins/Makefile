GO  = GO111MODULE=on go

help: Makefile
	@echo "Choose a command run:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

## make raft: build plugin (make plugin type= <raft>)
raft:
	@mkdir -p build
	$(GO) build --buildmode=plugin -o build/raft.so order/etcdraft/*.go

raftent:
	@mkdir -p build
	$(GO) build --buildmode=plugin -modfile ../../goent.mod  -o build/raft.so order/etcdraft/*.go

## make solo: build plugin (make plugin type= <solo>)
solo:
	@mkdir -p build
	$(GO) build --buildmode=plugin -o build/solo.so order/solo/*.go

soloent:
	@mkdir -p build
	$(GO) build --buildmode=plugin -modfile ../../goent.mod -o build/solo.so order/solo/*.go

## make plugins: build plugins (make plugin type=<raft> and <solo>)
plugins: raft solo
