################################################################################
# Stage 1: Build binaries
################################################################################
FROM golang:1.16.4-buster AS builder

ENV GOPATH=/go
ENV GOPROXY=https://goproxy.io,direct

WORKDIR /go/src/github.com/ntons/libra

COPY . .

RUN go build -ldflags "-X 'main.Version=`cat VERSION`' -X 'main.Built=`date -u`' -X 'main.GitCommit=`git rev-list -1 HEAD`' -X 'main.GoVersion=`go version | cut -d' ' -f3`' -X 'main.OSArch=`go version | cut -d' ' -f4`'" -o librad github.com/ntons/libra/librad

################################################################################
# Stage 2: Build images
################################################################################
FROM debian:buster

COPY --from=builder /go/src/github.com/ntons/libra/librad/librad      /bin/
COPY --from=builder /go/src/github.com/ntons/libra/librad/librad.yaml /etc/

ENTRYPOINT ["/bin/librad","-c","/etc/librad.yaml"]

