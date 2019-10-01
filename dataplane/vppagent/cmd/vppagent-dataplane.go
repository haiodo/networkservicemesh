// Copyright (c) 2018-2019 Cisco and/or its affiliates.
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

package main

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/pkg/probes"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/vppagent"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

var version string

func main() {
	logrus.Info("Starting vppagent-dataplane...")
	logrus.Infof("Version: %v", version)

	closer := jaeger.InitJaeger("vppagent-dataplane")
	defer func() { _ = closer.Close() }()

	span := spanhelper.FromContext(context.Background(), "Start.VPPAgent.Dataplane")
	defer span.Finish()
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	dataplaneGoals := &common.DataplaneProbeGoals{}
	dataplaneProbes := probes.New("Vppagent dataplane liveness/readiness healthcheck", dataplaneGoals)
	dataplaneProbes.BeginHealthCheck()

	agent := vppagent.CreateVPPAgent(span.Context())

	registration := common.CreateDataplane(span.Context(), agent, dataplaneGoals)

	for range c {
		logrus.Info("Closing Dataplane Registration")
		registration.Close()
	}
}
