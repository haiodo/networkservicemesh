package commands

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var rootCmd = &cobra.Command{
	Use:   "cloud_test",
	Short: "NSM Cloud Test is cloud helper continuous integration testing tool",
	Long:  `Allow to execute all set of individual tests across all clouds provided.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		findTests(args)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a test folder argument")
		}
		for _, arg := range args {
			if _, err := os.Stat(arg); os.IsNotExist(err) {
				return fmt.Errorf("Test folder must exist...")
			}
		}
		return nil
	},

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}



type TestArguments struct {
	providerConfig string // A folder to start scaning for tests inside.
	junitXml string	// File to publish JUnti xml like report into
	clusterTags string		// A list of tags to pass to do integration testing with one cluster.
	multiTags string 	// tags for tests with multi cluster values.
}

var testArguments *TestArguments = &TestArguments{}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&testArguments.providerConfig, "config", "c", "", "Config file for providers, default .cloudtest.yaml")
}

func initConfig() {
}


