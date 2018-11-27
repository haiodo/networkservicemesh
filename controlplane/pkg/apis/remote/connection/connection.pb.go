// Code generated by protoc-gen-go. DO NOT EDIT.
// source: connection.proto

package connection

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type MechanismType int32

const (
	MechanismType_NONE          MechanismType = 0
	MechanismType_VXLAN         MechanismType = 1
	MechanismType_VXLAN_GPE     MechanismType = 2
	MechanismType_GRE           MechanismType = 3
	MechanismType_SRV6          MechanismType = 4
	MechanismType_MPLSoEthernet MechanismType = 5
	MechanismType_MPLSoGRE      MechanismType = 6
	MechanismType_MPLSoUDP      MechanismType = 7
)

var MechanismType_name = map[int32]string{
	0: "NONE",
	1: "VXLAN",
	2: "VXLAN_GPE",
	3: "GRE",
	4: "SRV6",
	5: "MPLSoEthernet",
	6: "MPLSoGRE",
	7: "MPLSoUDP",
}

var MechanismType_value = map[string]int32{
	"NONE":          0,
	"VXLAN":         1,
	"VXLAN_GPE":     2,
	"GRE":           3,
	"SRV6":          4,
	"MPLSoEthernet": 5,
	"MPLSoGRE":      6,
	"MPLSoUDP":      7,
}

func (x MechanismType) String() string {
	return proto.EnumName(MechanismType_name, int32(x))
}

func (MechanismType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_51baa40a1cc6b48b, []int{0}
}

type ConnectionEventType int32

const (
	ConnectionEventType_INITIAL_STATE_TRANSFER ConnectionEventType = 0
	ConnectionEventType_UPDATE                 ConnectionEventType = 1
	ConnectionEventType_DELETE                 ConnectionEventType = 2
)

var ConnectionEventType_name = map[int32]string{
	0: "INITIAL_STATE_TRANSFER",
	1: "UPDATE",
	2: "DELETE",
}

var ConnectionEventType_value = map[string]int32{
	"INITIAL_STATE_TRANSFER": 0,
	"UPDATE":                 1,
	"DELETE":                 2,
}

func (x ConnectionEventType) String() string {
	return proto.EnumName(ConnectionEventType_name, int32(x))
}

func (ConnectionEventType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_51baa40a1cc6b48b, []int{1}
}

type Mechanism struct {
	Type                 MechanismType     `protobuf:"varint,1,opt,name=type,proto3,enum=remote.connection.MechanismType" json:"type,omitempty"`
	Parameters           map[string]string `protobuf:"bytes,2,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Mechanism) Reset()         { *m = Mechanism{} }
func (m *Mechanism) String() string { return proto.CompactTextString(m) }
func (*Mechanism) ProtoMessage()    {}
func (*Mechanism) Descriptor() ([]byte, []int) {
	return fileDescriptor_51baa40a1cc6b48b, []int{0}
}

func (m *Mechanism) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Mechanism.Unmarshal(m, b)
}
func (m *Mechanism) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Mechanism.Marshal(b, m, deterministic)
}
func (m *Mechanism) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Mechanism.Merge(m, src)
}
func (m *Mechanism) XXX_Size() int {
	return xxx_messageInfo_Mechanism.Size(m)
}
func (m *Mechanism) XXX_DiscardUnknown() {
	xxx_messageInfo_Mechanism.DiscardUnknown(m)
}

var xxx_messageInfo_Mechanism proto.InternalMessageInfo

func (m *Mechanism) GetType() MechanismType {
	if m != nil {
		return m.Type
	}
	return MechanismType_NONE
}

func (m *Mechanism) GetParameters() map[string]string {
	if m != nil {
		return m.Parameters
	}
	return nil
}

type Connection struct {
	Id                                   string            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	NetworkService                       string            `protobuf:"bytes,2,opt,name=network_service,json=networkService,proto3" json:"network_service,omitempty"`
	Mechanism                            *Mechanism        `protobuf:"bytes,3,opt,name=mechanism,proto3" json:"mechanism,omitempty"`
	Context                              map[string]string `protobuf:"bytes,4,rep,name=context,proto3" json:"context,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Labels                               map[string]string `protobuf:"bytes,5,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	SourceNetworkServiceManagerName      string            `protobuf:"bytes,6,opt,name=source_network_service_manager_name,json=sourceNetworkServiceManagerName,proto3" json:"source_network_service_manager_name,omitempty"`
	DestinationNetworkServiceManagerName string            `protobuf:"bytes,7,opt,name=destination_network_service_manager_name,json=destinationNetworkServiceManagerName,proto3" json:"destination_network_service_manager_name,omitempty"`
	NetworkServiceEndpointName           string            `protobuf:"bytes,8,opt,name=network_service_endpoint_name,json=networkServiceEndpointName,proto3" json:"network_service_endpoint_name,omitempty"`
	XXX_NoUnkeyedLiteral                 struct{}          `json:"-"`
	XXX_unrecognized                     []byte            `json:"-"`
	XXX_sizecache                        int32             `json:"-"`
}

