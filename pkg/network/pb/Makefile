GO  = GO111MODULE=on go

help: Makefile
	@echo "Choose a command run:"
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'

## make pb: build network message protobuf
proto:
	protoc -I=. \
	-I${GOPATH}/src \
	-I${GOPATH}/src/github.com/gogo/protobuf/protobuf \
	--gogofast_out=:. network.proto

.PHONY: proto
