package shell

import (
	"bufio"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ShellInstallScript = "install" //#1
	ShellStartScript   = "start"   //#2
	ShellConfigScript  = "config"  //#3
	ShellPrepareScript = "prepare"
	ShellStopScript    = "stop"
	ShellKeepAlive     = "keep-alive"
	ShellZoneSelector  = "zone-selector"
)

type shellProvider struct {
	root    string
	indexes map[string]int
	sync.Mutex
	clusters    []shellInstance
	installDone map[string]bool
}

type shellInstance struct {
	root               string
	id                 string
	config             *config.ClusterProviderConfig
	processedEnv       []string
	started            bool
	startFailed        int
	keepAlive          bool
	configLocation     string
	configScript       string
	installScript      []string
	startScript        []string
	prepareScript      []string
	stopScript         []string
	zoneSelectorScript string
	provider           *shellProvider
	factory            k8s.ValidationFactory
	validator          k8s.KubernetesValidator
}

func (si *shellInstance) GetId() string {
	return si.id
}

func (si *shellInstance) CheckIsAlive() error {
	if si.started {
		return si.validator.Validate()
	}
	return fmt.Errorf("Cluster is not running")
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

func (si *shellInstance) Start(manager execmanager.ExecutionManager, timeout time.Duration, doInstallStep bool) error {
	logrus.Infof("Starting cluster %s-%s", si.config.Name, si.id)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Process environment variable values

	si.processedEnv = os.Environ()
	// Set seed
	rand.Seed(time.Now().UnixNano())

	utils.ClearFolder(si.root, true)

	// Do prepare
	if doInstallStep {
		if err := si.doInstall(manager, context); err != nil {
			return err
		}
	}

	selectedZone := ""

	if si.zoneSelectorScript != "" {
		zones, err := utils.ExecRead(context, strings.Split(si.zoneSelectorScript, " "))
		if err != nil {
			logrus.Errorf("Failed to select zones...")
			return err
		}
		selectedZone += zones[rand.Intn(len(zones)-1)]
	}

	for _, varName := range si.config.Env {
		finalVar := varName
		randValue := fmt.Sprintf("%v", rand.Intn(1000000))
		uuidValue := uuid.New().String()[:30]
		finalVar = strings.Replace(finalVar, "$(cluster-name)", si.id, -1)
		finalVar = strings.Replace(finalVar, "$(provider-name)", si.config.Name, -1)
		finalVar = strings.Replace(finalVar, "$(random)", randValue, -1)
		finalVar = strings.Replace(finalVar, "$(uuid)", uuidValue, -1)
		finalVar = strings.Replace(finalVar, "$(tempdir)", si.root, -1)
		finalVar = strings.Replace(finalVar, "$(zone-selector)", selectedZone, -1)

		p := "KUBECONFIG="
		if strings.HasPrefix(finalVar, p ) {
			si.configLocation=finalVar[len(p):]
		}

		si.processedEnv = append(si.processedEnv, finalVar)
	}

	// Run start script
	if err := si.runCmd(manager, context, "start", si.startScript, nil); err != nil {
		return err
	}

	if si.configLocation == "" {
		output, err := utils.ExecRead(context, strings.Split(si.configScript, " "))
		if err != nil {
			msg := fmt.Sprintf("Failed to retrieve configuration location %v", err)
			logrus.Errorf(msg)
			return err
		}
		si.configLocation = output[0]
	}
	var err error
	si.validator, err = si.factory.CreateValidator(si.config, si.configLocation)
	if err != nil {
		msg := fmt.Sprintf("Failed to start validator %v", err)
		logrus.Errorf(msg)
		return err
	}
	// Run prepare script
	if err := si.runCmd(manager, context, "prepare", si.prepareScript, []string{"KUBECONFIG=" + si.configLocation}); err != nil {
		return err
	}

	si.started = true

	return nil
}
func (si *shellInstance) Destroy(manager execmanager.ExecutionManager, timeout time.Duration) error {
	logrus.Infof("Destroying cluster  %s", si.id)

	context, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return si.runCmd(manager, context, "destroy", si.stopScript, nil)
}

func (si *shellInstance) GetRoot() string {
	return si.root
}

func (si *shellInstance) runCommand(context context.Context, cmd, fileName string, writer *bufio.Writer, env []string) error {
	cmdLine := strings.Split(cmd, " ")

	proc, err := utils.ExecProc(context, cmdLine, append(si.processedEnv, env...))
	if err != nil {
		return fmt.Errorf("Failed to run %s %v", cmdLine, err)
	}
	go func() {
		reader := bufio.NewReader(proc.Stdout)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			_, _ = writer.WriteString(s)
			_ = writer.Flush()
			logrus.Infof("Output: %s => %s %v", si.id, cmd, s)
		}
	}()
	go func() {
		reader := bufio.NewReader(proc.Stderr)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			_, _ = writer.WriteString(s)
			_ = writer.Flush()
			logrus.Infof("StdErr: %s => %s %v", si.id, cmd, s)
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

func (si *shellInstance) doInstall(manager execmanager.ExecutionManager, context context.Context) error {
	si.provider.Lock()
	defer si.provider.Unlock()
	if si.installScript != nil && !si.provider.installDone[si.config.Name] {
		si.provider.installDone[si.config.Name] = true
		return si.runCmd(manager, context, "install", si.installScript, nil)
	}
	return nil
}

func (si *shellInstance) runCmd(manager execmanager.ExecutionManager, context context.Context, operation string, script []string, env []string) error {
	fileName, fileRef, err := manager.OpenFile(si.id, operation)
	if err != nil {
		logrus.Errorf("Failed to %s system for testing of cluster %s %v", operation, si.config.Name, err)
		return err
	}

	defer fileRef.Close()

	writer := bufio.NewWriter(fileRef)

	for _, cmd := range script {
		if len(strings.TrimSpace(cmd)) == 0 {
			continue
		}
		_, _ = writer.WriteString(fmt.Sprintf("%s: %v\n ENV=%v\n", operation, cmd, strings.Join(si.processedEnv, "\n")))
		_ = writer.Flush()
		logrus.Infof("%s: %s => %s", operation, si.id, cmd)

		if err := si.runCommand(context, cmd, fileName, writer, env); err != nil {
			_, _ = writer.WriteString(fmt.Sprintf("Error running command: %v\n", err))
			_ = writer.Flush()
			return err
		}
	}
	return nil
}

func (p *shellProvider) getProviderId(provider string) string {
	val, ok := p.indexes[provider]
	if ok {
		val++
	} else {
		val = 1
	}
	p.indexes[provider] = val
	return fmt.Sprintf("%d", val)
}

func (p *shellProvider) CreateCluster(config *config.ClusterProviderConfig, factory k8s.ValidationFactory) (providers.ClusterInstance, error) {
	err := p.ValidateConfig(config)
	if err != nil {
		return nil, err
	}
	p.Lock()
	defer p.Unlock()
	id := fmt.Sprintf("%s-%s", config.Name, p.getProviderId(config.Name))

	clusterInstance := &shellInstance{
		provider:           p,
		root:               path.Join(p.root, id),
		id:                 id,
		config:             config,
		configScript:       config.Parameters[ShellConfigScript],
		installScript:      p.parseScript(config.Parameters[ShellInstallScript]),
		startScript:        p.parseScript(config.Parameters[ShellStartScript]),
		prepareScript:      p.parseScript(config.Parameters[ShellPrepareScript]),
		stopScript:         p.parseScript(config.Parameters[ShellStopScript]),
		zoneSelectorScript: config.Parameters[ShellZoneSelector],
		factory:            factory,
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
	utils.ClearFolder(root, true)
	return &shellProvider{
		root:        root,
		clusters:    []shellInstance{},
		indexes:     map[string]int{},
		installDone: map[string]bool{},
	}
}

func (p *shellProvider) ValidateConfig(config *config.ClusterProviderConfig) error {
	if _, ok := config.Parameters[ShellConfigScript]; !ok {
		hasKubeConfig := false
		for _, e := range config.Env {
			if strings.HasPrefix(e, "KUBECONFIG=") {
				hasKubeConfig = true
				break
			}
		}
		if !hasKubeConfig {
			return fmt.Errorf("Invalid config location")
		}
	}
	if _, ok := config.Parameters[ShellStartScript]; !ok {
		return fmt.Errorf("Invalid start script")
	}
	if _, ok := config.Parameters[ShellStopScript]; !ok {
		return fmt.Errorf("Invalid shutdown script location")
	}

	for _, envVar := range config.EnvCheck {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return fmt.Errorf("Environment variable are not specified %s Required variables: %v", envValue, config.EnvCheck)
		}
	}

	return nil
}

func (p *shellProvider) parseScript(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}
