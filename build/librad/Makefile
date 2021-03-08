TARGETS = librad
IMAGES = librad-focal librad-alpine librad-debug
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
	go build -ldflags "-X 'main.Version=$(VERSION)' -X 'main.Built=$(BUILT)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.GoVersion=$(GO_VERSION)' -X 'main.OSArch=$(OS_ARCH)'" -o $@ github.com/ntons/libra/librad

librad-focal: 
	docker build -f dockerfiles/focal -t ntons/librad-focal:latest .
	docker tag ntons/librad-focal:latest ntons/librad-focal:$(VERSION)
	docker tag ntons/librad-focal:latest ntons/librad:latest
	docker tag ntons/librad:latest ntons/librad:$(VERSION)

librad-alpine:
	docker build -f dockerfiles/alpine -t ntons/librad-alpine:latest .
	docker tag ntons/librad-alpine:latest ntons/librad-alpine:$(VERSION)

librad-debug:
	docker build -f dockerfiles/debug -t ntons/librad-debug:latest .
	docker tag ntons/librad-debug:latest ntons/librad-debug:$(VERSION)

publish:
	docker push ntons/librad:latest
	docker push ntons/librad-focal:latest
	docker push ntons/librad-alpine:latest
	docker push ntons/librad:$(VERSION)
	docker push ntons/librad-focal:$(VERSION)
	docker push ntons/librad-alpine:$(VERSION)

