package types

type ClusterManager interface {
	Refresh() error
	Cleanup() error
	GetMachineSets(namespace string) ([]MachineSet, error)
	MachineSetForNode(name string) (MachineSet, error)
}

type MachineSet interface {
	Name() string
	Namespace() string
	MinSize() int
	MaxSize() int
	Replicas() int
	SetSize(n int) error
	Nodes() ([]string, error)
	DeleteNodes([]string) error
}
