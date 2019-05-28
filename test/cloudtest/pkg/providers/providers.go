package providers

import (
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"time"
)

type ClusterInstance interface {
	// Return cluster Kubernetes configuration file .config location.
	GetClusterConfig() (string, error)

	// Destroy cluster
	// Should destroy cluster with timeout passed, if time is left should report about error.
	Destroy(timeout time.Time) error

	// Return root folder to store test artifacts associated with this cluster
	GetRoot() string
}

type ClusterProvider interface {
	// Create a cluster based on parameters
	// CreateCluster - Creates a cluster instance and put Kubernetes config file into clusterConfigRoot
	// could fully use clusterConfigRoot folder for any temporary files related to cluster.
	CreateCluster( config *config.ClusterProviderConfig, clusterConfigRoot string ) (ClusterInstance, error)

	// Check if config are valid and all parameters required by this cluster are fit.
	ValidateConfig( config *config.ClusterProviderConfig ) error
}
