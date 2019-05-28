package commands

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
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
		}
	}

	logrus.Infof("Finding tests")

	// Collect executions and associate clusters
	var tests []*TestEntry
	for _, exec := range cloudTestConfig.Executions {
		execTests, err := GetTestConfiguration(manager, exec.PackageRoot, exec.Tags)
		if err != nil {
			logrus.Errorf("Failed during test lookup %v", err)
		}
		tests = append(tests, execTests...)
	}

	logrus.Infof("Total tests found: %v", len(tests))
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
