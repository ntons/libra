#!/bin/sh

VERSION=$(cat VERSION)

set -x && docker build -t ntons/librad:${VERSION} .

