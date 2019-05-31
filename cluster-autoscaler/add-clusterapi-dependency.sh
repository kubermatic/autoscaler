#!/usr/bin/env bash

set -euo pipefail

CLUSTERAPI_VERSION=4e465579634471d0a2ceb0e649dce733308299a9
CODEGENERATOR_VERSION=release-1.14
SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")

cd $(go env GOPATH)/src/sigs.k8s.io/cluster-api
git checkout $CLUSTERAPI_VERSION
cd -

rm -rf vendor/sigs.k8s.io/cluster-api/pkg/apis
mkdir -p vendor/sigs.k8s.io/cluster-api/pkg/apis
cp -r $(go env GOPATH)/src/sigs.k8s.io/cluster-api/pkg/apis/* vendor/sigs.k8s.io/cluster-api/pkg/apis

if ! [[ -d  $(go env GOPATH)/src/k8s.io/code-generator ]]; then
  git clone git@github.com:kubernetes/code-generator.git $(go env GOPATH)/src/k8s.io/code-generator
fi
cd $(go env GOPATH)/src/k8s.io/code-generator
git fetch
git checkout $CODEGENERATOR_VERSION
cd -

$(go env GOPATH)/src/k8s.io/code-generator/generate-groups.sh all \
  k8s.io/autoscaler/cluster-autoscaler/client/clusterapi \
  sigs.k8s.io/cluster-api/pkg/apis \
  cluster:v1alpha1 \
  --go-header-file=${SCRIPT_ROOT}/../hack/boilerplate/boilerplate.go.txt

sed -i 's#sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake#k8s.io/autoscaler/cluster-autoscaler/client/clusterapi/clientset/versioned/fake#g' ./cloudprovider/clusterapi/*
sed -i 's#sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset#k8s.io/autoscaler/cluster-autoscaler/client/clusterapi/clientset/versioned#g' ./cloudprovider/clusterapi/*
sed -i 's#sigs.k8s.io/cluster-api/pkg/client/informers_generated/externalversions#k8s.io/autoscaler/cluster-autoscaler/client/clusterapi/informers/externalversions#g' ./cloudprovider/clusterapi/*
sed -i 's#sigs.k8s.io/cluster-api/pkg/client/informers_generated/externalversions/cluster/v1alpha1#k8s.io/autoscaler/cluster-autoscaler/client/clusterapi/informers/externalversion/cluster/v1alpha1#g' \
  ./cloudprovider/clusterapi/*
find vendor/sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1 -name '*_test.go' -delete

# Controller-Runtime has its own way of adding the scheme. This requires the
# sigs.k8s.io/controller-runtime/pkg/runtime/scheme package which is incompatible with
# Kubernetes 1.14 so we have to rewrite register.go and remove the init funcs ¯\_(ツ)_/¯
cat <<EOF >vendor/sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1/register.go
/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha1 contains API Schema definitions for the cluster v1alpha1 API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=sigs.k8s.io/cluster-api/pkg/apis/cluster
// +k8s:defaulter-gen=TypeMeta
// +groupName=cluster.k8s.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	//"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: "cluster.k8s.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	//SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, addDefaultingFuncs)

	// AddToScheme adds registered types to the builder.
	// Required by pkg/client/...
	// TODO(pwittrock): Remove this after removing pkg/client/...
	AddToScheme = SchemeBuilder.AddToScheme
)

// Required by pkg/client/listers/...
// TODO(pwittrock): Remove this after removing pkg/client/...
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Machine{},
		&MachineList{},
		&MachineSet{},
		&MachineSetList{},
		&MachineDeployment{},
		&MachineDeploymentList{},
		&MachineClass{},
		&MachineClassList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return nil
}
EOF
sed -i '/SchemeBuilder.Register/d' vendor/sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1/*.go
