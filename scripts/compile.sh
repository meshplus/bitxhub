#!/usr/bin/env sh

go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
go get github.com/gobuffalo/packr/packr@v1.30.1
cd /code/bitxhub || exit
make install
mkdir -p /code/bitxhub/bin
cp /go/bin/bitxhub /code/bitxhub/bin/bitxhub_linux-amd64