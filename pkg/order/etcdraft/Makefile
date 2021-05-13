proto:
	cd proto && protoc -I=. \
	-I${GOPATH}/src \
	-I${GOPATH}/src/github.com/gogo/protobuf/protobuf \
	-I${GOPATH}/src/github.com/meshplus/bitxhub-model/pb \
	--gogofaster_out=:. \
	message.proto

.PHONY: proto
