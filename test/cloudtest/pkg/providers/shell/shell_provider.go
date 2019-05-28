package shell

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ShellConfigLocation = "config"
	ShellStartScript = "start"
	ShellShutdownScript = "shutdown"
	ShellKeepAlive	= "keep-alive"
)

type shellProvider struct {
	root string
	index int
	sync.Mutex
	clusters []shellInstance
}

type shellInstance struct {
	root           string
	id             string
	config         *config.ClusterProviderConfig
	started        bool
	startFailed    int
	keepAlive      bool
	configLocation string
	startScript    []string
	shutdownScript []string
}

func (si *shellInstance) GetClusterConfig() (string, error) {
	if si.started {
		return si.configLocation, nil
	}
	return "", fmt.Errorf("Cluster is not started yet...")
}

func (si *shellInstance) Start(timeout time.Time) error {
	panic("implement me")
}
func (si *shellInstance) Destroy(timeout time.Time) error {
	panic("implement me")
}

func (*shellInstance) GetRoot() string {
	panic("implement me")
}

func (p *shellProvider) CreateCluster(config *config.ClusterProviderConfig) (providers.ClusterInstance, error) {
	err := p.ValidateConfig(config)
	if err != nil {
		return nil, err
	}
	p.Lock()
	defer p.Unlock()
	p.index++
	id := fmt.Sprint("cluster-%d", p.index)

	clusterInstance := &shellInstance {
		root: path.Join(p.root, id ),
		id: id,
		config: config,
		configLocation: config.Parameters[ShellConfigLocation],
		startScript: p.parseScript(config.Parameters[ShellStartScript]),
		shutdownScript: p.parseScript(config.Parameters[ShellShutdownScript]),
	}

	if value, err := strconv.ParseBool(config.Parameters[ShellKeepAlive]); err == nil {
		clusterInstance.keepAlive = value
	}

	return clusterInstance, nil
}

func init() {
	providers.ClusterProviderFactories["shell"] = NewShellClusterProvider
}

func NewShellClusterProvider(root string) providers.ClusterProvider {
	utils.ClearFolder(root)
	return &shellProvider{
		root: root,
		clusters: []shellInstance{},
		index: 0,
	}
}

func (p *shellProvider) ValidateConfig(config *config.ClusterProviderConfig) error {
	if _, ok := config.Parameters[ShellConfigLocation]; !ok {
		return fmt.Errorf("Invalid config location")
	}
	if _, ok := config.Parameters[ShellStartScript]; !ok {
		return fmt.Errorf("Invalid start script")
	}
	if _, ok := config.Parameters[ShellShutdownScript]; !ok {
		return fmt.Errorf("Invalid shutdown script location")
	}
	return nil
}

func (p *shellProvider) parseScript(s string) []string {
	return strings.Split(s, "\n")
}



