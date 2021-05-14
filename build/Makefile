TARGETS = librad logbus
IMAGES = librad-buster
ACTIONS = publish

.PHONY: all $(TARGETS) $(IMAGES) $(ACTIONS)

all: $(TARGETS) $(IMAGES)

VERSION    := $(shell cat VERSION)
BUILT      := $(shell date -u)
GIT_COMMIT := $(shell git rev-list -1 HEAD)
GO_VERSION := $(shell go version | cut -d' ' -f3)
OS_ARCH    := $(shell go version | cut -d' ' -f4)

librad:
	go build -ldflags "-X 'main.Version=$(VERSION)' -X 'main.Built=$(BUILT)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.GoVersion=$(GO_VERSION)' -X 'main.OSArch=$(OS_ARCH)'" -o bin/$@ github.com/ntons/libra/librad

logbus:
	go build -ldflags "-X 'main.Version=$(VERSION)' -X 'main.Built=$(BUILT)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.GoVersion=$(GO_VERSION)' -X 'main.OSArch=$(OS_ARCH)'" -o bin/$@ github.com/ntons/libra/logbus

librad-buster:
	docker build -t ntons/librad:latest .
	docker tag ntons/librad:latest ntons/librad:$(VERSION)

publish:
	docker push ntons/librad:latest
	docker push ntons/librad:$(VERSION)

