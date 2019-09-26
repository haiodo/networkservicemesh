package nsm

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go/log"
	"time"

	"github.com/opentracing/opentracing-go"

	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type healProcessor struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
	properties      *nsm.Properties

	manager    nsm.NetworkServiceRequestManager
	nseManager nsm.NetworkServiceEndpointManager

	eventCh chan healEvent
}

type healEvent struct {
	healID    string
	cc        *model.ClientConnection
	healState nsm.HealState
	ctx       context.Context
}

func newNetworkServiceHealProcessor(
	serviceRegistry serviceregistry.ServiceRegistry,
	model model.Model,
	properties *nsm.Properties,
	manager nsm.NetworkServiceRequestManager,
	nseManager nsm.NetworkServiceEndpointManager) nsm.NetworkServiceHealProcessor {

	p := &healProcessor{
		serviceRegistry: serviceRegistry,
		model:           model,
		properties:      properties,
		manager:         manager,
		nseManager:      nseManager,
		eventCh:         make(chan healEvent, 1),
	}
	go p.serve()

	return p
}

func (p *healProcessor) Heal(ctx context.Context, clientConnection nsm.ClientConnection, healState nsm.HealState) {
	cc := clientConnection.(*model.ClientConnection)

	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		opName := "Heal"
		if clientConnection.GetConnectionSource().IsRemote() {
			opName = "RemoteHeal"
		}
		span, ctx = opentracing.StartSpanFromContext(ctx, opName)
		defer span.Finish()
	}

	logger := common.LogFromSpan(span)
	ctx = common.WithLog(ctx, logger)

	healID := create_logid()
	logger.Infof("NSM_Heal(%v) %v", healID, cc)

	if !p.properties.HealEnabled {
		logger.Infof("NSM_Heal(%v) Is Disabled/Closing connection %v", healID, cc)
		_ = p.CloseConnection(ctx, cc)
		return
	}

	if modelCC := p.model.GetClientConnection(cc.GetID()); modelCC == nil {
		logger.Errorf("NSM_Heal(%v) Trying to heal not existing connection", healID)
		return
	} else if modelCC.ConnectionState != model.ClientConnectionReady {
		healErr := fmt.Errorf("NSM_Heal(%v) Trying to heal connection in bad state", healID)
		if span != nil {
			span.LogFields(log.Error(healErr))
		}
		return
	}

	cc = p.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionHealingBegin
	})

	p.eventCh <- healEvent{
		healID:    healID,
		cc:        cc,
		healState: healState,
		ctx:       ctx,
	}
}

func (p *healProcessor) CloseConnection(ctx context.Context, conn nsm.ClientConnection) error {
	var err error
	if conn.GetConnectionSource().IsRemote() {
		_, err = p.manager.RemoteManager().Close(ctx, conn.GetConnectionSource().(*remote_connection.Connection))
	} else {
		_, err = p.manager.LocalManager(conn).Close(ctx, conn.GetConnectionSource().(*local_connection.Connection))
	}
	if err != nil {
		logrus.Errorf("NSM_Heal Error in Close: %v", err)
	}
	return err
}

func (p *healProcessor) serve() {
	for {
		e, ok := <-p.eventCh
		if !ok {
			return
		}

		go func() {
			var ctx = e.ctx
			var span opentracing.Span

			if tools.IsOpentracingEnabled() {
				span, ctx = opentracing.StartSpanFromContext(e.ctx, "heal")
			}
			logger := common.LogFromSpan(span)
			defer func() {
				logger.Infof("NSM_Heal(%v) Connection %v healing state is finished...", e.healID, e.cc.GetID())
				if span != nil {
					span.Finish()
				}
			}()

			healed := false

			ctx = common.WithModelConnection(ctx, e.cc)

			switch e.healState {
			case nsm.HealStateDstDown:
				healed = p.healDstDown(ctx, e.cc)
			case nsm.HealStateDataplaneDown:
				healed = p.healDataplaneDown(ctx, e.cc)
			case nsm.HealStateDstUpdate:
				healed = p.healDstUpdate(ctx, e.cc)
			case nsm.HealStateDstNmgrDown:
				healed = p.healDstNmgrDown(ctx, e.cc)
			}

			if healed {
				logger.Infof("NSM_Heal(%v) Heal: Connection recovered: %v", e.healID, e.cc)
			} else {
				_ = p.CloseConnection(ctx, e.cc)
			}
		}()
	}
}

