#!/bin/bash

rm -rf vendor
mkdir vendor
cd vendor
mkdir new-sigs.k8s.io
(cd new-sigs.k8s.io; git clone --depth=1 https://github.com/kubernetes-sigs/cluster-api)
mv new-sigs.k8s.io/cluster-api/vendor/* .
mv new-sigs.k8s.io/* sigs.k8s.io/
rmdir sigs.k8s.io/cluster-api/vendor

# clean up duplicates which result in panics at runtime
rm -rf github.com/golang/glog
rm -rf golang.org/x/net/trace
