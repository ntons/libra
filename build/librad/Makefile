TARGETS = librad image
ACTIONS = push

.PHONY: all $(TARGETS) $(ACTIONS)

all: $(TARGETS)

VERSION := $(shell cat VERSION)

librad:
	$(eval BUILT      := $(shell date -u))
	$(eval GIT_COMMIT := $(shell git rev-list -1 HEAD))
	$(eval GO_VERSION := $(shell go version | cut -d' ' -f3))
	$(eval OS_ARCH    := $(shell go version | cut -d' ' -f4))
#$(eval VERSION := $(shell awk -F'.' '{print $$1"."$$2"."$$3+1}' VERSION))
	go build -ldflags "-X 'main.Version=$(VERSION)' -X 'main.Built=$(BUILT)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.GoVersion=$(GO_VERSION)' -X 'main.OSArch=$(OS_ARCH)'" -o $@ github.com/ntons/libra/librad
#echo $(VERSION) > VERSION

image:
	docker build -t ntons/librad:latest .
	docker tag ntons/librad:latest ntons/librad:$(VERSION)

push:
	docker push ntons/librad:latest
	docker push ntons/librad:$(VERSION)