func (m *Connection) Reset()         { *m = Connection{} }
func (m *Connection) String() string { return proto.CompactTextString(m) }
func (*Connection) ProtoMessage()    {}
func (*Connection) Descriptor() ([]byte, []int) {
	return fileDescriptor_51baa40a1cc6b48b, []int{1}
}

func (m *Connection) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Connection.Unmarshal(m, b)
}
func (m *Connection) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Connection.Marshal(b, m, deterministic)
}
func (m *Connection) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Connection.Merge(m, src)
}
func (m *Connection) XXX_Size() int {
	return xxx_messageInfo_Connection.Size(m)
}
func (m *Connection) XXX_DiscardUnknown() {
	xxx_messageInfo_Connection.DiscardUnknown(m)
}

var xxx_messageInfo_Connection proto.InternalMessageInfo

func (m *Connection) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Connection) GetNetworkService() string {
	if m != nil {
		return m.NetworkService
	}
	return ""
}

func (m *Connection) GetMechanism() *Mechanism {
	if m != nil {
		return m.Mechanism
	}
	return nil
}

func (m *Connection) GetContext() map[string]string {
	if m != nil {
		return m.Context
	}
	return nil
}

func (m *Connection) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *Connection) GetSourceNetworkServiceManagerName() string {
	if m != nil {
		return m.SourceNetworkServiceManagerName
	}
	return ""
}

func (m *Connection) GetDestinationNetworkServiceManagerName() string {
	if m != nil {
		return m.DestinationNetworkServiceManagerName
	}
	return ""
}

func (m *Connection) GetNetworkServiceEndpointName() string {
	if m != nil {
		return m.NetworkServiceEndpointName
	}
	return ""
}

