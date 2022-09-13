#!/usr/bin/env bash
function go_install() {
  version=$(go env GOVERSION)
  if [[ ! "$version" < "go1.16" ]];then
      go install "$@"
  else
      go get "$@"
  fi
}
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
go_install github.com/gobuffalo/packr/packr@v1.30.1
cd /code/bitxhub || exit
make install
mkdir -p /code/bitxhub/bin
cp /go/bin/bitxhub /code/bitxhub/bin/bitxhub_linux-amd64