package nsm

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/pkg/errors"
)

//// Endpoint Connection Client
type networkServiceConnectionClient struct {
	client     networkservice.NetworkServiceClient
	connection *grpc.ClientConn
	nsm.NetworkServiceConnectionClient
}

func (c *networkServiceConnectionClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("NSE Connection is not initialized...")
	}

	response, err := c.client.Request(ctx, request)
	if err != nil {
		return nil, err
	}

	return response.Clone(), nil
}
func (c *networkServiceConnectionClient) Cleanup() error {
	if c == nil || c.client == nil {
		return errors.New("NSE Connection is not initialized...")
	}
	var err error
	if c.connection != nil { // Required for testing
		err = c.connection.Close()
	}
	c.connection = nil
	c.client = nil
	return err
}

func (c *networkServiceConnectionClient) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if c.client == nil {
		return nil, errors.New("Remote NSM Connection is already cleaned...")
	}
	empt, err := c.client.Close(ctx, conn)
	_ = c.Cleanup()
	return empt, err
}
