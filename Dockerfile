FROM golang:1.15.15 as builder

WORKDIR /go/src/github.com/meshplus/bitxhub
ARG http_proxy=""
ARG https_proxy=""
ENV PATH=$PATH:/go/bin
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/lib
COPY . /go/src/github.com/meshplus/bitxhub/

RUN go env -w GOPROXY=https://goproxy.cn,direct \
    && go get -u github.com/gobuffalo/packr/packr \
    && make install \
    && cp ./build/wasm/lib/linux-amd64/libwasmer.so /lib \
    && bitxhub init \
    && cp /go/src/github.com/meshplus/bitxhub/scripts/certs/node1/key.json /root/.bitxhub/ \
    && cp -r /go/src/github.com/meshplus/bitxhub/scripts/certs/node1/certs/* /root/.bitxhub/certs \
    && sed -i 's/solo = false/solo = true/g' /root/.bitxhub/bitxhub.toml \
    && sed -i 's/raft/solo/g' /root/.bitxhub/bitxhub.toml \
    # Clean Cache 
    && cd \ 
    && rm -rf /go/src/github.com/meshplus/bitxhub \
    && go clean -modcache

FROM frolvlad/alpine-glibc:glibc-2.32

COPY --from=0 /go/bin/bitxhub /usr/local/bin/bitxhub
COPY --from=0 /root/.bitxhub /root/.bitxhub
COPY --from=0 /lib/libwasmer.so /lib/libwasmer.so
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/lib

EXPOSE 8881 60011 9091 53121 40001
ENTRYPOINT ["bitxhub", "start"]


