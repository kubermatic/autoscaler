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

package clusterapi

import (
	"github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/internal"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/types"
	"k8s.io/autoscaler/cluster-autoscaler/utils/errors"
)

// ProviderName is the cloud provider name for the cluster-api
// provider.
const ProviderName = "cluster-api"

type provider struct {
	clusterManager  types.ClusterManager
	name            string
	resourceLimiter *cloudprovider.ResourceLimiter
}

func (p *provider) Name() string {
	glog.Info("provider.Name()")
	return p.name
}

func (p *provider) NodeGroups() []cloudprovider.NodeGroup {
	glog.Info("provider.NodeGroups()")

	machineSets, err := p.clusterManager.GetMachineSets("")
	if err != nil {
		glog.Fatalf("error fetching machinesets: %v", err)
		return nil
	}

	nodeGroups := make([]cloudprovider.NodeGroup, len(machineSets))

	for i, ms := range machineSets {
		nodeGroups[i] = NewNodeGroup(p.clusterManager, ms)
	}

	return nodeGroups
}

func (p *provider) NodeGroupForNode(node *apiv1.Node) (cloudprovider.NodeGroup, error) {
	glog.Infof("provider.NodeGroupForNode(): %q", node.Name)
	ms, err := p.clusterManager.MachineSetForNode(node.Name)
	if err != nil {
		return nil, err
	}
	return NewNodeGroup(p.clusterManager, ms), nil
}

func (p *provider) Pricing() (cloudprovider.PricingModel, errors.AutoscalerError) {
	glog.Info("provider.Pricing()")
	return nil, cloudprovider.ErrNotImplemented
}

func (p *provider) GetAvailableMachineTypes() ([]string, error) {
	glog.Info("provider.GetAvailableMachineTypes()")
	return []string{}, nil
}

func (p *provider) NewNodeGroup(machineType string, labels map[string]string, systemLabels map[string]string, taints []apiv1.Taint, extraResources map[string]resource.Quantity) (cloudprovider.NodeGroup, error) {
	glog.Info("provider.NewNodeGroup()")
	return nil, cloudprovider.ErrNotImplemented
}

func (p *provider) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	glog.Info("provider.GetResourceLimiter()")
	return p.resourceLimiter, nil
}

func (p *provider) Cleanup() error {
	glog.Info("provider.Cleanup()")
	return p.clusterManager.Cleanup()
}

func (p *provider) Refresh() error {
	glog.Info("provider.Refresh()")
	return p.clusterManager.Refresh()
}

func NewProvider(name string, manager types.ClusterManager, rl *cloudprovider.ResourceLimiter) (cloudprovider.CloudProvider, error) {
	return &provider{
		clusterManager:  manager,
		name:            name,
		resourceLimiter: rl,
	}, nil
}

func NewClusterManager(do cloudprovider.NodeGroupDiscoveryOptions) (types.ClusterManager, error) {
	return internal.NewClusterManager(do)
}
