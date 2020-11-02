#!/bin/bash

#[ -z "$NODE_CLUSTER" ] && echo "env NODE_CLUSTER is required" && exit 1
#[ -z "$NODE_ID" -a -n "$NODE_CLUSTER" ] && NODE_ID=${NODE_CLUSTER}_$(./tools/uuid)
[ -z "$NODE_ID" ] && NODE_ID=$(./tools/uuid)

# render envoy configuration template
if [ -n "$ENVOY_CONFIG_TEMPLATE" ]; then
    ./tools/render -o $ENVOY_CONFIG $ENVOY_CONFIG_TEMPLATE || exit 1
fi

# validate envoy configuration
echo $ENVOY_CONFIG && cat $ENVOY_CONFIG
$ENVOY -c $ENVOY_CONFIG --mode validate || exit 1

# start envoy
$ENVOY -c $ENVOY_CONFIG &

# wait for envoy
sleep 10

# fetch server package
[ ! -d $SERVER ] && ./tools/fetch.sh

# start service
[ -n "$NODE_ID" -a -n "$NODE_CLUSTER" -a -n "$NODE_WEIGHT" -a -n "$ETCD_PREFIX" -a -n "$ETCD_ENDPOINTS" -a -n "$INGRESS_IFACE" -a -n "$INGRESS_PORT" ]
[ $? -eq 0 ] && set -x && exec ./tools/register \
    --node-id="$NODE_ID" \
    --node-cluster="$NODE_CLUSTER" \
    --node-weight="$NODE_WEIGHT" \
    --etcd-prefix="$ETCD_PREFIX" \
    --etcd-endpoints="$ETCD_ENDPOINTS" \
    --iface="$INGRESS_IFACE" \
    --port="$INGRESS_PORT" \
    &

set -x && cd $SERVER && exec "$@"

