#!/bin/sh

IMAGE="onemore/librad"

VERSION=$(cat VERSION)

set -x && docker build -t ${IMAGE}:${VERSION} .

