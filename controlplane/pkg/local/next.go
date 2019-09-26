// Copyright (c) 2019 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package local

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
)

const nextKey common.ContextKeyType = "Next"

type nextEndpoint struct {
	composite *CompositeNetworkService
	index     int
}

// WithNext -
//    Wraps 'parent' in a new Context that has the Next networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeNetworkService.Request/Close
func WithNext(parent context.Context, next networkservice.NetworkServiceServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextKey, next)
}

// Next -
//   Returns the Next networkservice.NetworkServiceServer to be called in the chain from the context.Context
func Next(ctx context.Context) networkservice.NetworkServiceServer {
	if rv, ok := ctx.Value(nextKey).(networkservice.NetworkServiceServer); ok {
		return rv
	}
	return nil
}

// ProcessNext - performs a next operation on chain if defined.
func ProcessNext(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return request.Connection, nil
}

// ProcessClose - performs a next close operation on chain if defined
func ProcessClose(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func (n *nextEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if n.index+1 < len(n.composite.services) {
		ctx = WithNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	} else {
		ctx = WithNext(ctx, nil)
	}

	// Create a new span
	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		span, ctx = opentracing.StartSpanFromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(n.composite.services[n.index])))
		defer span.Finish()
		// Make sure we log to span
	}
	logger := common.LogFromSpan(span)

	ctx = common.WithLog(ctx, logger)
	logger.Infof("internal request %v", request)

	// Actually call the next
	rv, err := n.composite.services[n.index].Request(ctx, request)

	if err != nil {
		logger.Errorf("Error: %v", err)
		if span != nil {
			span.LogFields(log.Error(err))
		}
		return nil, err
	}
	logger.Infof("internal response %v", rv)
	return rv, err
}

func (n *nextEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if n.index+1 < len(n.composite.services) {
		ctx = WithNext(ctx, &nextEndpoint{composite: n.composite, index: n.index + 1})
	} else {
		ctx = WithNext(ctx, nil)
	}
	// Create a new span
	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		span, ctx = opentracing.StartSpanFromContext(ctx, fmt.Sprintf("%s.Close", typeutils.GetTypeName(n.composite.services[n.index])))
		defer span.Finish()
	}
	// Make sure we log to span
	logger := common.LogFromSpan(span)
	ctx = common.WithLog(ctx, logger)

	logger.Infof("internal request %v", connection)
	rv, err := n.composite.services[n.index].Close(ctx, connection)

	if err != nil {
		if span != nil {
			span.LogFields(log.Error(err))
		}
		logrus.Error(err)
		return nil, err
	}
	logger.Infof("internal response %v", rv)
	return rv, err
}
