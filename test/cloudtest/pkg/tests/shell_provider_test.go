package tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/commands"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"testing"
)

type testValidationFactory struct {
}

type testValidator struct {
	location string
	config   *config.ClusterProviderConfig
}

func (v *testValidator) Validate() error {
	// Validation is passed for now
	return nil
}

func (*testValidationFactory) CreateValidator(config *config.ClusterProviderConfig, location string) (k8s.KubernetesValidator, error) {
	return &testValidator{
		config:   config,
		location: location,
	}, nil
}

func TestShellProvider(t *testing.T) {
	RegisterTestingT(t)

	testConfig := &config.CloudTestConfig{
	}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	testConfig.Providers = append(testConfig.Providers, &config.ClusterProviderConfig{
		Timeout:    30,
		Name:       "a_provider",
		NodeCount:  1,
		Kind:       "shell",
		RetryCount: 1,
		Instances:  2,
		Parameters: map[string]string{
			shell.ShellConfigScript:  "echo ./.tests/config",
			shell.ShellStartScript:   "echo started",
			shell.ShellPrepareScript: "echo prepared",
			shell.ShellInstallScript: "echo installed",
			shell.ShellStopScript:    "echo stopped",
		},
		Enabled: true,
	})

	testConfig.Providers = append(testConfig.Providers, &config.ClusterProviderConfig{
		Timeout:    100,
		Name:       "b_provider",
		NodeCount:  1,
		Kind:       "shell",
		RetryCount: 1,
		Instances:  2,
		Parameters: map[string]string{
			shell.ShellConfigScript:  "echo ./.tests/config",
			shell.ShellStartScript:   "echo started",
			shell.ShellPrepareScript: "echo prepared",
			shell.ShellInstallScript: "echo installed",
			shell.ShellStopScript:    "echo stopped",
		},
		Enabled: true,
	})

	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:        "simple_tagged",
		Timeout:     2,
		Tags:        []string{"basic"},
		PackageRoot: "./sample",
	})

	err, report := commands.PerformTesting(testConfig, &testValidationFactory{})
	Expect(err.Error()).To(Equal("There is failed tests 8"))

	Expect(report).NotTo(BeNil())

	Expect(len(report.Suites)).To(Equal(2))
	Expect(report.Suites[0].Failures).To(Equal(4))
	Expect(report.Suites[0].Tests).To(Equal(6))
	Expect(len(report.Suites[0].TestCases)).To(Equal(6))
	Expect(report.Suites[0].Failures).To(Equal(4))

	// Do assertions
}

func TestInvalidProvider(t *testing.T) {
	RegisterTestingT(t)

	testConfig := &config.CloudTestConfig{
	}

	testConfig.Timeout = 300

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cloud-test-temp")
	defer utils.ClearFolder(tmpDir, false)
	Expect(err).To(BeNil())

	testConfig.ConfigRoot = tmpDir
	testConfig.Providers = append(testConfig.Providers, &config.ClusterProviderConfig{
		Timeout:    30,
		Name:       "a_provider",
		NodeCount:  1,
		Kind:       "shell",
		RetryCount: 1,
		Instances:  2,
		Parameters: map[string]string{
			shell.ShellConfigScript:  "echo ./.tests/config",
			shell.ShellStartScript:   "echo started",
			shell.ShellPrepareScript: "echo prepared",
			shell.ShellInstallScript: "echo installed",
			shell.ShellStopScript:    "echo stopped",
		},
		Enabled: true,
	})

	testConfig.Executions = append(testConfig.Executions, &config.ExecutionConfig{
		Name:        "simple",
		Timeout:     2,
		PackageRoot: "./sample",
	})

	err, report := commands.PerformTesting(testConfig, &testValidationFactory{})
	Expect(err.Error()).To(Equal("There is failed tests 4"))

	Expect(report).NotTo(BeNil())

	Expect(len(report.Suites)).To(Equal(1))
	Expect(report.Suites[0].Failures).To(Equal(2))
	Expect(report.Suites[0].Tests).To(Equal(3))
	Expect(len(report.Suites[0].TestCases)).To(Equal(3))
	Expect(report.Suites[0].Failures).To(Equal(2))

	// Do assertions
}

