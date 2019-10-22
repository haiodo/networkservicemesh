package nsm

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	nsm_properties "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"

	"github.com/sirupsen/logrus"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type nseManager struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
	properties      *nsm_properties.Properties
	nsm.NetworkServiceEndpointManager
}

func (nsem *nseManager) GetEndpoint(ctx context.Context, requestConnection connection.Connection, ignoreEndpoints map[registry.EndpointNSMName]*registry.NSERegistration) (*registry.NSERegistration, error) {
	span := spanhelper.FromContext(ctx, "GetEndpoint")
	defer span.Finish()
	span.LogObject("request", requestConnection)
	span.LogObject("ignores", ignoreEndpoints)
	// Handle case we are remote NSM and asked for particular endpoint to connect to.
	targetEndpoint := requestConnection.GetNetworkServiceEndpointName()
	span.LogObject("targetEndpoint", targetEndpoint)
	if len(targetEndpoint) > 0 {
		endpoint := nsem.model.GetEndpoint(targetEndpoint)
		if endpoint != nil && ignoreEndpoints[endpoint.Endpoint.GetEndpointNSMName()] == nil {
			return endpoint.Endpoint, nil
		} else {
			return nil, errors.Errorf("Could not find endpoint with name: %s at local registry", targetEndpoint)
		}
	}

	// Get endpoints, do it every time since we do not know if list are changed or not.
	discoveryClient, err := nsem.serviceRegistry.DiscoveryClient(ctx)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: requestConnection.GetNetworkService(),
	}
	span.LogObject("nseRequest", nseRequest)
	endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
	span.LogObject("nseResponse", endpointResponse)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	endpoints := nsem.filterEndpoints(endpointResponse.GetNetworkServiceEndpoints(), endpointResponse.NetworkServiceManagers, ignoreEndpoints)

	if len(endpoints) == 0 {
		err = errors.Errorf("failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignoreEndpoints), len(endpoints))
		span.LogError(err)
		return nil, err
	}

	endpoint := nsem.model.GetSelector().SelectEndpoint(requestConnection.(*local.Connection), endpointResponse.GetNetworkService(), endpoints)
	if endpoint == nil {
		err = errors.Errorf("failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignoreEndpoints), len(endpoints))
		span.LogError(err)
		return nil, err
	}

	return &registry.NSERegistration{
		NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[endpoint.GetNetworkServiceManagerName()],
		NetworkServiceEndpoint: endpoint,
		NetworkService:         endpointResponse.GetNetworkService(),
	}, nil
}

func (nsem *nseManager) CreateNSMClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceConnectionClient, error) {
	span := spanhelper.FromContext(ctx, "createNSMClient")
	defer span.Finish()
	logger := span.Logger()
	if !nsem.IsLocalEndpoint(endpoint) {
		logger.Infof("Create remote NSE connection to endpoint: %v", endpoint)
		ctx, cancel := context.WithTimeout(span.Context(), nsem.properties.HealRequestConnectTimeout)
		defer cancel()
		client, conn, err := nsem.serviceRegistry.RemoteNetworkServiceClient(ctx, endpoint.GetNetworkServiceManager())
		if err != nil {
			return nil, err
		}
		return &networkServiceConnectionClient{client: client, connection: conn}, nil
	}
	return nil, errors.New("endpoint connection should be local only.")
}
func (nsem *nseManager) CreateEndpointClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceConnectionClient, error) {
	span := spanhelper.FromContext(ctx, "createNSEClient")
	defer span.Finish()
	logger := span.Logger()
	if nsem.IsLocalEndpoint(endpoint) {
		modelEp := nsem.model.GetEndpoint(endpoint.GetNetworkServiceEndpoint().GetName())
		if modelEp == nil {
			return nil, errors.Errorf("Endpoint not found: %v", endpoint)
		}
		logger.Infof("Create local NSE connection to endpoint: %v", modelEp)
		client, conn, err := nsem.serviceRegistry.EndpointConnection(span.Context(), modelEp)
		if err != nil {
			span.LogError(err)
			// We failed to connect to local NSE.
			nsem.cleanupNSE(ctx, modelEp)
			return nil, err
		}
		return &networkServiceConnectionClient{connection: conn, client: client}, nil
	}
	return nil, errors.New("endpoint connection should be local only.")
}

func (nsem *nseManager) IsLocalEndpoint(endpoint *registry.NSERegistration) bool {
	return nsem.model.GetNsm().GetName() == endpoint.GetNetworkServiceEndpoint().GetNetworkServiceManagerName()
}

func (nsem *nseManager) cleanupNSE(ctx context.Context, endpoint *model.Endpoint) {
	// Remove endpoint from model and put workspace into BAD state.
	nsem.model.DeleteEndpoint(ctx, endpoint.EndpointName())
	logrus.Infof("NSM: Remove Endpoint since it is not available... %v", endpoint)
}

func (nsem *nseManager) filterEndpoints(endpoints []*registry.NetworkServiceEndpoint, managers map[string]*registry.NetworkServiceManager, ignoreEndpoints map[registry.EndpointNSMName]*registry.NSERegistration) []*registry.NetworkServiceEndpoint {
	result := []*registry.NetworkServiceEndpoint{}
	// Do filter of endpoints
	for _, candidate := range endpoints {
		endpointName := registry.NewEndpointNSMName(candidate, managers[candidate.NetworkServiceManagerName])
		if ignoreEndpoints[endpointName] == nil {
			result = append(result, candidate)
		}
	}
	return result
}

func (nsem *nseManager) CheckUpdateNSE(ctx context.Context, reg *registry.NSERegistration) bool {
	pingCtx, pingCancel := context.WithTimeout(ctx, nsem.properties.HealRequestConnectCheckTimeout)
	defer pingCancel()

	if nsem.IsLocalEndpoint(reg) {
		client, err := nsem.CreateEndpointClient(pingCtx, reg)
		if err == nil && client != nil {
			_ = client.Cleanup()
			return true
		}
	} else {
		client, err := nsem.CreateNSMClient(pingCtx, reg)
		if err == nil && client != nil {
			_ = client.Cleanup()
			return true
		}
	}

	return false
}
