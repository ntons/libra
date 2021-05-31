#!/bin/sh

VERSION=$(cat VERSION)

case $1 in
    onemore)
        REPOSITORY="ccr.ccs.tencentyun.com/onemore/librad"
        ;;
    *)
        REPOSITORY="ntons/librad"
        ;;
esac

set -x && docker build -t ${REPOSITORY}:${VERSION} .

