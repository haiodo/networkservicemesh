package commands

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const (
	DefaultConfigFile string = ".cloud_test.yaml"
)

var rootCmd = &cobra.Command{
	Use:   "cloud_test",
	Short: "NSM Cloud Test is cloud helper continuous integration testing tool",
	Long:  `Allow to execute all set of individual tests across all clouds provided.`,
	Run:   CloudTestRun,
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var cloudTestConfig config.CloudTestConfig

type Arguments struct {
	providerConfig string   // A folder to start scaning for tests inside.
	clusters       []string // A list of enabled clusters from configuration.
}

var cmdArguments *Arguments = &Arguments{
	providerConfig: DefaultConfigFile,
	clusters:       []string{},
}

type clustersGroup struct {
	instances []providers.ClusterInstance
	provider  providers.ClusterProvider
	config    *config.ClusterProviderConfig
}

type testTask struct {
	test    *TestEntry
	cluster *clustersGroup
}

func CloudTestRun(cmd *cobra.Command, args []string) {
	var configFileContent []byte
	var err error
	if cmdArguments.providerConfig == "" {
		cmdArguments.providerConfig = DefaultConfigFile
	}

	configFileContent, err = ioutil.ReadFile(cmdArguments.providerConfig)
	if err != nil {
		logrus.Errorf("Failed to read config file %v", err)
		return
	}

	err = yaml.Unmarshal(configFileContent, &cloudTestConfig)
	if err != nil {
		logrus.Errorf("Failed to parse configuration file: %v", err)
		return
	}
	logrus.Infof("Configuration file loaded successfully...")

	manager := execmanager.NewExecutionManager(cloudTestConfig.ConfigRoot)

	clusters := createClusters(manager, cloudTestConfig)

	if len(clusters) == 0 {
		logrus.Errorf("There is no clusters defined. Exiting...")
		os.Exit(1)
	}

	// Collect tests
	logrus.Infof("Finding tests")
	tests := findTests(manager)

	if len(tests) == 0 {
		logrus.Errorf("There is no tests defined. Exiting...")
	}

	tasks := []*testTask{}
	running := []*testTask{}
	completed := []*testTask{}

	// Fill tasks to be executed..
	for _, test := range tests {
		for _, cluster := range clusters {
			if (len(test.ExecutionConfig.ClusterSelector) > 0 && utils.Contains(test.ExecutionConfig.ClusterSelector, cluster.config.Name)) ||
				len(test.ExecutionConfig.ClusterSelector) == 0 {
				// Cluster selector is defined we need to add tasks for individual cluster only
				tasks = append(tasks, &testTask{
					test:    test,
					cluster: cluster,
				})
			}
		}
	}

	operationChannel := make(chan *testTask, 1)
	clusterCreateChannel := make(chan providers.ClusterInstance, 1)


	logrus.Infof("Starting test execution")
	for len(tasks) > 0 && len(running) > 0 {
		// WE take 1 test task from list and do execution.

		if len(tasks) > 0 {
			// Lets check if we have cluster required and start it
			
		}

		select {
		case jobDone := <-operationChannel:
			logrus.Infof("Tasks completed %v", jobDone)
		case startedInstance := <-clusterCreateChannel:
			logrus.Infof("CLuster instance are created %v", startedInstance)
		}
	}
	logrus.Infof("Completed tasks %v", len(completed))
}

func createClusters(manager execmanager.ExecutionManager, testConfig config.CloudTestConfig) []*clustersGroup {
	clusters := []*clustersGroup{}
	clusterProviders := createClusterProviders(manager)

	for _, cl := range cloudTestConfig.Providers {
		for _, cc := range cmdArguments.clusters {
			if cl.Name == cc {
				if !cl.Enabled {
					logrus.Infof("Enabling config:: %v", cl.Name)
				}
				cl.Enabled = true
			}
		}
		if cl.Enabled {
			logrus.Infof("Initialize provider for config:: %v %v", cl.Name, cl.Kind)
			if provider, ok := clusterProviders[cl.Kind]; !ok {
				logrus.Errorf("Cluster provider %s are not found...", cl.Kind)
				os.Exit(1)
			} else {
				instances := []providers.ClusterInstance{}
				for i := 0; i < cl.Instances; i++ {
					cluster, err := provider.CreateCluster(&cl)
					if err != nil {
						logrus.Errorf("Failed to create cluster instance. Error %v", err)
						os.Exit(1)
					}
					instances = append(instances, cluster)
				}
				clusters = append(clusters, &clustersGroup{
					provider:  provider,
					instances: instances,
					config:    &cl,
				})
			}
		}
	}
	return clusters
}

func findTests(manager execmanager.ExecutionManager) []*TestEntry {
	var tests []*TestEntry
	for _, exec := range cloudTestConfig.Executions {
		execTests, err := GetTestConfiguration(manager, exec.PackageRoot, exec.Tags)
		if err != nil {
			logrus.Errorf("Failed during test lookup %v", err)
		}
		tests = append(tests, execTests...)
	}
	logrus.Infof("Total tests found: %v", len(tests))
	return tests
}

func createClusterProviders(manager execmanager.ExecutionManager) map[string]providers.ClusterProvider {
	clusterProviders := map[string]providers.ClusterProvider{}
	for key, factory := range providers.ClusterProviderFactories {
		if _, ok := clusterProviders[key]; ok {
			logrus.Errorf("Re-definition of cluster provider... Exiting")
			os.Exit(1)
		}
		clusterProviders[key] = factory(manager.GetRoot(key))
	}
	return clusterProviders
}

func ExecuteCloudTest() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&cmdArguments.providerConfig, "config", "", "", "Config file for providers, default="+DefaultConfigFile)
	rootCmd.Flags().StringArrayVarP(&cmdArguments.clusters, "clusters", "c", []string{}, "Enable disable cluster configs, default use from config. Cloud be used to test agains selected configuration or locally...")

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of cloud_test",
		Long:  `All software has versions. This is Hugo's`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Cloud Test v0.9 -- HEAD")
		},
	}
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
}
