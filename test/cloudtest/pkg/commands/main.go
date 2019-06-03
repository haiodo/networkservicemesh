package commands

import (
	"bufio"
	"context"
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
	"sync"
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
	onlyEnabled    bool     // Disable all clusters and enable only enabled in command line.
	count          int      // Limit number of tests to be run per every cloud
	noStop         bool     // Disable stop operation
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
	clusterBusy     clusterState = 3
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
	cloudTestConfig  *config.CloudTestConfig
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

	ctx := &executionContext{
		cloudTestConfig: &config.CloudTestConfig{},
	}

	err = yaml.Unmarshal(configFileContent, &ctx.cloudTestConfig)
	if err != nil {
		logrus.Errorf("Failed to parse configuration file: %v", err)
		return
	}
	logrus.Infof("Configuration file loaded successfully...")

	ctx.manager = execmanager.NewExecutionManager(ctx.cloudTestConfig.ConfigRoot)

	// Create cluster instance handles
	ctx.createClusters()

	if len(ctx.clusters) == 0 {
		logrus.Errorf("There is no clusters defined. Exiting...")
		os.Exit(1)
	}

	// Collect tests
	logrus.Infof("Finding tests")
	ctx.tests = ctx.findTests()

	if len(ctx.tests) == 0 {
		logrus.Errorf("There is no tests defined. Exiting...")
	}

	ctx.tasks = []*testTask{}
	ctx.running = []*testTask{}
	ctx.completed = []*testTask{}

	// Fill tasks to be executed..
	ctx.createTasks()

	ctx.operationChannel = make(chan operationEvent)

	logrus.Infof("Starting test execution")
	for len(ctx.tasks) > 0 || len(ctx.running) > 0 {
		// WE take 1 test task from list and do execution.

		if len(ctx.tasks) > 0 {
			// Lets check if we have cluster required and start it
			// Check if we have cluster we could assign.
			newTasks := []*testTask{}
			for _, task := range ctx.tasks {
				store := true
				// Check if we have cluster available for running task.
				for _, ci := range task.cluster.instances {
					// No task is assigned for cluster.
					switch ci.state {
					case clusterAdded, clusterCrashed:
						// Try starting cluster
						ctx.startCluster(task.cluster, ci)
					case clusterReady:
						// We could assign task and start it running.
						task.clusterInstance = ci
						// We need to remove task from list
						ctx.running = append(ctx.running, task)
						ctx.startTask(task, []*clusterInstance{ci})
						store = false
					}
				}
				if store {
					newTasks = append(newTasks, task)
				}
			}
			ctx.tasks = newTasks
		}

		select {
		case event := <-ctx.operationChannel:
			switch event.kind {
			case eventClusterUpdate:
				logrus.Infof("Instance for cluster %s is updated %v", event.cluster.config.Name, event.clusterInstance)
			case eventTaskUpdate:
				if event.task.test.Status == Status_SUCCESS || event.task.test.Status == Status_FAILED {
					ctx.completed = append(ctx.completed, event.task)
					event.task.clusterInstance.state = clusterReady
					logrus.Infof("Complete task %s on cluster %s, time %v", event.task.test.Name, event.task.clusterInstance.id, 0)
					for idx, t := range ctx.running {
						if t == event.task {
							ctx.running = append(ctx.running[:idx], ctx.running[idx+1:]...)
							break
						}
					}
				} else {
					// for timeout tasks, we need to check if
				}
			}
		}
	}
	logrus.Infof("Completed tasks %v", len(ctx.completed))

	// We need to stop all clusters we started
	if !cmdArguments.noStop {
		var wg sync.WaitGroup
		for _, cl := range ctx.clusters {
			for _, inst := range cl.instances {
				wg.Add(1)

				go func() {
					defer wg.Done()
					logrus.Infof("Closing cluster %v %v", cl.config.Name, inst.id)
					err := inst.instance.Destroy(ctx.manager, 30*time.Minute)
					if err != nil {
						logrus.Error("Failed to close cluster %v", inst.id)
					}
				}()
			}
		}
		wg.Wait()
	}
}

func (ctx *executionContext) createTasks() {
	for i, test := range ctx.tests {
		if cmdArguments.count > 0 && i >= cmdArguments.count {
			logrus.Infof("Limit of tests for execution:: %v is reached", cmdArguments.count)
			break
		}
		for _, cluster := range ctx.clusters {
			if (len(test.ExecutionConfig.ClusterSelector) > 0 && utils.Contains(test.ExecutionConfig.ClusterSelector, cluster.config.Name)) ||
				len(test.ExecutionConfig.ClusterSelector) == 0 {
				// Cluster selector is defined we need to add tasks for individual cluster only
				ctx.tasks = append(ctx.tasks, &testTask{
					test:    test,
					cluster: cluster,
				})
			}
		}
	}
}

