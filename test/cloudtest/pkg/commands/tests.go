package commands

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Status int8
const (
	// Success execution on all clusters
	Status_SUCCESS Status = 0
	// Failed execution on all clusters
	Status_FAILED Status = 1
)

type TestEntry struct {
	Name string		// Test name
	Tags string	// A list of tags
	ExecutionConfig *config.ExecutionConfig
	Execution []struct {
		Cluster *providers.ClusterInstance
		OutputFile string	// Output file name
		retry int	// Did we retry execution on this cluster.
		Status Status	// Execution status
		Output string	// Output file name
	}
}

// Return list of available tests by calling of gotest --list .* $root -tag "" and parsing of output.
func GetTestConfiguration(manager execmanager.ExecutionManager, root string, tags []string) ([]*TestEntry, error) {
	gotestCmd := []string{"go", "test", root, "--list", ".*"}
	if len(tags) > 0 {
		result := []*TestEntry{}
		for _, tag := range tags {
			gotestCmd = append(gotestCmd, "-tags", tag)
			tests, err := getTests(manager, gotestCmd, tag)
			if err != nil {
				return nil, err
			}
			result = append( result, tests... )
		}
		return result, nil
	} else {
		return getTests(manager, gotestCmd, "")
	}
}

func getTests(manager execmanager.ExecutionManager, gotestCmd []string, tag string) ([]*TestEntry, error) {
	st := time.Now()
	result, err := utils.ExecRead(context.Background(), gotestCmd )
	if err != nil {
		logrus.Errorf("Error getting list of tests %v", err)
	}

	var testResult []*TestEntry

	manager.AddLog("gotest", "find-tests", strings.Join(result, "\n"))
	for _, testLine := range result {
		if strings.ContainsAny(testLine, "\t") {
			special := strings.Split(testLine, "\t")
			if len(special) == 3 {
				// This is special case.
			}
		} else {
			testResult = append( testResult, &TestEntry{
				Name: strings.TrimSpace(testLine),
				Tags: tag,
			})
		}
	}

	logrus.Infof("Tests found: %v Elapsed: %v", len(result), time.Since(st))
	return testResult, nil
}
