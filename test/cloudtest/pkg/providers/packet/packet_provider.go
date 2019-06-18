package packet

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/execmanager"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/k8s"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/providers"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/shell"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"
	"github.com/packethost/packngo"
	"github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	installScript   = "install" //#1
	setupScript     = "setup"   //#2
	startScript     = "start"   //#4
	configScript    = "config"  //#5
	prepareScript   = "prepare" //#6
	stopScript      = "stop"    // #7
	packetProjectID = "PACKET_PROJECT_ID"
)

type packetProvider struct {
	root    string
	indexes map[string]int
	sync.Mutex
	clusters    []packetInstance
	installDone map[string]bool
}

type packetInstance struct {
	installScript  []string
	setupScript    []string
	startScript    []string
	prepareScript  []string
	stopScript     []string
	manager        execmanager.ExecutionManager
	root           string
	id             string
	configScript   string
	factory        k8s.ValidationFactory
	validator      k8s.KubernetesValidator
	configLocation string
	shellInterface shell.Manager
	projectID      string
	packetAuthKey  string
	genID          string
	keyID          string
	config         *config.ClusterProviderConfig
	provider       *packetProvider
	client         *packngo.Client
	project        *packngo.Project
	devices        map[string]*packngo.Device
	sshKey         *packngo.SSHKey
	params         providers.InstanceOptions
	started        bool
	keyIds         []string
	facilitiesList []string
}

func (pi *packetInstance) GetID() string {
	return pi.id
}

func (pi *packetInstance) CheckIsAlive() error {
	if pi.started {
		return pi.validator.Validate()
	}
	return fmt.Errorf("cluster is not running")
}

func (pi *packetInstance) IsRunning() bool {
	return pi.started
}

func (pi *packetInstance) GetClusterConfig() (string, error) {
	if pi.started {
		return pi.configLocation, nil
	}
	return "", fmt.Errorf("cluster is not started yet")
}

func (pi *packetInstance) Start(timeout time.Duration) error {
	logrus.Infof("Starting cluster %s-%s", pi.config.Name, pi.id)
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set seed
	rand.Seed(time.Now().UnixNano())

	utils.ClearFolder(pi.root, true)

	pi.genID = uuid.New().String()[:30]
	// Process and prepare environment variables
	if err = pi.shellInterface.ProcessEnvironment(map[string]string{"cluster-uuid": pi.genID}); err != nil {
		logrus.Errorf("error during preocessing enviornment variables %v", err)
		return err
	}

	// Do prepare
	if !pi.params.NoInstall {
		if err = pi.doInstall(ctx); err != nil {
			return err
		}
	}

	// Run start script
	if err = pi.shellInterface.RunCmd(ctx, "setup", pi.setupScript, nil); err != nil {
		return err
	}

	keyFile := pi.config.Packet.SshKey
	if !utils.FileExists(keyFile) {
		// Relative file
		keyFile = path.Join(pi.root, keyFile)
		if !utils.FileExists(keyFile) {
			err = fmt.Errorf("failed to locate generated key file, please specify init script to generate it")
			logrus.Errorf(err.Error())
			return err
		}
	}

	if pi.client, err = packngo.NewClient(); err != nil {
		logrus.Errorf("failed to create Packet REST interface")
		return err
	}

	if err = pi.updateProject(); err != nil {
		return err
	}

	// Check and add key if it is not yet added.

	if pi.keyIds, err = pi.createKey(keyFile); err != nil {
		return err
	}

	if pi.facilitiesList, err = pi.findFacilities(); err != nil {
		return err
	}
	for _, devCfg := range pi.config.Packet.Devices {
		var device *packngo.Device
		if device, err = pi.createDevice(devCfg); err != nil {
			return err
		}
		pi.devices[devCfg.Name] = device
	}

	// All devices are created so we need to wait for them to get alive.
	if err = pi.waitDevicesStartup(ctx); err != nil {
		return err
	}
	// We need to add arguments

	pi.addDeviceContextArguments()

	printableEnv := pi.shellInterface.PrintEnv(pi.shellInterface.GetProcessedEnv())
	pi.manager.AddLog(pi.id, "environment", printableEnv)

	// Run start script
	if err = pi.shellInterface.RunCmd(ctx, "start", pi.startScript, nil); err != nil {
		return err
	}

	if err = pi.updateKUBEConfig(ctx); err != nil {
		return err
	}

	if pi.validator, err = pi.factory.CreateValidator(pi.config, pi.configLocation); err != nil {
		msg := fmt.Sprintf("Failed to start validator %v", err)
		logrus.Errorf(msg)
		return err
	}
	// Run prepare script
	if err = pi.shellInterface.RunCmd(ctx, "prepare", pi.prepareScript, []string{"KUBECONFIG=" + pi.configLocation}); err != nil {
		return err
	}

	pi.started = true

	return nil
}

