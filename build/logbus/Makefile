TARGETS = logbus

.PHONY: all $(TARGETS)

all: targets

targets: $(TARGETS)

VERSION    := $(shell cat VERSION)
BUILT      := $(shell date -u)
GIT_COMMIT := $(shell git rev-list -1 HEAD)
GO_VERSION := $(shell go version | cut -d' ' -f3)
OS_ARCH    := $(shell go version | cut -d' ' -f4)

logbus:
	CGO_ENABLED=0 go build -ldflags "-X 'main.Version=$(VERSION)' -X 'main.Built=$(BUILT)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.GoVersion=$(GO_VERSION)' -X 'main.OSArch=$(OS_ARCH)'" -o $@ github.com/ntons/libra/logbus


