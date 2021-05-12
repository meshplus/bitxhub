FROM golang:1.14.2 as builder

RUN mkdir -p /go/src/github.com/meshplus/bitxhub
WORKDIR /go/src/github.com/meshplus/bitxhub

# Cache dependencies
COPY go.mod .
COPY go.sum .

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download -x

# Build real binaries
COPY . .

RUN go get -u github.com/gobuffalo/packr/packr

# Build bitxhub node
RUN make install

# Build raft plugin
RUN cd internal/plugins && make raft


# Final image
FROM frolvlad/alpine-glibc
RUN mkdir -p /root/.bitxhub/plugins
WORKDIR /

# Copy over binaries from the builder
COPY --from=builder /go/bin/bitxhub /usr/local/bin
COPY --from=builder /go/bin/packr /usr/local/bin
COPY --from=builder /go/src/github.com/meshplus/bitxhub/internal/plugins/build/raft.so /root/.bitxhub/plugins/

COPY ./build/libwasmer.so /lib

ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/lib


EXPOSE 8881 60011 9091 53121 40011

ENTRYPOINT ["bitxhub", "start"]


