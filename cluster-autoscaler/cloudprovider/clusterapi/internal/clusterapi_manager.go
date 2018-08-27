//go:generate ./vendor-cluster-api.sh

package internal

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/clusterapi/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

const (
	refreshInterval  = 1 * time.Minute
	defaultNamespace = "default" // TODO(frobware)
)

type clusterManager struct {
	lastRefresh    time.Time
	clientapi      v1alpha1.ClusterV1alpha1Interface
	resourceLimits *cloudprovider.ResourceLimiter
}

func (m *clusterManager) Cleanup() error {
	return nil
}

func (m *clusterManager) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return m.resourceLimits, nil
}

func (m *clusterManager) GetMachineSets(namespace string) ([]types.MachineSet, error) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	clusterMachineSets, err := m.clientapi.MachineSets(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("cannot list machinesets: %v", err))
	}

	result := make([]types.MachineSet, len(clusterMachineSets.Items))

	for i := range clusterMachineSets.Items {
		glog.Infof("[MachineSet:%v] %s/%s", i,
			clusterMachineSets.Items[i].Namespace,
			clusterMachineSets.Items[i].Name)
		result[i] = &clusterMachineSet{
			clusterManager: m,
			MachineSet:     &clusterMachineSets.Items[i],
		}
	}

	return result, nil
}

func (m *clusterManager) MachineSetForNode(node string) (types.MachineSet, error) {
	machineSets, err := m.GetMachineSets("")
	if err != nil {
		return nil, err
	}

	for i := range machineSets {
		if node == machineSets[i].Name() {
			return machineSets[i], nil
		}
	}

	return nil, fmt.Errorf("node %q not found", node)
}

func (m *clusterManager) Refresh() error {
	if m.lastRefresh.Add(refreshInterval).After(time.Now()) {
		return nil
	}
	return nil
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
		clientapi: clientapi.ClusterV1alpha1(),
	}, nil
}
