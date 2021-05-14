#!/bin/sh
# build development image

VERSION="dev"
BUILT=$(date -u)
GIT_COMMIT=$(git rev-list -1 HEAD)
GO_VERSION=$(go version | cut -d' ' -f3)
OS_ARCH=$(go version | cut -d' ' -f4)

echo "VERSION:    $VERSION"
echo "BUILT:      $BUILT"
echo "GIT_COMMIT: $GIT_COMMIT"
echo "GO_VERSION: $GO_VERSION"
echo "OS_ARCH:    $OS_ARCH"

set -x

go build -ldflags "-X 'main.Version=${VERSION}' -X 'main.Built=${BUILT}' -X 'main.GitCommit=${GIT_COMMIT}' -X 'main.GoVersion=${GO_VERSION}' -X 'main.OSArch=${OS_ARCH}'"

docker build -t ntons/librad:dev .

