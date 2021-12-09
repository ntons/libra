#!/bin/bash


IMAGE="onemore/librad"

VERSION=$(cat VERSION)

echo "Start building at $(date)"

. $(dirname $0)/env.sh

set -x && time docker build -t ${IMAGE}:${VERSION} $(dirname $0)/..

