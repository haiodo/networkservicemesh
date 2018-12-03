// Copyright (c) 2018 Cisco and/or its affiliates.
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
	"encoding/binary"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/ligato/networkservicemesh/pkg/tools"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

func (ns *networkService) CompleteConnection(request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	err := request.IsValid()
	if err != nil {
		return nil, err
	}
	netns, _ := tools.GetCurrentNS()
	mechanism := &connection.Mechanism{
		Type: connection.MechanismType_KERNEL_INTERFACE,
		Parameters: map[string]string{
			connection.NetNsInodeKey: netns,
			// TODO: Fix this terrible hack using xid for getting a unique interface name
			connection.InterfaceNameKey: "nsm" + request.GetConnection().GetId(),
		},
	}

	srcIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(srcIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 1

	dstIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(dstIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 3

	// TODO take into consideration LocalMechnism preferences sent in request

	connection := &connection.Connection{
		Id:             request.GetConnection().GetId(),
		NetworkService: request.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context: &connectioncontext.ConnectionContext{
			SrcIpAddr: srcIP.String() + "/30",
			DstIpAddr: dstIP.String() + "/30",
			Routes: []*connectioncontext.Route{
				&connectioncontext.Route{
					Prefix: "8.8.8.8",
				},
			},
		},
	}
	err = connection.IsComplete()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return connection, nil
}