func (pi *packetInstance) updateKUBEConfig(context context.Context) error {
	if pi.configLocation == "" {
		pi.configLocation = pi.shellInterface.GetConfigLocation()
	}
	if pi.configLocation == "" {
		output, err := utils.ExecRead(context, strings.Split(pi.configScript, " "))
		if err != nil {
			err = fmt.Errorf("failed to retrieve configuration location %v", err)
			logrus.Errorf(err.Error())
		}
		pi.configLocation = output[0]
	}
	return nil
}

func (pi *packetInstance) addDeviceContextArguments() {
	for key, dev := range pi.devices {
		for _, n := range dev.Network {
			pub := "pub"
			if !n.Public {
				pub = "private"
			}
			pi.shellInterface.AddExtraArgs(fmt.Sprintf("device.%v.%v.%v.%v", key, pub, "ip", n.AddressFamily), n.Address)
			pi.shellInterface.AddExtraArgs(fmt.Sprintf("device.%v.%v.%v.%v", key, pub, "gw", n.AddressFamily), n.Gateway)
			pi.shellInterface.AddExtraArgs(fmt.Sprintf("device.%v.%v.%v.%v", key, pub, "net", n.AddressFamily), n.Network)
		}
	}
}

func (pi *packetInstance) waitDevicesStartup(context context.Context) error {
	for {
		alive := map[string]*packngo.Device{}
		for key, d := range pi.devices {
			var updatedDevice *packngo.Device
			updatedDevice, _, err := pi.client.Devices.Get(d.ID, &packngo.GetOptions{})
			if err != nil {
				logrus.Errorf("Error %v", err)
			} else if updatedDevice.State == "active" {
				alive[key] = updatedDevice
			}
		}
		if len(alive) == len(pi.devices) {
			pi.devices = alive
			break
		}
		select {
		case <-time.After(100 * time.Millisecond):
			continue
		case <-context.Done():
			return fmt.Errorf("timeout %v", context.Err())
		}
	}
	return nil
}

func (pi *packetInstance) createDevice(devCfg *config.DeviceConfig) (*packngo.Device, error) {
	devReq := &packngo.DeviceCreateRequest{
		//Facility:
		Plan:           devCfg.Plan,
		Facility:       pi.facilitiesList,
		Hostname:       devCfg.Name + "-" + pi.genID,
		BillingCycle:   devCfg.BillingCycle,
		OS:             devCfg.OperatingSystem,
		ProjectID:      pi.projectID,
		ProjectSSHKeys: pi.keyIds,
	}
	var device *packngo.Device
	var response *packngo.Response
	device, response, err := pi.client.Devices.Create(devReq)
	pi.manager.AddLog(pi.id, fmt.Sprintf("create-device-%s", devCfg.Name), fmt.Sprintf("%v", response))
	return device, err
}

