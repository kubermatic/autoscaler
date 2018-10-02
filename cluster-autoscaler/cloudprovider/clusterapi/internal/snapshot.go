//go:generate ./vendor-cluster-api.sh

package internal

import (
	"fmt"

	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	v1alpha1apis "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type MachineSetID string

type clusterSnapshot struct {
	NodeMap             map[MachineSetID]map[string]string
	MachineSetMap       map[MachineSetID]*clusterMachineSet
	NodeToMachineSetMap map[string]MachineSetID
	MachineSetNodeMap   map[MachineSetID][]string
}

func machineSetID(m *v1alpha1.MachineSet) MachineSetID {
	return MachineSetID(fmt.Sprintf("%s/%s", m.Namespace, m.Name))
}

func getMachinesInMachineSet(m *clusterManager, ms *v1alpha1apis.MachineSet) ([]*v1alpha1apis.Machine, error) {
	machines, err := m.clusterapi.Machines(ms.Namespace).List(v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(ms.Spec.Selector.MatchLabels).String(),
	})

	if err != nil {
		return nil, fmt.Errorf("unable to list machines in namespace %s: %v", ms.Namespace, err)
	}

	names := make([]string, len(machines.Items))
	result := make([]*v1alpha1apis.Machine, len(machines.Items))

	for i := range machines.Items {
		names[i] = machines.Items[i].Name
		result[i] = &machines.Items[i]
	}

	glog.Infof("%d machines in machineset %s/%s: %#v", len(result), ms.Namespace, ms.Name, names)

	return result, nil
}

func getMachineSetsInNamespace(m *clusterManager, namespace string) ([]*v1alpha1apis.MachineSet, error) {
	machineSets, err := m.clusterapi.MachineSets(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list machinesets in namespace %q: %v", namespace, err)
	}

	names := make([]string, len(machineSets.Items))
	result := make([]*v1alpha1apis.MachineSet, len(machineSets.Items))

	for i := range machineSets.Items {
		names[i] = machineSets.Items[i].Name
		result[i] = &machineSets.Items[i]
	}

	glog.Infof("%d machinesets in namespace %q: %#v", len(result), namespace, names)

	return result, nil
}

func getNamespaces(m *clusterManager) ([]string, error) {
	namespaces, err := m.kubeclient.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]string, len(namespaces.Items))

	for i := range namespaces.Items {
		result[i] = namespaces.Items[i].Name
	}

	glog.Infof("%d namespaces: %#v", len(result), result)

	return result, nil
}

func mapMachinesForMachineSet(m *clusterManager, snapshot *clusterSnapshot, ms *v1alpha1apis.MachineSet) error {
	machines, err := getMachinesInMachineSet(m, ms)
	if err != nil {
		return err
	}

	msid := machineSetID(ms)

	snapshot.NodeMap[msid] = make(map[string]string)

	for _, machine := range machines {
		if machine.Status.NodeRef == nil {
			glog.Errorf("Status.NodeRef of machine %q is nil", machine.Name)
			continue
		}
		if machine.Status.NodeRef.Kind != "Node" {
			glog.Error("Status.NodeRef of machine %q does not reference a node (rather %q)", machine.Name, machine.Status.NodeRef.Kind)
			continue
		}
		snapshot.NodeMap[msid][machine.Status.NodeRef.Name] = machine.Name
		snapshot.NodeToMachineSetMap[machine.Status.NodeRef.Name] = msid
		snapshot.MachineSetNodeMap[msid] = append(snapshot.MachineSetNodeMap[msid], machine.Status.NodeRef.Name)
	}

	return nil
}

func mapMachineSetsForNS(m *clusterManager, snapshot *clusterSnapshot, namespace string) error {
	machineSets, err := getMachineSetsInNamespace(m, namespace)
	if err != nil {
		return err
	}

	for _, ms := range machineSets {
		if err := mapMachinesForMachineSet(m, snapshot, ms); err != nil {
			return err
		}
		msid := machineSetID(ms)
		snapshot.MachineSetMap[msid] = newClusterMachineSet(m, ms, snapshot.MachineSetNodeMap[msid])
	}

	return nil
}

func getClusterSnaphot(m *clusterManager) (*clusterSnapshot, error) {
	snapshot := newEmptySnapshot()
	namespaces, err := getNamespaces(m)
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces {
		if err := mapMachineSetsForNS(m, snapshot, ns); err != nil {
			return nil, err
		}
	}

	glog.Infof("cluster snapshot: %+v", snapshot)

	return snapshot, err
}

func newEmptySnapshot() *clusterSnapshot {
	return &clusterSnapshot{
		NodeMap:             make(map[MachineSetID]map[string]string),
		MachineSetMap:       make(map[MachineSetID]*clusterMachineSet),
		MachineSetNodeMap:   make(map[MachineSetID][]string),
		NodeToMachineSetMap: make(map[string]MachineSetID),
	}
}