func (p *healProcessor) healDstDown(ctx context.Context, cc *model.ClientConnection) bool {
	var span opentracing.Span
	if tools.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, "healDstDown")
		defer span.Finish()
	}
	logger := common.LogFromSpan(span)

	logger.Infof("NSM_Heal(1.1.1) Checking if DST die is NSMD/DST die...")
	// Check if this is a really HealStateDstDown or HealStateDstNmgrDown
	if !p.nseManager.IsLocalEndpoint(cc.Endpoint) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.properties.HealTimeout*3)
		defer cancel()
		remoteNsmClient, err := p.nseManager.CreateNSEClient(ctx, cc.Endpoint)
		if remoteNsmClient != nil {
			_ = remoteNsmClient.Cleanup()
		}
		if err != nil {
			// This is NSMD die case.
			logger.Infof("NSM_Heal(1.1.2) Connection healing state is %v...", nsm.HealStateDstNmgrDown)
			if span != nil {
				span.LogFields(log.Error(err))
			}
			return p.healDstNmgrDown(ctx, cc)
		}
	}

	logger.Infof("NSM_Heal(1.1.2) Connection healing state is %v...", nsm.HealStateDstDown)

	// Destination is down, we need to find it again.
	if cc.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		logger.Infof("NSM_Heal(2.1) Remote NSE heal is done on source side")
		return false
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, p.properties.HealTimeout*3)
	defer cancel()

	logger.Infof("NSM_Heal(2.2) Starting DST Heal...")
	// We are client NSMd, we need to try recover our connection srv.

	endpointName := cc.Endpoint.GetNetworkServiceEndpoint().GetName()

	// Wait for NSE not equal to down one, since we know it will be re-registered with new endpoint name.
	if !p.waitNSE(ctx, cc, endpointName, cc.GetNetworkService(), p.nseIsNewAndAvailable) {
		// Mark endpoint as ignored.
		ctx = common.WithIgnoredEndpoints(ctx, map[string]*registry.NSERegistration{
			endpointName: cc.Endpoint,
		})
	}
	// Fallback to heal with choose of new NSE.
	requestCtx, requestCancel := context.WithTimeout(ctx, p.properties.HealRequestTimeout)
	defer requestCancel()

	logger.Infof("NSM_Heal(2.3.0) Starting Heal by calling request: %v", cc.Request)
	if _, err := p.manager.LocalManager(cc).Request(requestCtx, cc.Request.(*local_networkservice.NetworkServiceRequest)); err != nil {
		logger.Errorf("NSM_Heal(2.3.1) Failed to heal connection: %v", err)
		return false
	}

	return true
}

