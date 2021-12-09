#!/bin/bash

. $(dirname $0)/env.sh

while [ $# -gt 0 ]; do
    REGISTRY="$1ccr.ccs.tencentyun.com"
    IMAGE="onemore/librad"
    VERSION=$(cat VERSION)
    SOURCE=${IMAGE}:${VERSION}
    TARGET=${REGISTRY}/${SOURCE}

    read -p "Release '${TARGET}', are you sure? (y/n)" -n 1 -r yn
    echo
    [ "x$yn" != "xy" ] && exit 1

    docker tag ${SOURCE} ${TARGET} && docker push ${TARGET}

    shift
done
