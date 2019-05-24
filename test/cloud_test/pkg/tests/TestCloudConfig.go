package tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/cloud_test/pkg/config"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

func TestClusterConfiguration(t *testing.T) {
	RegisterTestingT(t)

	var testConfig config.CloudTestConfig

	file1, err := ioutil.ReadFile("./config1.yaml")
	Expect(err).To(BeNil())


	err = yaml.Unmarshal(file1, &testConfig)
	Expect(err).To(BeNil())
}