func (ctx *executionContext) startTask(task *testTask, instances []*clusterInstance) error {
	ids := ""
	for _, ci := range instances {
		if len(ids) > 0 {
			ids += "_"
		}
		ids += ci.id

		ci.state = clusterBusy
	}
	fileName, file, err := ctx.manager.OpenFileTest("test", ids, task.test.Name, "run")
	if err != nil {
		return err
	}

	clusterConfigs := []string{}

	for _, inst := range instances {
		clusterConfig, err := inst.instance.GetClusterConfig()
		if err != nil {
			return err
		}
		clusterConfigs = append(clusterConfigs, clusterConfig)
	}

	go func() {
		cmdLine := []string{
			"go", "test",
			task.test.ExecutionConfig.PackageRoot,
			"-test.timeout", "10m",
			"-count", "1",
			"--run", task.test.Name,
			"--tags", task.test.Tags,
			"--test.v", task.test.Tags,
		}

		env := []string{
		}
		// Fill Kubernetes environment variables.

		for ind, envV := range task.test.ExecutionConfig.KubernetesEnv {
			env = append(env, fmt.Sprintf("%s=%s", envV, clusterConfigs[ind]))
		}

		writer := bufio.NewWriter(file)
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		logrus.Infof(fmt.Sprintf("Running test %s on cluster's %v \n", task.test.Name, ids))

		_, _ = writer.WriteString(fmt.Sprintf("Running test %s on cluster's %v \n", task.test.Name, ids))

		proc, error := utils.ExecProc(timeoutCtx, cmdLine, env)
		if error != nil {
			logrus.Errorf("Failed to run %s %v", cmdLine)
			ctx.updateTestExecution(task, fileName, Status_FAILED)
		}
		go func() {
			reader := bufio.NewReader(proc.Stdout)
			for {
				s, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				_, _ = writer.WriteString(s)
				writer.Flush()
			}
		}()
		if code := proc.ExitCode(); code != 0 {
			logrus.Errorf("Failed to run %s %v. Logs inside %v", cmdLine, code, fileName)
			_, _ = writer.WriteString(fmt.Sprintf("Exit code %v \n", code))
			writer.Flush()
			ctx.updateTestExecution(task, fileName, Status_FAILED)
		}

		ctx.operationChannel <- operationEvent{
			kind: eventTaskUpdate,
			task: task,
		}
	}()
	return nil
}

func (ctx *executionContext) updateTestExecution(task *testTask, fileName string, status Status) {
	task.test.Status = status
	task.test.Executions = append(task.test.Executions, TestEntryExecution{
		Status:     status,
		retry:      len(task.test.Executions) + 1,
		OutputFile: fileName,
	})
}

func (ctx *executionContext) startCluster(group *clustersGroup, ci *clusterInstance) {
	ci.state = clusterStarting
	go func() {
		err := ci.instance.Start(ctx.manager, time.Minute*30)
		if err != nil {
			logrus.Infof("Failed to start cluster instance. Retrying...")
			ci.startFailed++
			ci.state = clusterCrashed
			err := ci.instance.Destroy(ctx.manager, time.Minute*30)
			logrus.Errorf("Failed to stop cluster %v", err)
		} else {
			ci.state = clusterReady
		}
		ctx.operationChannel <- operationEvent{
			kind:            eventClusterUpdate,
			cluster:         group,
			clusterInstance: ci,
		}
		logrus.Infof("Cluster started...")
	}()
}

func (ctx *executionContext) createClusters() {
	ctx.clusters = []*clustersGroup{}
	clusterProviders := createClusterProviders(ctx.manager)

	for _, cl := range ctx.cloudTestConfig.Providers {
		if cmdArguments.onlyEnabled {
			logrus.Infof("Disable cluster config:: %v since onlyEnabled is passed...", cl.Name)
			cl.Enabled = false
		}
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
					cluster, err := provider.CreateCluster(cl)
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
				ctx.clusters = append(ctx.clusters, &clustersGroup{
					provider:  provider,
					instances: instances,
					config:    cl,
				})
			}
		}
	}
}

func (ctx *executionContext) findTests() []*TestEntry {
	var tests []*TestEntry
	for _, exec := range ctx.cloudTestConfig.Executions {
		execTests, err := GetTestConfiguration(ctx.manager, exec.PackageRoot, exec.Tags)
		if err != nil {
			logrus.Errorf("Failed during test lookup %v", err)
		}
		for _, t := range execTests {
			t.ExecutionConfig = exec
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
	rootCmd.Flags().BoolVarP(&cmdArguments.onlyEnabled, "enabled", "e", false, "Use only passed cluster names...")
	rootCmd.Flags().IntVarP(&cmdArguments.count, "count", "", -1, "Execute only count of tests")

	rootCmd.Flags().BoolVarP(&cmdArguments.noStop, "noStop", "", false, "Pass to disable stop operations...")

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
