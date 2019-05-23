package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "cloud_test",
	Short: "NSM Cloud Test is cloud helper continuous integration testing tool",
	Long:  `Allow to execute all set of individual tests across all clouds provided.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		fmt.Printf("Some stuff")
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
	junitXml string
}

var testArguments *TestArguments = &TestArguments{}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&testArguments.providerConfig, "providers", "p", "", "Config file for providers")
	rootCmd.MarkFlagRequired("providers")
}

func initConfig() {
}
