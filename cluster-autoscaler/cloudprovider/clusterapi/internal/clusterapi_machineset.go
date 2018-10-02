package internal

import (
	"fmt"
	"strconv"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1apis "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	machineDeleteAnnotationKey = "sigs.k8s.io/cluster-api-delete-machine"
)

type clusterMachineSet struct {
	*clusterManager
	*v1alpha1apis.MachineSet
	min     int
	max     int
	nodes   []string
	nodeSet map[string]bool
}

func (m *clusterMachineSet) Name() string {
	return m.MachineSet.Name
}

func (m *clusterMachineSet) Namespace() string {
	return m.MachineSet.Namespace
}

func (m *clusterMachineSet) MinSize() int {
	return m.min
}

func (m *clusterMachineSet) MaxSize() int {
	return m.max
}

func (m *clusterMachineSet) Replicas() int {
	if m.MachineSet.Spec.Replicas == nil {
		return 0
	}
	glog.Infof("MS: %q, Replicas=%d", m.MachineSet.Name, *m.MachineSet.Spec.Replicas)
	return int(*m.MachineSet.Spec.Replicas)
}

func (m *clusterMachineSet) SetSize(nreplicas int) error {
	ms, err := m.clusterapi.MachineSets(m.MachineSet.Namespace).Get(m.MachineSet.Name, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Unable to get machineset %q: %v", m.MachineSet.Name, err)
	}

	newMachineSet := ms.DeepCopy()
	replicas := int32(nreplicas)
	newMachineSet.Spec.Replicas = &replicas

	_, err = m.clusterapi.MachineSets(m.MachineSet.Namespace).Update(newMachineSet)
	if err != nil {
		return fmt.Errorf("Unable to update number of replicas of machineset %q: %v", m.MachineSet.Name, err)
	}

	return nil
}

func (m *clusterMachineSet) Nodes() ([]string, error) {
	glog.Infof("%s/%s has nodes %v", m.Namespace(), m.Name(), spew.Sdump(m.nodes))
	return m.nodes, nil
}

func (m *clusterMachineSet) DeleteNodes(nodenames []string) error {
	snapshot := m.getClusterState()

	for _, nodename := range nodenames {
		name, exists := snapshot.NodeMap[machineSetID(m.MachineSet)][nodename]
		if !exists {
			return fmt.Errorf("cannot map nodename %q to machine: %v", nodename)
		}
		machine, err := m.clusterapi.Machines(m.MachineSet.Namespace).Get(name, v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("cannot get machine %s/%s: %v", m.MachineSet.Namespace, name, err)
		}

		machine = machine.DeepCopy()

		if machine.Annotations == nil {
			machine.Annotations = map[string]string{}
		}

		// Annotate machine that it is the chosen one.
		machine.Annotations["sigs.k8s.io/cluster-api-delete-machine"] = time.Now().String()

		_, err = m.clusterapi.Machines(m.MachineSet.Namespace).Update(machine)
		if err != nil {
			return fmt.Errorf("unable to update machine %s: %v", *m, err)
		}
	}

	return nil
}

func (m *clusterMachineSet) String() string {
	return fmt.Sprintf("%s/%s", m.Namespace(), m.Name())
}

func parseLabel(ms *v1alpha1apis.MachineSet, label string, result *int) {
	val, exists := ms.Labels[label]
	if !exists {
		glog.Infof("machineset %s/%s has no label named %q", ms.Namespace, ms.Name, label)
		return
	}

	u, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		glog.Errorf("machineset %s/%s: cannot parse %q as an integral value: %v", ms.Namespace, ms.Name, val, err)
		return
	}

	*result = int(u)
}

func newClusterMachineSet(m *clusterManager, ms *v1alpha1apis.MachineSet, nodes []string) *clusterMachineSet {
	cms := clusterMachineSet{
		clusterManager: m,
		MachineSet:     ms,
		nodes:          nodes,
		nodeSet:        make(map[string]bool),
	}

	parseLabel(ms, "sigs.k8s.io/cluster-api-autoscaler-node-group-min-size", &cms.min)
	parseLabel(ms, "sigs.k8s.io/cluster-api-autoscaler-node-group-max-size", &cms.max)

	for i := range nodes {
		cms.nodeSet[nodes[i]] = true
	}

	return &cms
}
