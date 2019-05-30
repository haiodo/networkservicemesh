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
	"time"
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

type clusterState byte

const (
	clusterAdded clusterState = 0
	clusterStarting clusterState = 1
	clusterReady clusterState = 2
	clusterBudy clusterState = 3
	clusterCashed clusterState = 4
	clusterShutdown clusterState = 5
)

type clusterInstance struct {
	instance    providers.ClusterInstance
	state       clusterState
	startFailed int
	task        *testTask
}
type clustersGroup struct {
	instances []*clusterInstance
	provider  providers.ClusterProvider
	config    *config.ClusterProviderConfig
}

type testTask struct {
	test       *TestEntry
	cluster    *clustersGroup
}

type executionContext struct {
	manager          execmanager.ExecutionManager
	clusters         []*clustersGroup
	operationChannel chan *testTask
	tests            []*TestEntry
	tasks            []*testTask
	running          []*testTask
	completed        []*testTask
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

	context := &executionContext {
		manager: execmanager.NewExecutionManager(cloudTestConfig.ConfigRoot),
	}

	// Create cluster instance handles
	context.createClusters(cloudTestConfig)

	if len(context.clusters) == 0 {
		logrus.Errorf("There is no clusters defined. Exiting...")
		os.Exit(1)
	}

	// Collect tests
	logrus.Infof("Finding tests")
	context.tests = findTests(context.manager)

	if len(context.tests) == 0 {
		logrus.Errorf("There is no tests defined. Exiting...")
	}

	context.tasks = []*testTask{}
	context.running = []*testTask{}
	context.completed = []*testTask{}

	// Fill tasks to be executed..
	context.createTasks()

	context.operationChannel = make(chan *testTask, 1)

	logrus.Infof("Starting test execution")
	for len(context.tasks) > 0 && len(context.running) > 0 {
		// WE take 1 test task from list and do execution.

		if len(context.tasks) > 0 {
			// Lets check if we have cluster required and start it
			// Check if we have cluster we could assign.

			for idx, task := range context.tasks {

				// Check if we have cluster available for running task.
				for _, ci := range task.cluster.instances {
					if ci.task == nil {
						// No task is assigned for cluster.
						switch ci.state {
						case clusterAdded, clusterCashed:
							context.startCluster(ci, task)
						case clusterReady:
							// We could assign task and start it running.
							ci.task = task
							// We need to remove task from list
							context.tasks = append(context.tasks[:idx], context.tasks[idx+1:]...)
							context.startTask(task, ci)
						}
					}
				}
			}
		}

		select {
		case jobDone := <-context.operationChannel:
			logrus.Infof("Tasks completed %v", jobDone)
		}
	}
	logrus.Infof("Completed tasks %v", len(context.completed))
}

func (context *executionContext) createTasks() {
	for _, test := range context.tests {
		for _, cluster := range context.clusters {
			if (len(test.ExecutionConfig.ClusterSelector) > 0 && utils.Contains(test.ExecutionConfig.ClusterSelector, cluster.config.Name)) ||
				len(test.ExecutionConfig.ClusterSelector) == 0 {
				// Cluster selector is defined we need to add tasks for individual cluster only
				context.tasks = append(context.tasks, &testTask{
					test:    test,
					cluster: cluster,
				})
			}
		}
	}
}

func (context *executionContext) startTask(task *testTask, instance *clusterInstance) {
	
}

func (context *executionContext) startCluster(ci *clusterInstance, task *testTask) {
	ci.state = clusterStarting
	ci.task = task
	go func() {
		err := ci.instance.Start(context.manager, time.Second*time.Duration(task.cluster.config.AverageStartTime*2))
		if err != nil {
			logrus.Infof("Failed to start cluster instance. Retrying...")
			ci.startFailed++
			ci.state = clusterCashed
		} else {
			ci.state = clusterReady
		}
		context.operationChannel <- task
	}()
}

func (context *executionContext) createClusters(testConfig config.CloudTestConfig) {
	context.clusters = []*clustersGroup{}
	clusterProviders := createClusterProviders(context.manager)

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
				instances := []*clusterInstance{}
				for i := 0; i < cl.Instances; i++ {
					cluster, err := provider.CreateCluster(&cl)
					if err != nil {
						logrus.Errorf("Failed to create cluster instance. Error %v", err)
						os.Exit(1)
					}
					instances = append(instances, &clusterInstance{
						instance: cluster,
						state: clusterAdded,
					})
				}
				context.clusters = append(context.clusters, &clustersGroup{
					provider:  provider,
					instances: instances,
					config:    &cl,
				})
			}
		}
	}
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
