//go:generate ./vendor-cluster-api.sh

package internal

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/types"
	"k8s.io/client-go/tools/clientcmd"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	v1alpha1apis "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

const (
	refreshInterval  = 1 * time.Minute
	defaultNamespace = "openshift-cluster-api" // TODO(frobware)
)

type MachineSetID string

type clusterSnapshot struct {
	MachineSetMap          map[MachineSetID]*clusterMachineSet
	MachineToMachineSetMap map[string]MachineSetID
	MachineSetNodeMap      map[MachineSetID][]string
}

type clusterManager struct {
	lastRefresh          time.Time
	clientapi            v1alpha1apis.ClusterV1alpha1Interface
	resourceLimits       *cloudprovider.ResourceLimiter
	clusterSnapshotMutex sync.Mutex
	clusterSnapshot      *clusterSnapshot
}

func init() {
	spew.Config = spew.ConfigState{
		DisablePointerAddresses: true,
		DisableCapacities:       true,
		SortKeys:                true,
		SpewKeys:                true,
		Indent:                  "  ",
	}
}

func (m *clusterManager) Cleanup() error {
	return nil
}

func (m *clusterManager) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return m.resourceLimits, nil
}

func (m *clusterManager) GetMachineSets(namespace string) ([]types.MachineSet, error) {
	result := []types.MachineSet{}

	for _, ms := range m.getClusterState().MachineSetMap {
		if ms.hasBounds() {
			result = append(result, ms)
		}
	}

	glog.Infof("NODE GROUPS %v", spew.Sdump(result))

	return result, nil
}

func (m *clusterManager) MachineSetForNode(nodename string) (types.MachineSet, error) {
	snapshot := m.getClusterState()
	if key, exists := snapshot.MachineToMachineSetMap[nodename]; exists {
		glog.Infof("MachineSetForNode: %q is node %q", nodename, key)
		return snapshot.MachineSetMap[key], nil
	}
	return nil, nil
}

func (m *clusterManager) getClusterState() *clusterSnapshot {
	m.clusterSnapshotMutex.Lock()
	defer m.clusterSnapshotMutex.Unlock()
	return m.clusterSnapshot
}

func (m *clusterManager) setClusterState(s *clusterSnapshot) {
	m.clusterSnapshotMutex.Lock()
	defer m.clusterSnapshotMutex.Unlock()
	m.clusterSnapshot = s
}

func (m *clusterManager) Refresh() error {
	if m.lastRefresh.Add(refreshInterval).After(time.Now()) && m.clusterSnapshot != nil {
		return nil
	}
	return m.forceRefresh()
}

func (m *clusterManager) forceRefresh() error {
	s, err := m.clusterRefresh(defaultNamespace)
	if err == nil {
		m.lastRefresh = time.Now()
		glog.Infof("cluster refreshed at %v\n%v", m.lastRefresh, spew.Sdump(s))
		m.setClusterState(s)
	}
	return err
}

func NewClusterManager(do cloudprovider.NodeGroupDiscoveryOptions) (*clusterManager, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		kubeconfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}

	clientapi, err := clientset.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not create client for talking to the apiserver: %v", err)
	}

	return &clusterManager{
		clientapi:       clientapi.ClusterV1alpha1(),
		clusterSnapshot: newClusterSnapshot(),
	}, nil
}

func (m *clusterManager) clusterRefresh(namespace string) (*clusterSnapshot, error) {
	snapshot := newClusterSnapshot()
	machineSets, err := m.clientapi.MachineSets(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list machinesets in the %q namespace: %v", namespace, err)
	}

	for i := range machineSets.Items {
		ms := &clusterMachineSet{
			clusterManager: m,
			MachineSet:     &machineSets.Items[i],
		}

		msid := machineSetID(ms.MachineSet)
		machines, err := m.clientapi.Machines(namespace).List(v1.ListOptions{
			LabelSelector: labels.SelectorFromSet(ms.MachineSet.Spec.Selector.MatchLabels).String(),
		})

		if err != nil {
			glog.Errorf("unable to get machines for %q: %v", msid)
			continue
		}

		snapshot.MachineSetMap[msid] = ms
		snapshot.MachineSetNodeMap[msid] = []string{}

		for _, machine := range machines.Items {
			if machine.Status.NodeRef == nil {
				glog.Errorf("Status.NodeRef of machine %q is nil", machine.Name)
				continue
			}
			if machine.Status.NodeRef.Kind != "Node" {
				glog.Error("Status.NodeRef of machine %q does not reference a node (rather %q)", machine.Name, machine.Status.NodeRef.Kind)
				continue
			}

			snapshot.MachineToMachineSetMap[machine.Name] = msid
			snapshot.MachineSetNodeMap[msid] = append(snapshot.MachineSetNodeMap[msid], machine.Status.NodeRef.Name)
		}

		ms.nodes = snapshot.MachineSetNodeMap[msid]

		glog.Infof("MachineSet: %q has nodes %v", msid, snapshot.MachineSetNodeMap[msid])
	}

	return snapshot, nil
}

func machineSetID(m *v1alpha1.MachineSet) MachineSetID {
	return MachineSetID(fmt.Sprintf("%s/%s", m.Namespace, m.Name))
}

func newClusterSnapshot() *clusterSnapshot {
	return &clusterSnapshot{
		MachineSetMap:          make(map[MachineSetID]*clusterMachineSet),
		MachineSetNodeMap:      make(map[MachineSetID][]string),
		MachineToMachineSetMap: make(map[string]MachineSetID),
	}
}
