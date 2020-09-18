proto:
	cd proto && protoc -I=. \
	-I${GOPATH}/src \
	-I${GOPATH}/src/github.com/gogo/protobuf/protobuf \
	--gogofast_out=:. \
	message.proto

.PHONY: proto