func (pi *packetInstance) findFacilities() ([]string, error) {
	facilities, response, err := pi.client.Facilities.List(&packngo.ListOptions{})

	pi.manager.AddLog(pi.id, "list-facilities", response.String())

	if err != nil {
		return nil, err
	}

	facilitiesList := []string{}
	for _, f := range facilities {
		facilityReqs := map[string]string{}
		for _, ff := range f.Features {
			facilityReqs[ff] = ff
		}

		found := true
		for _, ff := range pi.config.Packet.Facilities {
			if _, ok := facilityReqs[ff]; !ok {
				found = false
				break
			}
		}
		if found {
			facilitiesList = append(facilitiesList, f.Code)
		}
	}
	logrus.Infof("List of facilities: %v %v", facilities, response)

	// Randomize facilities.

	ind := -1

	if pi.config.Packet.PreferredFacility != "" {
		for i, f := range facilitiesList {
			if f == pi.config.Packet.PreferredFacility {
				ind = i
				break
			}
		}
	}

	if ind != -1 {
		selected := facilitiesList[ind]

		facilitiesList[ind] = facilitiesList[0]
		facilitiesList[0] = selected
	}

	return facilitiesList, nil
}

func (pi *packetInstance) Destroy(timeout time.Duration) error {
	logrus.Infof("Destroying cluster  %s", pi.id)
	response, err := pi.client.SSHKeys.Delete(pi.sshKey.ID)
	pi.manager.AddLog(pi.id, "delete-sshkey", fmt.Sprintf("%v\n%v\n%v", pi.sshKey, response, err))
	for key, device := range pi.devices {
		response, err := pi.client.Devices.Delete(device.ID)
		pi.manager.AddLog(pi.id, fmt.Sprintf("delete-device-%s", key), fmt.Sprintf("%v\n%v", response, err))
	}
	return nil
}

func (pi *packetInstance) GetRoot() string {
	return pi.root
}

func (pi *packetInstance) doDestroy(writer io.StringWriter, timeout time.Duration, err error) {
	_, _ = writer.WriteString(fmt.Sprintf("Error during k8s API initialisation %v", err))
	_, _ = writer.WriteString(fmt.Sprintf("Trying to destroy cluster"))
	// In case we failed to start and create cluster utils.
	err2 := pi.Destroy(timeout)
	if err2 != nil {
		_, _ = writer.WriteString(fmt.Sprintf("Error during destroy of cluster %v", err2))
	}
}

func (pi *packetInstance) doInstall(context context.Context) error {
	pi.provider.Lock()
	defer pi.provider.Unlock()
	if pi.installScript != nil && !pi.provider.installDone[pi.config.Name] {
		pi.provider.installDone[pi.config.Name] = true
		return pi.shellInterface.RunCmd(context, "install", pi.installScript, nil)
	}
	return nil
}

func (pi *packetInstance) updateProject() error {
	ps, response, err := pi.client.Projects.List(nil)

	pi.manager.AddLog(pi.id, "list-projects", fmt.Sprintf("%v\n%v\n%v", ps, response, err))

	if err != nil {
		logrus.Errorf("Failed to list Packet projects")
	}

	for i := 0; i > len(ps); i++ {
		p := &ps[i]
		if p.ID == pi.projectID {
			pp := ps[i]
			pi.project = &pp
			break
		}
	}

	if pi.project == nil {
		err := fmt.Errorf("specified project are not found on Packet %v", pi.projectID)
		logrus.Errorf(err.Error())
		return err
	}
	return nil
}

func (pi *packetInstance) createKey(keyFile string) ([]string, error) {
	pi.keyID = "dev-ci-cloud-" + pi.genID

	keyFileContent, err := utils.ReadFile(keyFile)
	if err != nil {
		logrus.Errorf("Failed to read file %v %v", keyFile, err)
		return nil, err
	}

	keyRequest := &packngo.SSHKeyCreateRequest{
		ProjectID: pi.project.ID,
		Label:     pi.keyID,
		Key:       strings.Join(keyFileContent, "\n"),
	}
	sshKey, _, _ := pi.client.SSHKeys.Create(keyRequest)

	sshKeys, response, err := pi.client.SSHKeys.List()

	keyIds := []string{}
	for k := 0; k < len(sshKeys); k++ {
		kk := &sshKeys[k]
		if kk.Label == pi.keyID {
			sshKey = &packngo.SSHKey{
				ID:          kk.ID,
				Label:       kk.Label,
				URL:         kk.URL,
				User:        kk.User,
				Key:         kk.Key,
				FingerPrint: kk.FingerPrint,
				Created:     kk.Created,
				Updated:     kk.Updated,
			}
		}
		keyIds = append(keyIds, kk.ID)
	}

	if sshKey == nil && err != nil {
		logrus.Errorf("Failed to create ssh key %v", err)
		return nil, err
	}

	pi.sshKey = sshKey
	pi.manager.AddLog(pi.id, "create-sshkey", fmt.Sprintf("%v\n%v\n%v", sshKey, response, err))
	return keyIds, nil
}

