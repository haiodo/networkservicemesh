package shell

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestVariableSubstitutions(t *testing.T) {
	RegisterTestingT(t)

	env := map[string]string{
		"KUBECONFIG": "~/.kube/config",
	}

	args := map[string]string{
		"cluster-name": "idd",
		"provider-name": "name",
		"random": "r1",
		"uuid": "uu-uu",
		"tempdir": "/tmp",
		"zone-selector": "zone",
	}

	var1, err := substituteVariable("qwe ${KUBECONFIG} $(uuid) BBB", env, args)
	Expect(err).To(BeNil())
	Expect(var1).To(Equal("qwe ~/.kube/config uu-uu BBB"))

}
