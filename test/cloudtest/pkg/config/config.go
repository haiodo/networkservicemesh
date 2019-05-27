package config



type ClusterProviderConfig struct {
	Name string `yaml:"name"`	// name of provider, GKE, Azure, etc.
	Kind string `yaml:"kind"`	// register provider type, 'generic', 'gke', etc.
	Instances int `yaml:"instances"` // Number of required instances, executions will be split between instances.
	AverageStartTime int `yaml:"average-start-time"` // Average time to start one instance
	AverageShutdownTime int `yaml:"average-shutdown-time"` // Average time to start one instance
	Timeout int `yaml:"timeout"`	// Timeout for start, stop
	RetryCount int `yaml:"retry"` // A count of start retrying steps.
	Enabled bool `yaml:"enabled"` // Is it enabled by default or not
	Parameters map[string]string `yaml:"parameters"` // A parameters specific for provider
}

type CloudTestConfig struct {
	Version string `yaml:"version"`		// Provider file version, 1.0
	Providers []ClusterProviderConfig `yaml:"providers"`
	KubernetesConfigRoot string `yaml:"root"` // A provider stored configurations root.
	Reporting struct { // A junit report location.
		JUnitReportFile string `yaml:"junit-report"`
	} `yaml:"reporting"`

	Executions []struct { // Executions, every execution execute some tests agains configured set of clusters
		Name string `yaml:"name"` // Execution name
		Tags []string `yaml:"tags"`	// A list of tags for this configured execution.
		PackageRoot string `yaml:"root"` // A package root for this test execution, default .
		TestTimeout string `yaml:"timeout"` // Invidiaul test timeout, "60" passed to gotest, in seconds
		ExtraOptions []string `yaml:"extra-options"` // Extra options to pass to gotest
		ClusterCount int `yaml:"cluster-count"` // A number of clusters required for this execution, default 1
		KubernetesEnv []string `yaml:"kubernetes-env"` // Names of environment variables to put cluster names inside.
		ClusterSelector []struct { // Pass a cluster selector, if not specified will use all clusters
			ClusterName string `yaml:"cluster-name"` //  Set cluster name to be matched for inter.domain testing
		} `yaml:"selector"`
		// Multi Cluster tests
	} `yaml:"executions"`
	Timeout int `yaml:"timeout"` // Global timeout in minutes
}