func (p *packetProvider) getProviderID(provider string) string {
	val, ok := p.indexes[provider]
	if ok {
		val++
	} else {
		val = 1
	}
	p.indexes[provider] = val
	return fmt.Sprintf("%d", val)
}

func (p *packetProvider) CreateCluster(config *config.ClusterProviderConfig, factory k8s.ValidationFactory,
	manager execmanager.ExecutionManager,
	instanceOptions providers.InstanceOptions) (providers.ClusterInstance, error) {
	err := p.ValidateConfig(config)
	if err != nil {
		return nil, err
	}
	p.Lock()
	defer p.Unlock()
	id := fmt.Sprintf("%s-%s", config.Name, p.getProviderID(config.Name))

	root := path.Join(p.root, id)

	clusterInstance := &packetInstance{
		manager:        manager,
		provider:       p,
		root:           root,
		id:             id,
		config:         config,
		configScript:   config.Scripts[configScript],
		installScript:  utils.ParseScript(config.Scripts[installScript]),
		setupScript:    utils.ParseScript(config.Scripts[setupScript]),
		startScript:    utils.ParseScript(config.Scripts[startScript]),
		prepareScript:  utils.ParseScript(config.Scripts[prepareScript]),
		stopScript:     utils.ParseScript(config.Scripts[stopScript]),
		factory:        factory,
		shellInterface: shell.NewManager(manager, id, root, config, instanceOptions),
		params:         instanceOptions,
		projectID:      os.Getenv(packetProjectID),
		packetAuthKey:  os.Getenv("PACKET_AUTH_TOKEN"),
		devices:        map[string]*packngo.Device{},
	}

	return clusterInstance, nil
}

// NewPacketClusterProvider - create new packet provider.
func NewPacketClusterProvider(root string) providers.ClusterProvider {
	utils.ClearFolder(root, true)
	return &packetProvider{
		root:        root,
		clusters:    []packetInstance{},
		indexes:     map[string]int{},
		installDone: map[string]bool{},
	}
}

func (p *packetProvider) ValidateConfig(config *config.ClusterProviderConfig) error {

	if config.Packet == nil {
		return fmt.Errorf("packet configuration element should be specified")
	}

	if len(config.Packet.Facilities) == 0 {
		return fmt.Errorf("packet configuration facilities should be specified")
	}

	if len(config.Packet.Devices) == 0 {
		return fmt.Errorf("packet configuration devices should be specified")
	}

	if _, ok := config.Scripts[configScript]; !ok {
		hasKubeConfig := false
		for _, e := range config.Env {
			if strings.HasPrefix(e, "KUBECONFIG=") {
				hasKubeConfig = true
				break
			}
		}
		if !hasKubeConfig {
			return fmt.Errorf("invalid config location")
		}
	}
	if _, ok := config.Scripts[startScript]; !ok {
		return fmt.Errorf("invalid start script")
	}

	for _, envVar := range config.EnvCheck {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return fmt.Errorf("environment variable are not specified %s Required variables: %v", envValue, config.EnvCheck)
		}
	}

	envValue := os.Getenv("PACKET_AUTH_TOKEN")
	if envValue == "" {
		return fmt.Errorf("environment variable are not specified PACKET_AUTH_TOKEN")
	}

	envValue = os.Getenv("PACKET_PROJECT_ID")
	if envValue == "" {
		return fmt.Errorf("environment variable are not specified PACKET_AUTH_TOKEN")
	}

	return nil
}
