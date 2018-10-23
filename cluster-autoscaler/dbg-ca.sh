#!/bin/bash

filename=${1:-./cluster-autoscaler}; shift

echo "Debugging $filename"

dlv "$@" exec \
    $filename -- \
    --kubeconfig ${KUBECONFIG:?} \
    -v=4 \
    --cloud-provider=cluster-api \
    --scale-down-delay-after-failure=10s \
    --scale-down-unneeded-time=10s \
    --scale-down-delay-after-add=10s \
    --leader-elect=false \
    --logtostderr \
    --scan-interval=10s \
    --balance-similar-node-groups
