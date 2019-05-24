package config

import "time"

type ClusterInstance interface {
	// Return cluster Kubernetes configuration file .config location.
	GetClusterConfig() (string, error)

	// Destroy cluster
	// Should destroy cluster with timeout passed, if time is left should report about error.
	Destroy(timeout time.Time) error
}

type ClusterProvider interface {
	// Create a cluster based on parameters
	// CreateCluster - Creates a cluster instance and put Kubernetes config file into clusterConfigRoot
	// could fully use clusterConfigRoot folder for any temporary files related to cluster.
	CreateCluster( config *ClusterProviderConfig, clusterConfigRoot string ) (ClusterInstance, error)

	// Check if config are valid and all parameters required by this cluster are fit.
	ValidateConfig( config *ClusterProviderConfig ) error
}

type ClusterProviderConfig struct {
	Name string `yaml:"name"`	// name of provider, GKE, Azure, etc.
	Kind string `yaml:"kind"`	// register provider type, 'generic', 'gke', etc.
	Instances int `yaml:"instances"` // Number of required instances, executions will be split between instances.
	AverageStartTime int `yaml:"average-start-time"` // Average time to start one instance
	AverageShutdownTime int `yaml:"average-shutdown-time"` // Average time to start one instance
	Timeout int `yaml:"timeout"`	// Timeout for start, stop
	RetryCount int `yaml:"retry"` // A count of start retrying steps.
	Parameters map[string]string `yaml:"parameters"` // A parameters specific for provider
}

type CloudTestConfig struct {
	Version string `yaml:"version"`		// Provider file version, 1.0
	Providers []ClusterProviderConfig `yaml:"providers"`
	KubernetesConfigRoot string `yaml:"./.tests/cloud_test/"` // A provider stored configurations root.
	JUnitReportFile string `yaml:"./.tests/junit.xml"` // A junit report location.

	Executions []struct { // Executions, every execution execute some tests agains configured set of clusters
		Tags []string `yaml:"tags"`	// A list of tags for this configured execution.
		PackageRoot string `yaml:"root"` // A package root for this test execution, default .
		TestTimeout string `yaml:"timeout"` // Invidiaul test timeout, "5m" passed to gotest
		ExtraOptions []string `yaml:"extra-options"` // Extra options to pass to gotest
		ClusterCount int `yaml:"cluster-count"` // A number of clusters required for this execution, default 1
		KubernetesEnv []string `yaml:"kubernetes-env"` // Names of environment variables to put cluster names inside.
		ClusterSelector []struct { // Pass a cluster selector, if not specified will use all clusters
			ClusterName string `yaml:"cluster-name"` //  Set cluster name to be matched for inter.domain testing
		} `yaml:"selector"`
		// Multi Cluster tests
	} `yaml:"executions"`
}
