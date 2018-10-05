package clusterapi

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/types"
	schedulercache "k8s.io/kubernetes/pkg/scheduler/cache"
)

// NodeGroup is implementation of the NodeGroup interface
type NodeGroup struct {
	manager    types.ClusterManager
	machineSet types.MachineSet
}

var _ cloudprovider.NodeGroup = (*NodeGroup)(nil)

func NewNodeGroup(manager types.ClusterManager, machineSet types.MachineSet) *NodeGroup {
	return &NodeGroup{
		manager:    manager,
		machineSet: machineSet,
	}
}

// MaxSize returns maximum size of the node group.
func (ng *NodeGroup) MaxSize() int {
	return ng.machineSet.MaxSize()
}

// MinSize returns minimum size of the node group.
func (ng *NodeGroup) MinSize() int {
	return ng.machineSet.MinSize()
}

// TargetSize returns the current target size of the node group. It is possible that the
// number of nodes in Kubernetes is different at the moment but should be equal
// to Size() once everything stabilizes (new nodes finish startup and registration or
// removed nodes are deleted completely). Implementation required.
func (ng *NodeGroup) TargetSize() (int, error) {
	return ng.machineSet.Replicas(), nil
}

// IncreaseSize increases the size of the node group. To delete a node you need
// to explicitly name it and use DeleteNode. This function should wait until
// node group size is updated. Implementation required.
func (ng *NodeGroup) IncreaseSize(delta int) error {
	if delta <= 0 {
		return fmt.Errorf("size increase must be positive")
	}
	size := ng.machineSet.Replicas()
	if size+delta > ng.MaxSize() {
		return fmt.Errorf("size increase too large - desired:%d max:%d", size+delta, ng.MaxSize())
	}
	return ng.machineSet.SetSize(size + delta)
}

// DeleteNodes deletes nodes from this node group. Error is returned either on
// failure or if the given node doesn't belong to this node group. This function
// should wait until node group size is updated. Implementation required.
func (ng *NodeGroup) DeleteNodes(nodes []*apiv1.Node) error {
	names := make([]string, len(nodes))
	for i := range nodes {
		names[i] = nodes[i].Name
	}
	if err := ng.machineSet.DeleteNodes(names); err != nil {
		return err
	}
	return nil
}

// DecreaseTargetSize decreases the target size of the node group. This function
// doesn't permit to delete any existing node and can be used only to reduce the
// request for new nodes that have not been yet fulfilled. Delta should be negative.
// It is assumed that cloud provider will not delete the existing nodes when there
// is an option to just decrease the target. Implementation required.
func (ng *NodeGroup) DecreaseTargetSize(delta int) error {
	if delta >= 0 {
		return fmt.Errorf("size decrease must be negative")
	}

	size, err := ng.TargetSize()
	if err != nil {
		return err
	}

	nodes, err := ng.Nodes()
	if err != nil {
		return err
	}

	if int(size)+delta < len(nodes) {
		return fmt.Errorf("attempt to delete existing nodes targetSize:%d delta:%d existingNodes: %d",
			size, delta, len(nodes))
	}

	return ng.machineSet.SetSize(size + delta)
}

// Id returns an unique identifier of the node group.
func (ng *NodeGroup) Id() string {
	return ng.machineSet.Name()
}

// Debug returns a string containing all information regarding this node group.
func (ng *NodeGroup) Debug() string {
	return fmt.Sprintf("%s (min: %d, max: %d, replicas: %d)", ng.Id(), ng.MinSize(), ng.MaxSize(), ng.machineSet.Replicas())
}

// Nodes returns a list of all nodes that belong to this node group.
func (ng *NodeGroup) Nodes() ([]string, error) {
	return ng.machineSet.Nodes()
}

// TemplateNodeInfo returns a schedulercache.NodeInfo structure of an empty
// (as if just started) node. This will be used in scale-up simulations to
// predict what would a new node look like if a node group was expanded. The returned
// NodeInfo is expected to have a fully populated Node object, with all of the labels,
// capacity and allocatable information as well as all pods that are started on
// the node by default, using manifest (most likely only kube-proxy). Implementation optional.
func (ng *NodeGroup) TemplateNodeInfo() (*schedulercache.NodeInfo, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// Exist checks if the node group really exists on the cloud provider side. Allows to tell the
// theoretical node group from the real one. Implementation required.
func (ng *NodeGroup) Exist() bool {
	return true
}

// Create creates the node group on the cloud provider side. Implementation optional.
func (ng *NodeGroup) Create() (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrAlreadyExist
}

// Delete deletes the node group on the cloud provider side.
// This will be executed only for autoprovisioned node groups, once their size drops to 0.
// Implementation optional.
func (ng *NodeGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

// Autoprovisioned returns true if the node group is autoprovisioned. An autoprovisioned group
// was created by CA and can be deleted when scaled to 0.
func (ng *NodeGroup) Autoprovisioned() bool {
	return false
}
