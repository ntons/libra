################################################################################
# Stage 1: Build binaries
################################################################################
FROM golang:1.18-buster AS builder

ENV GOPATH=/go
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /go/src/github.com/ntons/libra

COPY go.mod go.sum ./

RUN go mod download -x

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X 'main.Version=`cat VERSION`' -X 'main.Built=`date -u`' -X 'main.GitCommit=`git rev-list -1 HEAD`' -X 'main.GoVersion=`go version | cut -d' ' -f3`' -X 'main.OSArch=`go version | cut -d' ' -f4`'" -o librad/librad github.com/ntons/libra/librad

################################################################################
# Stage 2: Build images
################################################################################
#FROM debian:buster
FROM alpine:3.14 AS final

COPY --from=builder /go/src/github.com/ntons/libra/librad/librad      /bin/
COPY --from=builder /go/src/github.com/ntons/libra/librad/librad.yaml /etc/

ENTRYPOINT ["/bin/librad","-c","/etc/librad.yaml"]

