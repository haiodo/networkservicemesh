package providers

import (
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	v1 "k8s.io/api/core/v1"
	"time"
)


// Instanceof of one cluster
// Some of cluster cloud be alive by default, it could bare metal cluster,
// and we do not need to perform any startup, shutdown code on them.
type ClusterInstance interface {
	// Return cluster Kubernetes configuration file .config location.
	GetClusterConfig() (string, error)

	// Perform startup of cluster
	Start( manager execmanager.ExecutionManager, timeout time.Duration ) error
	// Destroy cluster
	// Should destroy cluster with timeout passed, if time is left should report about error.
	Destroy(manager execmanager.ExecutionManager, timeout time.Duration) error

	// Return root folder to store test artifacts associated with this cluster
	GetRoot() string

	// Is cluster is running right now
	IsRunning() bool
	CheckIsAlive() ([]v1.Node, error)
}

type ClusterProvider interface {
	// Create a cluster based on parameters
	// CreateCluster - Creates a cluster instance and put Kubernetes config file into clusterConfigRoot
	// could fully use clusterConfigRoot folder for any temporary files related to cluster.
	CreateCluster( config *config.ClusterProviderConfig ) (ClusterInstance, error)

	// Check if config are valid and all parameters required by this cluster are fit.
	ValidateConfig( config *config.ClusterProviderConfig ) error
}

type ClusterProviderFunction func(root string) ClusterProvider

var ClusterProviderFactories = map[string]ClusterProviderFunction{}