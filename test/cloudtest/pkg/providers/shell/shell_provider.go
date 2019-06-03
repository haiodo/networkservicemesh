package shell

import (
	"bufio"
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ShellConfigScript = "config"
	ShellStartScript  = "start"
	ShellStopScript   = "stop"
	ShellKeepAlive    = "keep-alive"
)

type shellProvider struct {
	root  string
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
	configScript   string
	startScript    []string
	stopScript     []string
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

	logrus.Infof("Starting cluster %s-%s logfile: %v", si.config.Name, si.id, fileName)

	writer := bufio.NewWriter(file)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, _ = writer.WriteString(fmt.Sprintf("Starting cluster %s\n with configuration %v\n", si.id, si.config))
	writer.Flush()

	for _, cmd := range si.startScript {
		if len(strings.TrimSpace(cmd)) == 0 {
			continue
		}
		_, _ = writer.WriteString(fmt.Sprintf("Running: %v\n", cmd))
		writer.Flush()
		logrus.Infof("Running: %s => %s", si.id, cmd)

		if err := si.runCommand(context, cmd, fileName, writer); err != nil {
			_, _ = writer.WriteString(fmt.Sprintf("Error running command: %v\n", err))
			writer.Flush()
			return err
		}
	}


	output, err := utils.ExecRead(context, strings.Split(si.configScript, " "))
	if err != nil {
		return err
	}
	si.configLocation = output[0]

	si.started = true

	defer func() {
		_ = file.Close()
	}()

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
		root:         path.Join(p.root, id),
		id:           id,
		config:       config,
		configScript: config.Parameters[ShellConfigScript],
		startScript:  p.parseScript(config.Parameters[ShellStartScript]),
		stopScript:   p.parseScript(config.Parameters[ShellStopScript]),
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
