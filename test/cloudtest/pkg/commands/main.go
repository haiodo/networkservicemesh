package commands

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	_ "github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers/shell"
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
	clusterAdded    clusterState = 0
	clusterStarting clusterState = 1
	clusterReady    clusterState = 2
	clusterBudy     clusterState = 3
	clusterCrashed  clusterState = 4
	clusterShutdown clusterState = 5
)

type clusterInstance struct {
	instance    providers.ClusterInstance
	state       clusterState
	startFailed int
	id          string
}
type clustersGroup struct {
	instances []*clusterInstance
	provider  providers.ClusterProvider
	config    *config.ClusterProviderConfig
}

type testTask struct {
	test            *TestEntry
	cluster         *clustersGroup
	clusterInstance *clusterInstance
}

type eventKind byte

const (
	eventTaskUpdate    eventKind = 0
	eventClusterUpdate eventKind = 1
)

type operationEvent struct {
	kind            eventKind
	cluster         *clustersGroup
	clusterInstance *clusterInstance
	task            *testTask
}

type executionContext struct {
	manager          execmanager.ExecutionManager
	clusters         []*clustersGroup
	operationChannel chan operationEvent
	tests            []*TestEntry
	tasks            []*testTask
	running          []*testTask
	completed        []*testTask
	cloudTestConfig  config.CloudTestConfig
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

	context := &executionContext{
	}

	err = yaml.Unmarshal(configFileContent, &context.cloudTestConfig)
	if err != nil {
		logrus.Errorf("Failed to parse configuration file: %v", err)
		return
	}
	logrus.Infof("Configuration file loaded successfully...")

	context.manager = execmanager.NewExecutionManager(context.cloudTestConfig.ConfigRoot)

	// Create cluster instance handles
	context.createClusters(context.cloudTestConfig)

	if len(context.clusters) == 0 {
		logrus.Errorf("There is no clusters defined. Exiting...")
		os.Exit(1)
	}

	// Collect tests
	logrus.Infof("Finding tests")
	context.tests = context.findTests()

	if len(context.tests) == 0 {
		logrus.Errorf("There is no tests defined. Exiting...")
	}

	context.tasks = []*testTask{}
	context.running = []*testTask{}
	context.completed = []*testTask{}

	// Fill tasks to be executed..
	context.createTasks()

	context.operationChannel = make(chan operationEvent)

	logrus.Infof("Starting test execution")
	for len(context.tasks) > 0 || len(context.running) > 0 {
		// WE take 1 test task from list and do execution.

		if len(context.tasks) > 0 {
			// Lets check if we have cluster required and start it
			// Check if we have cluster we could assign.
			newTasks := []*testTask{}
			for _, task := range context.tasks {
				store := true
				// Check if we have cluster available for running task.
				for _, ci := range task.cluster.instances {
					// No task is assigned for cluster.
					switch ci.state {
					case clusterAdded, clusterCrashed:
						// Try starting cluster
						context.startCluster(task.cluster, ci)
					case clusterReady:
						// We could assign task and start it running.
						task.clusterInstance = ci
						// We need to remove task from list
						context.running = append(context.running, task)
						context.startTask(task, ci)
						store = false
					}
				}
				if store {
					newTasks = append(newTasks, task)
				}
			}
			context.tasks = newTasks
		}

		select {
		case event := <-context.operationChannel:
			switch event.kind {
			case eventClusterUpdate:
				logrus.Infof("Instance for cluster %s is updated %v", event.cluster.config.Name, event.clusterInstance)
			case eventTaskUpdate:
				if event.task.test.Status == Status_SUCCESS || event.task.test.Status == Status_FAILED {
					context.completed = append(context.completed, event.task)
					event.task.clusterInstance.state = clusterReady
					logrus.Infof("Complete task %s on cluster %s, time %v", event.task.test.Name, event.task.clusterInstance.id, 0)
					for idx, t := range context.running {
						if t == event.task {
							context.running = append(context.running[:idx], context.running[idx+1:]...)
							break
						}
					}
				} else {
					// for timeout tasks, we need to check if
				}
			}
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
	task.test.Status = Status_SUCCESS
	task.test.Executions = append(task.test.Executions, TestEntryExecution{
		Status:     Status_SUCCESS,
		retry:      0,
		OutputFile: "-",
	})
	go func() {
		context.operationChannel <- operationEvent{
			kind: eventTaskUpdate,
			task: task,
		}
	}()
}

func (context *executionContext) startCluster(group *clustersGroup, ci *clusterInstance) {
	ci.state = clusterStarting
	go func() {
		err := ci.instance.Start(context.manager, time.Second*time.Duration(group.config.AverageStartTime*2))
		if err != nil {
			logrus.Infof("Failed to start cluster instance. Retrying...")
			ci.startFailed++
			ci.state = clusterCrashed
		} else {
			ci.state = clusterReady
		}
		context.operationChannel <- operationEvent{
			kind:            eventClusterUpdate,
			cluster:         group,
			clusterInstance: ci,
		}
		logrus.Infof("Cluster started...")
	}()
}

func (context *executionContext) createClusters(testConfig config.CloudTestConfig) {
	context.clusters = []*clustersGroup{}
	clusterProviders := createClusterProviders(context.manager)

	for _, cl := range context.cloudTestConfig.Providers {
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
						state:    clusterAdded,
						id:       fmt.Sprintf("%s-%d", cl.Name, i),
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

func (context *executionContext) findTests() []*TestEntry {
	var tests []*TestEntry
	for _, exec := range context.cloudTestConfig.Executions {
		execTests, err := GetTestConfiguration(context.manager, exec.PackageRoot, exec.Tags)
		if err != nil {
			logrus.Errorf("Failed during test lookup %v", err)
		}
		for _, t := range execTests {
			t.ExecutionConfig = &exec
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