func (p *healProcessor) healDataplaneDown(ctx context.Context, cc *model.ClientConnection) bool {
	var span opentracing.Span
	if tools.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, "healDataplaneDown")
		defer span.Finish()
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, p.properties.HealTimeout)
	defer cancel()

	logger := common.LogFromSpan(span)
	// Dataplane is down, we only need to re-programm dataplane.
	// 1. Wait for dataplane to appear.
	logger.Infof("NSM_Heal(3.1) Waiting for Dataplane to recovery...")
	if err := p.serviceRegistry.WaitForDataplaneAvailable(ctx, p.model, p.properties.HealDataplaneTimeout); err != nil {
		err = fmt.Errorf("NSM_Heal(3.1) Dataplane is not available on recovery for timeout %v: %v", p.properties.HealDataplaneTimeout, err)
		if span != nil {
			span.LogFields(log.Error(err))
		}
		return false
	}
	logger.Infof("NSM_Heal(3.2) Dataplane is now available...")

	// 3.3. Set source connection down
	p.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.GetConnectionSource().SetConnectionState(connection.StateDown)
	})

	if cc.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		// Recovery will be performed by NSM client side.
		logger.Infof("NSM_Heal(3.4)  Healing will be continued on source side...")
		return true
	}

	request := cc.Request.Clone()
	request.SetRequestConnection(cc.GetConnectionSource())

	if _, err := p.manager.LocalManager(cc).Request(ctx, cc.Request.(*local_networkservice.NetworkServiceRequest)); err != nil {
		logger.Errorf("NSM_Heal(3.5) Failed to heal connection: %v", err)
		return false
	}

	return true
}

func (p *healProcessor) healDstUpdate(ctx context.Context, cc *model.ClientConnection) bool {
	var span opentracing.Span
	if tools.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, "healDstUpdate")
		defer span.Finish()
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, p.properties.HealTimeout)
	defer cancel()

	logger := common.LogFromSpan(span)
	// Destination is updated.
	// Update request to contain a proper connection object from previous attempt.
	logger.Infof("NSM_Heal(5.1-%v) Healing Src Update... %v", cc)
	if cc.Request == nil {
		return false
	}

	request := cc.Request.Clone()
	request.SetRequestConnection(cc.GetConnectionSource())

	err := p.performRequest(ctx, request, cc)
	if err != nil {
		if span != nil {
			span.LogFields(log.Error(err))
		}
		logger.Errorf("NSM_Heal(5.2) Failed to heal connection: %v", err)
		return false
	}

	return true
}

func (p *healProcessor) performRequest(ctx context.Context, request networkservice.Request, cc *model.ClientConnection) error {
	var err error
	if request.IsRemote() {
		_, err = p.manager.RemoteManager().Request(ctx, request.(*remote_networkservice.NetworkServiceRequest))
	} else {
		_, err = p.manager.LocalManager(cc).Request(ctx, request.(*local_networkservice.NetworkServiceRequest))
	}
	return err
}

func (p *healProcessor) healDstNmgrDown(ctx context.Context, cc *model.ClientConnection) bool {
	var span opentracing.Span
	if tools.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, "healDstNsmgrDown")
		defer span.Finish()
	}
	logger := common.LogFromSpan(span)
	logger.Infof("NSM_Heal(6.1-%v) Starting DST + NSMGR Heal...")

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, p.properties.HealTimeout*3)
	defer cancel()

	networkService := cc.GetNetworkService()

	var endpointName string
	// Wait for exact same NSE to be available with NSMD connection alive.
	if cc.Endpoint != nil {
		endpointName = cc.Endpoint.GetNetworkServiceEndpoint().GetName()
		if !p.waitNSE(ctx, cc, endpointName, networkService, p.nseIsSameAndAvailable) {
			ctx = common.WithIgnoredEndpoints(ctx, map[string]*registry.NSERegistration{
				endpointName: cc.Endpoint,
			})
		}
	}

	requestCtx, requestCancel := context.WithTimeout(ctx, p.properties.HealRequestTimeout)
	defer requestCancel()

	if err := p.performRequest(requestCtx, cc.Request, cc); err != nil {
		logger.Warnf("NSM_Heal(6.2.1) Failed to heal connection with same NSE from registry: %v", err)

		// 6.2.2. We are still healing
		p.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(modelCC *model.ClientConnection) {
			modelCC.ConnectionState = model.ClientConnectionHealingBegin
		})

		// In this case, most probable both NSMD and NSE are die, and registry was outdated on moment of waitNSE.
		if endpointName == "" || !p.waitNSE(ctx, cc, endpointName, networkService, p.nseIsNewAndAvailable) {
			return false
		}

		// Ok we have NSE, lets retry request
		requestCtx, requestCancel = context.WithTimeout(ctx, p.properties.HealRequestTimeout)
		defer requestCancel()

		if err := p.performRequest(requestCtx, cc.Request, cc); err != nil {
			err = fmt.Errorf("NSM_Heal(6.2.3) Failed to heal connection: %v", err)
			if span != nil {
				span.LogFields(log.Error(err))
			}
			return false
		}
	}

	return true
}

