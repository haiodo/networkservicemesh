package nsm

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

//// Endpoint Connection Client
type endpointClient struct {
	client     networkservice.NetworkServiceClient
	connection *grpc.ClientConn
}

func (c *endpointClient) Request(ctx context.Context, request nsm.NSMRequest) (nsm.NSMConnection, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("NSE Connection is not initialized...")
	}
	return c.client.Request(ctx, request.(*networkservice.NetworkServiceRequest))
}
func (c *endpointClient) Cleanup() error {
	if c == nil || c.client == nil {
		return fmt.Errorf("NSE Connection is not initialized...")
	}
	var err error
	if c.connection != nil { // Required for testing
		err = c.connection.Close()
	}
	c.connection = nil
	return err
}
func (c *endpointClient) Close(ctx context.Context, conn nsm.NSMConnection) error {
	if c.connection == nil {
		return fmt.Errorf("Remote NSM Connection is already cleaned...")
	}
	_, err := c.client.Close(ctx, conn.(*connection.Connection))
	c.connection = nil
	return err
}
