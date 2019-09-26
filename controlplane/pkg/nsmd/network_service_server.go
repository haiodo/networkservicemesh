package nsmd

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/local"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

// NewNetworkServiceServer - construct a local network service chain
func NewNetworkServiceServer(model model.Model, ws *Workspace,
	nsmManager nsm.NetworkServiceManager) networkservice.NetworkServiceServer {
	return local.NewCompositeService(
		local.NewRequestValidator(),
		local.NewMonitorService(ws.MonitorConnectionServer()),
		local.NewWorkspaceService(ws.Name()),
		local.NewConnectionService(model),
		local.NewDataplaneService(model, nsmManager.ServiceRegistry()),
		local.NewEndpointSelectorService(nsmManager.NseManager(), nsmManager.PluginRegistry()),
		local.NewEndpointService(nsmManager.NseManager(), nsmManager.GetHealProperties(), nsmManager.Model(), nsmManager.PluginRegistry()),
		local.NewCrossConnectService(),
	)
}
