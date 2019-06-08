package shell

import (
	"bufio"
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ShellConfigScript      = "config"
	ShellStartScript       = "start"
	ShellStartConfigScript = "start-config"
	ShellStopScript        = "stop"
	ShellKeepAlive         = "keep-alive"
)

type shellProvider struct {
	root  string
	index int
	sync.Mutex
	clusters []shellInstance
}

type shellInstance struct {
	root              string
	id                string
	config            *config.ClusterProviderConfig
	started           bool
	startFailed       int
	keepAlive         bool
	configLocation    string
	configScript      string
	startScript       []string
	startConfigScript []string
	stopScript        []string
	utils             *k8s.K8sUtils
}

func (si *shellInstance) CheckIsAlive() ([]v1.Node, error) {
	if si.started {
		nodes, err := si.utils.GetNodes()
		if err != nil {
			return nodes, err
		}
		if len(nodes) < si.config.NodeCount {
			msg := fmt.Sprintf("Cluster lost some of its nodes: %v required nodes: %v", nodes, si.config.NodeCount)
			logrus.Error(msg)
			return nodes, fmt.Errorf(msg)
		}
		return nodes, err
	}
	return nil, fmt.Errorf("Cluster is not running")
}

func (si *shellInstance) IsRunning() bool {
	return si.started
}

func (si *shellInstance) GetClusterConfig() (string, error) {
	if si.started {
		return si.configLocation, nil
	}
	return "", fmt.Errorf("Cluster is not started yet...")
}

func (si *shellInstance) Start(manager execmanager.ExecutionManager, timeout time.Duration) error {
	fileName, file, err := manager.OpenFile("cluster_start", fmt.Sprintf("starting_%s", si.id))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	logrus.Infof("Starting cluster %s-%s logfile: %v", si.config.Name, si.id, fileName)

	writer := bufio.NewWriter(file)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, _ = writer.WriteString(fmt.Sprintf("Starting cluster %s\n with configuration %v\n", si.id, si.config))
	_ = writer.Flush()

	for _, cmd := range si.startScript {
		if len(strings.TrimSpace(cmd)) == 0 {
			continue
		}
		_, _ = writer.WriteString(fmt.Sprintf("Running: %v\n", cmd))
		_ = writer.Flush()
		logrus.Infof("Running: %s => %s", si.id, cmd)

		if err := si.runCommand(context, cmd, fileName, writer); err != nil {
			_, _ = writer.WriteString(fmt.Sprintf("Error running command: %v\n", err))
			_ = writer.Flush()
			return err
		}
	}

	_, _ = writer.WriteString("Retrieving configuration location\n")
	_ = writer.Flush()
	output, err := utils.ExecRead(context, strings.Split(si.configScript, " "))
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve configuration location %v", err)
		_, _ = writer.WriteString(msg)
		logrus.Errorf(msg)
		return err
	}
	si.configLocation = output[0]

	_, _ = writer.WriteString(strings.Join(output, "\n"))

	defer func() {
		_ = file.Close()
	}()

	_, _ = writer.WriteString("Constructing K8s client API to connect to cluster.\n")

	si.utils, err = k8s.NewK8sUtils(si.configLocation)
	if err != nil {
		return err
	}

	requiedNodes := si.config.NodeCount
	for {
		nodes, err := si.utils.GetNodes()
		if err != nil {
			return err
		}
		if len(nodes) >= requiedNodes {
			_, _ = writer.WriteString(fmt.Sprintf("Cluster started properly with nodes: %v\n", nodes))
			break;
		}
		msg := fmt.Sprintf("Cluster %s doesn't have required number of nodes to be available. Required: %v Available: %v\n", si.id, requiedNodes, len(nodes))
		logrus.Errorf(msg)
		err = fmt.Errorf(msg)
		return err
	}

	// Running start config script
	for _, cmd := range si.startConfigScript {
		if len(strings.TrimSpace(cmd)) == 0 {
			continue
		}
		_, _ = writer.WriteString(fmt.Sprintf("Running: %v\n", cmd))
		_ = writer.Flush()
		logrus.Infof("Running: %s => %s", si.id, cmd)

		si.config.Env = append(si.config.Env, "KUBECONFIG="+si.configLocation)

		if err := si.runCommand(context, cmd, fileName, writer); err != nil {
			_, _ = writer.WriteString(fmt.Sprintf("Error running command: %v\n", err))
			_ = writer.Flush()
			return err
		}
	}

	si.started = true
	return nil
}
func (si *shellInstance) Destroy(manager execmanager.ExecutionManager, timeout time.Duration) error {
	fileName, file, err := manager.OpenFile("cluster_stop", fmt.Sprintf("stopping_%s", si.id))
	if err != nil {
		return err
	}

	logrus.Infof("Destroying cluster  %s-%s logfile: %v", si.config.Name, si.id, fileName)

	writer := bufio.NewWriter(file)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, _ = writer.WriteString(fmt.Sprintf("Stopping cluster %s with configuration %v \n", si.id, si.config))

	for _, cmd := range si.stopScript {
		_, _ = writer.WriteString(fmt.Sprintf("Running: %v\n", cmd))
		logrus.Infof("Running: %s => %s", si.id, cmd)

		if err := si.runCommand(context, cmd, fileName, writer); err != nil {
			return err
		}
	}
	defer func() {
		_ = file.Close()
	}()

	return nil
}

