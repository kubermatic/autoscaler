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

package clusterapi_test

import (
	"reflect"
	"testing"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/types"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
)

type fakeCluster struct {
	machineSets []fakeMachineSet
}

var _ types.ClusterManager = (*fakeCluster)(nil)

type fakeMachineSet struct {
	clusterManager *fakeCluster
	name           string
	minSize        int
	maxSize        int
	curSize        int
}

var _ types.MachineSet = (*fakeMachineSet)(nil)

func (f *fakeCluster) Refresh() error {
	panic("should not be called")
}

func (f *fakeCluster) Cleanup() error {
	panic("should not be called")
}

func (f *fakeCluster) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	panic("should not be called")
}

func (f *fakeCluster) GetMachineSets(namespace string) ([]types.MachineSet, error) {
	result := make([]types.MachineSet, len(f.machineSets))

	for i := range f.machineSets {
		result[i] = &f.machineSets[i]
	}

	return result, nil
}

func (f *fakeMachineSet) Name() string {
	return f.name
}

func (f *fakeMachineSet) MinSize() int {
	return f.minSize
}

func (f *fakeMachineSet) MaxSize() int {
	return f.maxSize
}

func (f *fakeMachineSet) Replicas() int {
	return f.curSize
}

func (f *fakeMachineSet) TargetSize() (int, error) {
	return f.curSize, nil
}

func (f *fakeMachineSet) IncreaseSize(delta int) error {
	f.curSize += delta
	return nil
}

func (f *fakeMachineSet) DeleteNodes([]*apiv1.Node) error {
	return cloudprovider.ErrNotImplemented
}

func (f *fakeMachineSet) DecreaseTargetSize(delta int) error {
	return cloudprovider.ErrNotImplemented
}

func (f *fakeMachineSet) Id() string {
	return f.name
}

func (f *fakeMachineSet) Debug() string {
	return f.name
}

func (f *fakeMachineSet) Nodes() ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (f *fakeMachineSet) TemplateNodeInfo() (*schedulercache.NodeInfo, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (f *fakeMachineSet) Exist() bool {
	return true
}

func (f *fakeMachineSet) Create() (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (f *fakeMachineSet) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (f *fakeMachineSet) Autoprovisioned() bool {
	return false
}

func testProvider(t *testing.T, name string, c *fakeCluster) cloudprovider.CloudProvider {
	t.Helper()
	provider, err := clusterapi.NewProvider(name, c, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	return provider
}

func TestProviderName(t *testing.T) {
	name := clusterapi.ProviderName + "test"
	provider := testProvider(t, name, &fakeCluster{})
	if name != provider.Name() {
		t.Fatalf("expected %q, got %q", name, provider.Name())
	}
}

func TestProviderEmptyNodeGroups(t *testing.T) {
	provider := testProvider(t, clusterapi.ProviderName, &fakeCluster{})
	nodeGroups := provider.NodeGroups()
	if len(nodeGroups) != 0 {
		t.Fatalf("expected 0, got %v", len(nodeGroups))
	}
}

func TestProviderNonEmptyNodeGroups(t *testing.T) {
	expected := []fakeMachineSet{
		fakeMachineSet{
			name:    "foo",
			curSize: 3,
			minSize: 1,
			maxSize: 5,
		},
		fakeMachineSet{
			name:    "bar",
			curSize: 8,
			minSize: 3,
			maxSize: 10,
		},
	}

	clusterManager := &fakeCluster{machineSets: expected}
	provider := testProvider(t, clusterapi.ProviderName, clusterManager)
	nodegroups := provider.NodeGroups()

	if len(expected) != len(nodegroups) {
		t.Fatalf("expected %v, got %v", len(expected), len(nodegroups))
	}

	actual := make([]fakeMachineSet, len(expected))

	for i := range nodegroups {
		actual[i] = fakeMachineSet{
			name:    nodegroups[i].Id(),
			minSize: nodegroups[i].MinSize(),
			maxSize: nodegroups[i].MaxSize(),
		}
		targetSize, err := nodegroups[i].TargetSize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		actual[i].curSize = targetSize
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %+v, got %+v", expected, actual)
	}
}

func TestProviderIncreaseMachineSetSize(t *testing.T) {
	ms := fakeMachineSet{
		name:    "foo",
		minSize: 1,
		maxSize: 5,
		curSize: 3,
	}

	clusterManager := &fakeCluster{
		machineSets: []fakeMachineSet{ms},
	}

	provider := testProvider(t, clusterapi.ProviderName, clusterManager)

	ng := provider.NodeGroups()
	if len(ng) != 1 {
		t.Fatalf("expected 1, got %v", len(ng))
	}

	if err := ng[0].IncreaseSize(0); err == nil {
		t.Fatalf("expected error")
	}

	if err := ng[0].IncreaseSize(100); err == nil {
		t.Fatalf("expected error")
	}

	if err := ng[0].IncreaseSize(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targetSize, err := ng[0].TargetSize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if 4 != targetSize {
		t.Fatalf("expected 4, got %d", targetSize)
	}
}
