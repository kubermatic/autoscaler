package internal

import (
	"fmt"
	"strconv"

	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	v1alpha1apis "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type clusterMachineSet struct {
	*clusterManager
	*v1alpha1apis.MachineSet
}

func getLabelAsInt(clusterMachineSet *v1alpha1apis.MachineSet, label string) (int, error) {
	if _, exists := clusterMachineSet.Labels[label]; !exists {
		return 0, fmt.Errorf("%q label not found", label)
	}
	u, err := strconv.ParseUint(clusterMachineSet.Labels[label], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as an integral value: %v", clusterMachineSet.Labels[label], err)
	}
	return int(u), nil
}

func machineSetMinSize(clusterMachineSet *v1alpha1apis.MachineSet) (int, error) {
	return getLabelAsInt(clusterMachineSet, "sigs.k8s.io/clusterManager-autoscaler-node-group-min-size")
}

func machineSetMaxSize(clusterMachineSet *v1alpha1apis.MachineSet) (int, error) {
	return getLabelAsInt(clusterMachineSet, "sigs.k8s.io/clusterManager-autoscaler-node-group-max-size")
}

func (m *clusterMachineSet) Name() string {
	return m.MachineSet.Name
}

func (m *clusterMachineSet) Namespace() string {
	return m.MachineSet.Namespace
}

func (m *clusterMachineSet) MinSize() int {
	sz, err := machineSetMinSize(m.MachineSet)
	if err != nil {
		glog.Errorf("failed to get minimum size from %s/%s: %v", m.MachineSet.Name, m.MachineSet.Namespace, err)
		return 0
	}
	return sz
}

func (m *clusterMachineSet) MaxSize() int {
	sz, err := machineSetMaxSize(m.MachineSet)
	if err != nil {
		glog.Errorf("failed to get maximum size from %s/%s: %v", m.MachineSet.Name, m.MachineSet.Namespace, err)
		return 0
	}
	return sz
}

func (m *clusterMachineSet) Replicas() int {
	if m.MachineSet.Spec.Replicas == nil {
		return 0
	}
	glog.Infof("MS: %q, Replicas=%d", m.MachineSet.Name, *m.MachineSet.Spec.Replicas)
	return int(*m.MachineSet.Spec.Replicas)
}

func (m *clusterMachineSet) IncreaseSize(delta int) error {
	ms, err := m.clientapi.MachineSets(m.MachineSet.Namespace).Get(m.MachineSet.Name, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Unable to get machineset %q: %v", m.MachineSet.Name, err)
	}

	newMachineSet := ms.DeepCopy()
	replicas := int32(delta)
	newMachineSet.Spec.Replicas = &replicas

	_, err = m.clientapi.MachineSets(m.MachineSet.Namespace).Update(newMachineSet)
	if err != nil {
		return fmt.Errorf("Unable to update number of replicas of machineset %q: %v", m.MachineSet.Name, err)
	}

	return nil
}

func (m *clusterMachineSet) Nodes() ([]string, error) {
	machines, err := m.clientapi.Machines(m.MachineSet.Namespace).List(v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(m.MachineSet.Spec.Selector.MatchLabels).String(),
	})

	if err != nil {
		return nil, fmt.Errorf("unable to list machines of machineset %q: %v", m.Name(), err)
	}

	result := make([]string, len(machines.Items))

	for i, machine := range machines.Items {
		glog.Infof("MachineSet: %q, nodes[%d]=%q", m.MachineSet.Name, i, machine.Name)
		if machine.Status.NodeRef == nil {
			glog.Errorf("Status.NodeRef of machine %q is nil", machine.Name)
			continue
		}
		if machine.Status.NodeRef.Kind != "Node" {
			glog.Errorf("Status.NodeRef of machine %q does not reference a node (rather %q)", machine.Name, machine.Status.NodeRef.Kind)
			continue
		}
		result[i] = machine.Status.NodeRef.Name
	}

	return result, nil
}