func (si *shellInstance) GetRoot() string {
	return si.root
}

func (si *shellInstance) runCommand(context context.Context, cmd, fileName string, writer *bufio.Writer) error {
	cmdLine := strings.Split(cmd, " ")

	proc, error := utils.ExecProc(context, cmdLine, si.config.Env)
	if error != nil {
		return fmt.Errorf("Failed to run %s %v", cmdLine)
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
			logrus.Infof("Output: %s => %s %v", si.id, cmd, s)
		}
	}()
	if code := proc.ExitCode(); code != 0 {
		logrus.Errorf("Failed to run %s ExitCode: %v. Logs inside %v", cmdLine, code, fileName)
		return fmt.Errorf("Failed to run %s ExitCode: %v. Logs inside %v", cmdLine, code, fileName)
	}
	return nil
}

func (si *shellInstance) doDestroy(writer *bufio.Writer, manager execmanager.ExecutionManager, timeout time.Duration, err error) {
	_, _ = writer.WriteString(fmt.Sprintf("Error during k8s API initialisation %v", err))
	_, _ = writer.WriteString(fmt.Sprintf("Trying to destroy cluster"))
	// In case we failed to start and create cluster utils.
	err2 := si.Destroy(manager, timeout)
	if err2 != nil {
		_, _ = writer.WriteString(fmt.Sprintf("Error during destroy of cluster %v", err2))
	}
}

func (p *shellProvider) CreateCluster(config *config.ClusterProviderConfig) (providers.ClusterInstance, error) {
	err := p.ValidateConfig(config)
	if err != nil {
		return nil, err
	}
	p.Lock()
	defer p.Unlock()
	p.index++
	id := fmt.Sprintf("cluster-%d", p.index)

	clusterInstance := &shellInstance{
		root:              path.Join(p.root, id),
		id:                id,
		config:            config,
		configScript:      config.Parameters[ShellConfigScript],
		startScript:       p.parseScript(config.Parameters[ShellStartScript]),
		startConfigScript: p.parseScript(config.Parameters[ShellStartConfigScript]),
		stopScript:        p.parseScript(config.Parameters[ShellStopScript]),
	}

	if value, err := strconv.ParseBool(config.Parameters[ShellKeepAlive]); err == nil {
		clusterInstance.keepAlive = value
	}

	return clusterInstance, nil
}

func init() {
	logrus.Infof("Adding shell as supported providers...")
	providers.ClusterProviderFactories["shell"] = NewShellClusterProvider
}

func NewShellClusterProvider(root string) providers.ClusterProvider {
	utils.ClearFolder(root)
	return &shellProvider{
		root:     root,
		clusters: []shellInstance{},
		index:    0,
	}
}

func (p *shellProvider) ValidateConfig(config *config.ClusterProviderConfig) error {
	if _, ok := config.Parameters[ShellConfigScript]; !ok {
		return fmt.Errorf("Invalid config location")
	}
	if _, ok := config.Parameters[ShellStartScript]; !ok {
		return fmt.Errorf("Invalid start script")
	}
	if _, ok := config.Parameters[ShellStopScript]; !ok {
		return fmt.Errorf("Invalid shutdown script location")
	}
	return nil
}

func (p *shellProvider) parseScript(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}
