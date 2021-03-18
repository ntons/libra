TARGETS = librad
IMAGES = librad-alpine librad-ubuntu
ACTIONS = publish

.PHONY: all $(TARGETS) $(IMAGES) $(ACTIONS)

all: targets images

targets: $(TARGETS)

images: $(IMAGES)

VERSION    := $(shell cat VERSION)
BUILT      := $(shell date -u)
GIT_COMMIT := $(shell git rev-list -1 HEAD)
GO_VERSION := $(shell go version | cut -d' ' -f3)
OS_ARCH    := $(shell go version | cut -d' ' -f4)

librad:
	CGO_ENABLED=0 go build -ldflags "-X 'main.Version=$(VERSION)' -X 'main.Built=$(BUILT)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.GoVersion=$(GO_VERSION)' -X 'main.OSArch=$(OS_ARCH)'" -o $@ github.com/ntons/libra/librad

librad-alpine:
	docker build -t ntons/librad-alpine:latest --build-arg BASE=alpine:3.12 .
	docker tag ntons/librad-alpine:latest ntons/librad-alpine:$(VERSION)
	docker tag ntons/librad-alpine:latest ntons/librad:latest
	docker tag ntons/librad-alpine:latest ntons/librad:$(VERSION)

librad-ubuntu:
	docker build -t ntons/librad-ubuntu:latest --build-arg BASE=ubuntu:20.04 .
	docker tag ntons/librad-ubuntu:latest ntons/librad-ubuntu:$(VERSION)

publish:
	docker push ntons/librad:latest
	docker push ntons/librad-alpine:latest
	docker push ntons/librad-ubuntu:latest
	docker push ntons/librad:$(VERSION)
	docker push ntons/librad-alpine:$(VERSION)
	docker push ntons/librad-ubuntu:$(VERSION)