type ConnectionEvent struct {
	Type                 ConnectionEventType    `protobuf:"varint,1,opt,name=type,proto3,enum=remote.connection.ConnectionEventType" json:"type,omitempty"`
	Connections          map[string]*Connection `protobuf:"bytes,2,rep,name=connections,proto3" json:"connections,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}               `json:"-"`
	XXX_unrecognized     []byte                 `json:"-"`
	XXX_sizecache        int32                  `json:"-"`
}

func (m *ConnectionEvent) Reset()         { *m = ConnectionEvent{} }
func (m *ConnectionEvent) String() string { return proto.CompactTextString(m) }
func (*ConnectionEvent) ProtoMessage()    {}
func (*ConnectionEvent) Descriptor() ([]byte, []int) {
	return fileDescriptor_51baa40a1cc6b48b, []int{2}
}

func (m *ConnectionEvent) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ConnectionEvent.Unmarshal(m, b)
}
func (m *ConnectionEvent) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ConnectionEvent.Marshal(b, m, deterministic)
}
func (m *ConnectionEvent) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConnectionEvent.Merge(m, src)
}
func (m *ConnectionEvent) XXX_Size() int {
	return xxx_messageInfo_ConnectionEvent.Size(m)
}
func (m *ConnectionEvent) XXX_DiscardUnknown() {
	xxx_messageInfo_ConnectionEvent.DiscardUnknown(m)
}

var xxx_messageInfo_ConnectionEvent proto.InternalMessageInfo

func (m *ConnectionEvent) GetType() ConnectionEventType {
	if m != nil {
		return m.Type
	}
	return ConnectionEventType_INITIAL_STATE_TRANSFER
}

func (m *ConnectionEvent) GetConnections() map[string]*Connection {
	if m != nil {
		return m.Connections
	}
	return nil
}

type MonitorScopeSelector struct {
	NetworkServiceManagerName string   `protobuf:"bytes,1,opt,name=network_service_manager_name,json=networkServiceManagerName,proto3" json:"network_service_manager_name,omitempty"`
	XXX_NoUnkeyedLiteral      struct{} `json:"-"`
	XXX_unrecognized          []byte   `json:"-"`
	XXX_sizecache             int32    `json:"-"`
}

func (m *MonitorScopeSelector) Reset()         { *m = MonitorScopeSelector{} }
func (m *MonitorScopeSelector) String() string { return proto.CompactTextString(m) }
func (*MonitorScopeSelector) ProtoMessage()    {}
func (*MonitorScopeSelector) Descriptor() ([]byte, []int) {
	return fileDescriptor_51baa40a1cc6b48b, []int{3}
}

func (m *MonitorScopeSelector) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MonitorScopeSelector.Unmarshal(m, b)
}
func (m *MonitorScopeSelector) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MonitorScopeSelector.Marshal(b, m, deterministic)
}
func (m *MonitorScopeSelector) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MonitorScopeSelector.Merge(m, src)
}
func (m *MonitorScopeSelector) XXX_Size() int {
	return xxx_messageInfo_MonitorScopeSelector.Size(m)
}
func (m *MonitorScopeSelector) XXX_DiscardUnknown() {
	xxx_messageInfo_MonitorScopeSelector.DiscardUnknown(m)
}

var xxx_messageInfo_MonitorScopeSelector proto.InternalMessageInfo

func (m *MonitorScopeSelector) GetNetworkServiceManagerName() string {
	if m != nil {
		return m.NetworkServiceManagerName
	}
	return ""
}

func init() {
	proto.RegisterEnum("remote.connection.MechanismType", MechanismType_name, MechanismType_value)
	proto.RegisterEnum("remote.connection.ConnectionEventType", ConnectionEventType_name, ConnectionEventType_value)
	proto.RegisterType((*Mechanism)(nil), "remote.connection.Mechanism")
	proto.RegisterMapType((map[string]string)(nil), "remote.connection.Mechanism.ParametersEntry")
	proto.RegisterType((*Connection)(nil), "remote.connection.Connection")
	proto.RegisterMapType((map[string]string)(nil), "remote.connection.Connection.ContextEntry")
	proto.RegisterMapType((map[string]string)(nil), "remote.connection.Connection.LabelsEntry")
	proto.RegisterType((*ConnectionEvent)(nil), "remote.connection.ConnectionEvent")
	proto.RegisterMapType((map[string]*Connection)(nil), "remote.connection.ConnectionEvent.ConnectionsEntry")
	proto.RegisterType((*MonitorScopeSelector)(nil), "remote.connection.MonitorScopeSelector")
}

func init() { proto.RegisterFile("connection.proto", fileDescriptor_51baa40a1cc6b48b) }

var fileDescriptor_51baa40a1cc6b48b = []byte{
	// 639 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x95, 0xeb, 0x6a, 0xdb, 0x4c,
	0x10, 0x86, 0x23, 0xf9, 0x14, 0x8f, 0x73, 0x50, 0xf6, 0x0b, 0x1f, 0xaa, 0x48, 0xa8, 0x49, 0x4b,
	0xe3, 0x86, 0x62, 0x8a, 0x53, 0x4a, 0x6b, 0x28, 0x45, 0x8d, 0xd5, 0x60, 0x90, 0x55, 0x23, 0x29,
	0x69, 0x29, 0x14, 0xa1, 0xc8, 0x43, 0x23, 0x62, 0xed, 0x1a, 0x69, 0x93, 0xc6, 0xbf, 0x7b, 0x7f,
	0xbd, 0x90, 0x5e, 0x45, 0x91, 0xe4, 0x83, 0xec, 0x38, 0x0a, 0xf9, 0xb7, 0x3b, 0xfb, 0xcc, 0xbb,
	0xda, 0x77, 0x66, 0x57, 0x20, 0x79, 0x8c, 0x52, 0xf4, 0xb8, 0xcf, 0x68, 0x73, 0x14, 0x32, 0xce,
	0xc8, 0x4e, 0x88, 0x01, 0xe3, 0xd8, 0x9c, 0x2f, 0x1c, 0xfc, 0x11, 0xa0, 0xda, 0x43, 0xef, 0xd2,
	0xa5, 0x7e, 0x14, 0x90, 0x37, 0x50, 0xe4, 0xe3, 0x11, 0xca, 0x42, 0x5d, 0x68, 0x6c, 0xb5, 0xea,
	0xcd, 0x3b, 0x7c, 0x73, 0xc6, 0xda, 0xe3, 0x11, 0x9a, 0x09, 0x4d, 0x74, 0x80, 0x91, 0x1b, 0xba,
	0x01, 0x72, 0x0c, 0x23, 0x59, 0xac, 0x17, 0x1a, 0xb5, 0xd6, 0xab, 0xbc, 0xdc, 0x66, 0x7f, 0x86,
	0x6b, 0x94, 0x87, 0x63, 0x33, 0x93, 0xaf, 0x7c, 0x80, 0xed, 0xa5, 0x65, 0x22, 0x41, 0xe1, 0x0a,
	0xc7, 0xc9, 0x57, 0x55, 0xcd, 0x78, 0x48, 0x76, 0xa1, 0x74, 0xe3, 0x0e, 0xaf, 0x51, 0x16, 0x93,
	0x58, 0x3a, 0x69, 0x8b, 0xef, 0x84, 0x83, 0xbf, 0x45, 0x80, 0x93, 0xd9, 0x9e, 0x64, 0x0b, 0x44,
	0x7f, 0x30, 0xc9, 0x14, 0xfd, 0x01, 0x39, 0x84, 0x6d, 0x8a, 0xfc, 0x17, 0x0b, 0xaf, 0x9c, 0x08,
	0xc3, 0x1b, 0xdf, 0x9b, 0x4a, 0x6c, 0x4d, 0xc2, 0x56, 0x1a, 0x25, 0x6d, 0xa8, 0x06, 0xd3, 0xef,
	0x95, 0x0b, 0x75, 0xa1, 0x51, 0x6b, 0xed, 0xe5, 0x9d, 0xc9, 0x9c, 0xe3, 0xa4, 0x03, 0x15, 0x8f,
	0x51, 0x8e, 0xb7, 0x5c, 0x2e, 0x26, 0x6e, 0x1c, 0xad, 0xc8, 0x3c, 0x59, 0x18, 0xc6, 0x70, 0xea,
	0xc5, 0x34, 0x95, 0xa8, 0x50, 0x1e, 0xba, 0x17, 0x38, 0x8c, 0xe4, 0x52, 0x22, 0xf2, 0x32, 0x5f,
	0x44, 0x4f, 0xd8, 0x54, 0x63, 0x92, 0x48, 0x74, 0x78, 0x16, 0xb1, 0xeb, 0xd0, 0x43, 0x67, 0xe9,
	0xd0, 0x4e, 0xe0, 0x52, 0xf7, 0x27, 0x86, 0x0e, 0x75, 0x03, 0x94, 0xcb, 0x89, 0x03, 0x4f, 0x53,
	0xd4, 0x58, 0xf0, 0xa1, 0x97, 0x72, 0x86, 0x1b, 0x20, 0x39, 0x87, 0xc6, 0x00, 0x23, 0xee, 0x53,
	0x37, 0xde, 0x30, 0x5f, 0xb2, 0x92, 0x48, 0x3e, 0xcf, 0xf0, 0xf7, 0xeb, 0xaa, 0xb0, 0xbf, 0xac,
	0x85, 0x74, 0x30, 0x62, 0x3e, 0xe5, 0xa9, 0xd8, 0x7a, 0x22, 0xa6, 0x2c, 0x56, 0x48, 0x9b, 0x20,
	0xb1, 0x84, 0xd2, 0x86, 0x8d, 0xac, 0x89, 0x8f, 0xe9, 0x18, 0xe5, 0x3d, 0xd4, 0x32, 0xde, 0x3d,
	0xaa, 0xd9, 0x7e, 0x8b, 0xb0, 0x3d, 0x2f, 0x81, 0x76, 0x83, 0x94, 0x93, 0xf6, 0xc2, 0x1d, 0x7a,
	0x91, 0x5b, 0xb4, 0x24, 0x23, 0x73, 0x93, 0xce, 0xa0, 0x36, 0xe7, 0xa6, 0x57, 0xe9, 0xf8, 0x61,
	0x89, 0xcc, 0x7c, 0xd2, 0x01, 0x59, 0x1d, 0xe5, 0x07, 0x48, 0xcb, 0xc0, 0x8a, 0x63, 0x1e, 0x67,
	0x8f, 0x59, 0x6b, 0xed, 0xe7, 0x6e, 0x9b, 0x75, 0xe1, 0x2b, 0xec, 0xf6, 0x18, 0xf5, 0x39, 0x0b,
	0x2d, 0x8f, 0x8d, 0xd0, 0xc2, 0x21, 0x7a, 0x9c, 0x85, 0xe4, 0x23, 0xec, 0xe5, 0xf6, 0x48, 0xba,
	0xf7, 0x13, 0x7a, 0x5f, 0x63, 0x1c, 0x5d, 0xc3, 0xe6, 0xc2, 0x7b, 0x43, 0xd6, 0xa1, 0x68, 0x7c,
	0x31, 0x34, 0x69, 0x8d, 0x54, 0xa1, 0x74, 0xfe, 0x4d, 0x57, 0x0d, 0x49, 0x20, 0x9b, 0x50, 0x4d,
	0x86, 0xce, 0x69, 0x5f, 0x93, 0x44, 0x52, 0x81, 0xc2, 0xa9, 0xa9, 0x49, 0x85, 0x18, 0xb6, 0xcc,
	0xf3, 0xb7, 0x52, 0x91, 0xec, 0xc0, 0x66, 0xaf, 0xaf, 0x5b, 0x4c, 0xe3, 0x97, 0x18, 0x52, 0xe4,
	0x52, 0x89, 0x6c, 0xc0, 0x7a, 0x12, 0x8a, 0xd1, 0xf2, 0x6c, 0x76, 0xd6, 0xe9, 0x4b, 0x95, 0xa3,
	0x2e, 0xfc, 0xb7, 0xa2, 0x44, 0x44, 0x81, 0xff, 0xbb, 0x46, 0xd7, 0xee, 0xaa, 0xba, 0x63, 0xd9,
	0xaa, 0xad, 0x39, 0xb6, 0xa9, 0x1a, 0xd6, 0x67, 0xcd, 0x94, 0xd6, 0x08, 0x40, 0xf9, 0xac, 0xdf,
	0x51, 0x6d, 0x4d, 0x12, 0xe2, 0x71, 0x47, 0xd3, 0x35, 0x5b, 0x93, 0xc4, 0xd6, 0x2d, 0xec, 0x4c,
	0xac, 0xc9, 0xbc, 0x49, 0x1e, 0x90, 0x3b, 0xc1, 0x88, 0x1c, 0xae, 0x7a, 0x5d, 0x56, 0xd8, 0xaa,
	0x1c, 0x3c, 0xdc, 0x0f, 0xaf, 0x85, 0x4f, 0x1b, 0xdf, 0x61, 0xbe, 0x7e, 0x51, 0x4e, 0x7e, 0x00,
	0xc7, 0xff, 0x02, 0x00, 0x00, 0xff, 0xff, 0x56, 0xf2, 0x78, 0xaf, 0x14, 0x06, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MonitorConnectionClient is the client API for MonitorConnection service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MonitorConnectionClient interface {
	MonitorConnections(ctx context.Context, in *MonitorScopeSelector, opts ...grpc.CallOption) (MonitorConnection_MonitorConnectionsClient, error)
}

type monitorConnectionClient struct {
	cc *grpc.ClientConn
}

func NewMonitorConnectionClient(cc *grpc.ClientConn) MonitorConnectionClient {
	return &monitorConnectionClient{cc}
}

func (c *monitorConnectionClient) MonitorConnections(ctx context.Context, in *MonitorScopeSelector, opts ...grpc.CallOption) (MonitorConnection_MonitorConnectionsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_MonitorConnection_serviceDesc.Streams[0], "/remote.connection.MonitorConnection/MonitorConnections", opts...)
	if err != nil {
		return nil, err
	}
	x := &monitorConnectionMonitorConnectionsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type MonitorConnection_MonitorConnectionsClient interface {
	Recv() (*ConnectionEvent, error)
	grpc.ClientStream
}

type monitorConnectionMonitorConnectionsClient struct {
	grpc.ClientStream
}

func (x *monitorConnectionMonitorConnectionsClient) Recv() (*ConnectionEvent, error) {
	m := new(ConnectionEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// MonitorConnectionServer is the server API for MonitorConnection service.
type MonitorConnectionServer interface {
	MonitorConnections(*MonitorScopeSelector, MonitorConnection_MonitorConnectionsServer) error
}

func RegisterMonitorConnectionServer(s *grpc.Server, srv MonitorConnectionServer) {
	s.RegisterService(&_MonitorConnection_serviceDesc, srv)
}

func _MonitorConnection_MonitorConnections_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(MonitorScopeSelector)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(MonitorConnectionServer).MonitorConnections(m, &monitorConnectionMonitorConnectionsServer{stream})
}

type MonitorConnection_MonitorConnectionsServer interface {
	Send(*ConnectionEvent) error
	grpc.ServerStream
}

type monitorConnectionMonitorConnectionsServer struct {
	grpc.ServerStream
}

func (x *monitorConnectionMonitorConnectionsServer) Send(m *ConnectionEvent) error {
	return x.ServerStream.SendMsg(m)
}

var _MonitorConnection_serviceDesc = grpc.ServiceDesc{
	ServiceName: "remote.connection.MonitorConnection",
	HandlerType: (*MonitorConnectionServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "MonitorConnections",
			Handler:       _MonitorConnection_MonitorConnections_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "connection.proto",
}
