#!/bin/sh

VERSION=$(cat VERSION)

set -x

docker build -t ntons/librad:latest .
docker tag ntons/librad:latest ntons/librad:${VERSION}

