package network_service_server

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/remote/monitor_connection_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"math/rand"
	"time"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type remoteNetworkServiceServer struct {
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
	monitor         monitor_connection_server.MonitorConnectionServer
}

func (srv *remoteNetworkServiceServer) Request(ctx context.Context, request *remote_networkservice.NetworkServiceRequest) (*remote_connection.Connection, error) {
	logrus.Infof("RemoteNSMD: Received request from client to connect to NetworkService: %v", request)
	// Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	// Create a ConnectId for the request.GetConnection()
	request.GetConnection().Id = srv.model.ConnectionId()
	// TODO: Mechanism selection
	request.GetConnection().Mechanism = request.MechanismPreferences[0]

	// Get endpoints
	endpoint, err := getEndpoint(srv, request)
	if err != nil {
		return nil, err
	}

	// get dataplane
	dp, err := srv.model.SelectDataplane()
	if err != nil {
		return nil, err
	}

	logrus.Infof("RemoteNSMD: Preparing to program dataplane: %v...", dp)

	dataplaneClient, dataplaneConn, err := srv.serviceRegistry.DataplaneConnection(dp)
	if err != nil {
		return nil, err
	}
	if dataplaneConn != nil {
		defer dataplaneConn.Close()
	}

	var dpApiConnection *crossconnect.CrossConnect

	dpApiConnection, err = srv.performLocalNSERequest(ctx, request, endpoint)

	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Infof("RemoteNSMD: Sending request to dataplane: %v", dpApiConnection)

	dpCtx, dpCancel := context.WithTimeout(context.Background(), nseConnectionTimeout)
	defer dpCancel()
	rv, err := dataplaneClient.Request(dpCtx, dpApiConnection)
	if err != nil {
		logrus.Errorf("RemoteNSMD: Dataplane request failed: %s", err)
		return nil, err
	}
	// TODO - be more cautious here about bad return values from Dataplane
	con := rv.GetSource().(*crossconnect.CrossConnect_RemoteSource).RemoteSource
	srv.monitor.UpdateConnection(con)
	logrus.Info("RemoteNSMD: Dataplane configuration done...")
	return con, nil
}

func (srv *remoteNetworkServiceServer) createLocalNSERequest(endpoint *registry.NSERegistration, request *remote_networkservice.NetworkServiceRequest) *networkservice.NetworkServiceRequest {
	message := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			// TODO track connection ids
			Id:             srv.model.ConnectionId(),
			NetworkService: endpoint.GetNetworkService().GetName(),
			Context:        request.GetConnection().GetContext(),
			Labels:         nil,
		},
		MechanismPreferences: []*connection.Mechanism{
			&connection.Mechanism{
				Type:       connection.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{},
			},
			&connection.Mechanism{
				Type:       connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{},
			},
		},
	}
	return message
}

func (srv *remoteNetworkServiceServer) performLocalNSERequest(ctx context.Context, request *remote_networkservice.NetworkServiceRequest, endpoint *registry.NSERegistration) (*crossconnect.CrossConnect, error) {
	client, nseConn, err := srv.serviceRegistry.EndpointConnection(endpoint)
	if err != nil {
		return nil, err
	}
	if nseConn != nil {
		defer nseConn.Close()
	}

	message := srv.createLocalNSERequest(endpoint, request)

	nseConnection, e := client.Request(ctx, message)

	if e != nil {
		logrus.Errorf("error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
		return nil, e
	}

	err = srv.validateNSEConnection(nseConnection)
	if err != nil {
		return nil, err
	}

	request.GetConnection().Context = nseConnection.Context
	err = request.GetConnection().IsComplete()
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return nil, err
	}

	dpApiConnection := &crossconnect.CrossConnect{
		Id:      request.GetConnection().GetId(),
		Payload: endpoint.GetNetworkService().GetPayload(),
		Source: &crossconnect.CrossConnect_RemoteSource{
			request.GetConnection(),
		},
		Destination: &crossconnect.CrossConnect_LocalDestination{
			nseConnection,
		},
	}
	return dpApiConnection, nil
}

func (srv *remoteNetworkServiceServer) validateNSEConnection(nseConnection *connection.Connection) error {
	err := nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("NetworkService.Request() failed with error: %s", err)
		logrus.Error(err)
		return err
	}
	err = nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return err
	}
	return nil
}

func (srv *remoteNetworkServiceServer) Close(ctx context.Context, connection *remote_connection.Connection) (*empty.Empty, error) {
	srv.monitor.DeleteConnection(connection)
	//TODO: Add call to dataplane
	return nil, nil
}

func getEndpoint(srv *remoteNetworkServiceServer, request *remote_networkservice.NetworkServiceRequest) (*registry.NSERegistration, error) {
	endpoints := srv.model.GetNetworkServiceEndpoints(request.GetConnection().GetNetworkService())
	if len(endpoints) == 0 {
		// Request endpoints from registry
		return nil, errors.New(fmt.Sprintf("network service '%s' not found", request.Connection.NetworkService))
	}

	// Select endpoint at random
	idx := rand.Intn(len(endpoints))
	endpoint := endpoints[idx]
	if endpoint == nil {
		return nil, errors.New("should not see this error, scaffolding called")
	}
	return endpoint, nil
}

func NewRemoteNetworkServiceServer(model model.Model, serviceRegistry serviceregistry.ServiceRegistry, grpcServer *grpc.Server) {
	server := &remoteNetworkServiceServer{
		model:           model,
		serviceRegistry: serviceRegistry,
		monitor:         monitor_connection_server.NewMonitorConnectionServer(),
	}
	remote_networkservice.RegisterNetworkServiceServer(grpcServer, server)
}
