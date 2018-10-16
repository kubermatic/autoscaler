#!/bin/bash

dbg-ca() {
    local filename=${1:-./cluster-autoscaler}; shift
    echo "Debugging $filename"
    set -o xtrace
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
	--scan-interval 10s \
	--balance-similar-node-groups
    set +o xtrace
}

dbg-ca-headless() {
    dbg-ca ${1:-./cluster-autoscaler} --log --headless --listen 127.0.0.1:12345 --api-version=2
}

