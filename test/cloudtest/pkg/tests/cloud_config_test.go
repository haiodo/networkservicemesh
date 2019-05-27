package tests

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
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
	configStr := fmt.Sprintf("%v", testConfig)
	Expect(configStr).To(Equal("{1.0 [{GKE gke 1 5 5 0 3 false map[GCLOUD_SERVICE_KEY:$GCLOUD_SERVICE_KEY]} {Kind generic 1 5 5 0 3 false map[config:~/.kube/config start:make kind-start\nmake k8s-config\nmake k8s-load-images\n stop:make kind-stop]} {Vagrant generic 1 5 5 0 3 false map[config:./scripts/vagrant/.kube/config start:make vagrant-start\nmake k8s-config\nmake k8s-load-images\n stop:make kind-stop]}] ./.tests/cloud_test/ {./.tests/junit.xml} [{Single cluster tests [basic recovery usecase] ./test/integration 5 [] 1 [KUBECONFIG] []}] 0}"))

}