type nseValidator func(ctx context.Context, endpoint string, reg *registry.NSERegistration) bool

func (p *healProcessor) nseIsNewAndAvailable(ctx context.Context, endpointName string, reg *registry.NSERegistration) bool {
	if endpointName != "" && reg.GetNetworkServiceEndpoint().GetName() == endpointName {
		// Skip ignored endpoint
		return false
	}

	// Check local only if not waiting for specific NSE.
	if p.model.GetNsm().GetName() == reg.GetNetworkServiceManager().GetName() {
		// Another local endpoint is found, success.
		return true
	}

	// Check remote is accessible.
	if p.nseManager.CheckUpdateNSE(ctx, reg) {
		logrus.Infof("NSE is available and Remote NSMD is accessible. %s.", reg.NetworkServiceManager.Url)
		// We are able to connect to NSM with required NSE
		return true
	}

	return false
}

func (p *healProcessor) nseIsSameAndAvailable(ctx context.Context, endpointName string, reg *registry.NSERegistration) bool {
	if reg.GetNetworkServiceEndpoint().GetName() != endpointName {
		return false
	}

	// Our endpoint, we need to check if it is remote one and NSM is accessible.

	// Check remote is accessible.
	if p.nseManager.CheckUpdateNSE(ctx, reg) {
		logrus.Infof("NSE is available and Remote NSMD is accessible. %s.", reg.NetworkServiceManager.Url)
		// We are able to connect to NSM with required NSE
		return true
	}

	return false
}

func (p *healProcessor) waitNSE(ctx context.Context, cc *model.ClientConnection, endpointName, networkService string, nseValidator nseValidator) bool {
	var span opentracing.Span
	if tools.IsOpentracingEnabled() {
		span, ctx = opentracing.StartSpanFromContext(ctx, "waitNSE")
		span.LogFields(log.String("endpointName", endpointName))
		span.LogFields(log.String("networkService", networkService))
		defer span.Finish()
	}
	logger := common.LogFromSpan(span)
	discoveryClient, err := p.serviceRegistry.DiscoveryClient()
	if err != nil {
		if span != nil {
			span.LogFields(log.Error(err))
		}
		logger.Errorf("Failed to connect to Registry... %v", err)
		// Still try to recovery
		return false
	}

	st := time.Now()

	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: networkService,
	}

	defer func() {
		logger.Infof("Complete Waiting for Remote NSE/NSMD with network service %s. Since elapsed: %v", networkService, time.Since(st))
	}()

	for {
		logger.Infof("NSM: RemoteNSE: Waiting for NSE with network service %s. Since elapsed: %v", networkService, time.Since(st))

		endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
		if err == nil {
			for _, ep := range endpointResponse.NetworkServiceEndpoints {
				reg := &registry.NSERegistration{
					NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[ep.GetNetworkServiceManagerName()],
					NetworkServiceEndpoint: ep,
					NetworkService:         endpointResponse.GetNetworkService(),
				}

				if nseValidator(ctx, endpointName, reg) {
					return true
				}
			}
		}

		if time.Since(st) > p.properties.HealDSTNSEWaitTimeout {
			err = fmt.Errorf("Timeout waiting for NetworkService: %v", networkService)
			if span != nil {
				span.LogFields(log.Error(err))
			}
			return false
		}
		// Wait a bit
		<-time.After(p.properties.HealDSTNSEWaitTick)
	}
}
