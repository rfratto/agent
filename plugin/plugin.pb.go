// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: plugin/plugin.proto

package plugin

import (
	context "context"
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	io "io"
	math "math"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// LogMessage is a plaintext message to send to the host for it to log.
type LogMessage struct {
	Content string `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
}

func (m *LogMessage) Reset()      { *m = LogMessage{} }
func (*LogMessage) ProtoMessage() {}
func (*LogMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_5f7429f2a742b54b, []int{0}
}
func (m *LogMessage) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LogMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LogMessage.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *LogMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogMessage.Merge(m, src)
}
func (m *LogMessage) XXX_Size() int {
	return m.Size()
}
func (m *LogMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_LogMessage.DiscardUnknown(m)
}

var xxx_messageInfo_LogMessage proto.InternalMessageInfo

func (m *LogMessage) GetContent() string {
	if m != nil {
		return m.Content
	}
	return ""
}

// InitializeRequest is a payload provided by a Host when starting the plugin.
type InitializeRequest struct {
	// host_addr can be used to connect to the Host as a client for logging and
	// sending other types of messages.
	//
	// This host_addr may be one of the following protocols:
	//
	// - tcp://: use TCP to connect to the host
	// - plugin://: use stdin/stdout to communicate with the host. This is a special
	//   protocol used by github.com/grafana/agent/plugin.
	HostAddr string `protobuf:"bytes,1,opt,name=host_addr,json=hostAddr,proto3" json:"host_addr,omitempty"`
}

func (m *InitializeRequest) Reset()      { *m = InitializeRequest{} }
func (*InitializeRequest) ProtoMessage() {}
func (*InitializeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5f7429f2a742b54b, []int{1}
}
func (m *InitializeRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InitializeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_InitializeRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *InitializeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InitializeRequest.Merge(m, src)
}
func (m *InitializeRequest) XXX_Size() int {
	return m.Size()
}
func (m *InitializeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_InitializeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_InitializeRequest proto.InternalMessageInfo

func (m *InitializeRequest) GetHostAddr() string {
	if m != nil {
		return m.HostAddr
	}
	return ""
}

// InitializeResponse is the result of an Initialize rpc call. It can provide
// extra information to the host to hook up extra events.
type InitializeResponse struct {
	// HTTP path prefixes for HTTP requests that should be routed to this plugin.
	// This requires that the plugin service also implements httpgrpc.HTTP:
	//
	// https://github.com/weaveworks/common/blob/master/httpgrpc/httpgrpc.proto.
	//
	// Prefixes should end in a /.
	//
	// Registering the same path prefix that is already registered by another plugin
	// is considered invalid, and will cause the host to terminate the plugin. To avoid
	// this, use path prefixes that contain the plugin name, such as
	// /agent/api/v1/prometheus/.
	HttpPathPrefixes []string `protobuf:"bytes,1,rep,name=http_path_prefixes,json=httpPathPrefixes,proto3" json:"http_path_prefixes,omitempty"`
	// A series of gRPC services to proxy requests for. The name to use here should be
	// <package name>.<service name>. For example, the name for the Plugin service is
	// plugin.Plugin.
	//
	// Only proxy plugin-specific services for plugin-specific behavior. Registering the
	// same service already registered by another plugin is invalid, and will cause the
	// host to terminate the plugin.
	GrpcServiceProxy []string `protobuf:"bytes,2,rep,name=grpc_service_proxy,json=grpcServiceProxy,proto3" json:"grpc_service_proxy,omitempty"`
}

func (m *InitializeResponse) Reset()      { *m = InitializeResponse{} }
func (*InitializeResponse) ProtoMessage() {}
func (*InitializeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_5f7429f2a742b54b, []int{2}
}
func (m *InitializeResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InitializeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_InitializeResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *InitializeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InitializeResponse.Merge(m, src)
}
func (m *InitializeResponse) XXX_Size() int {
	return m.Size()
}
func (m *InitializeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_InitializeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_InitializeResponse proto.InternalMessageInfo

func (m *InitializeResponse) GetHttpPathPrefixes() []string {
	if m != nil {
		return m.HttpPathPrefixes
	}
	return nil
}

func (m *InitializeResponse) GetGrpcServiceProxy() []string {
	if m != nil {
		return m.GrpcServiceProxy
	}
	return nil
}

type StartRequest struct {
	// Config is any config string that can be used to initialize the plugin.
	Config string `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
}

func (m *StartRequest) Reset()      { *m = StartRequest{} }
func (*StartRequest) ProtoMessage() {}
func (*StartRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5f7429f2a742b54b, []int{3}
}
func (m *StartRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *StartRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_StartRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *StartRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StartRequest.Merge(m, src)
}
func (m *StartRequest) XXX_Size() int {
	return m.Size()
}
func (m *StartRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_StartRequest.DiscardUnknown(m)
}

var xxx_messageInfo_StartRequest proto.InternalMessageInfo

func (m *StartRequest) GetConfig() string {
	if m != nil {
		return m.Config
	}
	return ""
}

type StopRequest struct {
}

func (m *StopRequest) Reset()      { *m = StopRequest{} }
func (*StopRequest) ProtoMessage() {}
func (*StopRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5f7429f2a742b54b, []int{4}
}
func (m *StopRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *StopRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_StopRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *StopRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StopRequest.Merge(m, src)
}
func (m *StopRequest) XXX_Size() int {
	return m.Size()
}
func (m *StopRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_StopRequest.DiscardUnknown(m)
}

var xxx_messageInfo_StopRequest proto.InternalMessageInfo

func init() {
	proto.RegisterType((*LogMessage)(nil), "plugin.LogMessage")
	proto.RegisterType((*InitializeRequest)(nil), "plugin.InitializeRequest")
	proto.RegisterType((*InitializeResponse)(nil), "plugin.InitializeResponse")
	proto.RegisterType((*StartRequest)(nil), "plugin.StartRequest")
	proto.RegisterType((*StopRequest)(nil), "plugin.StopRequest")
}

func init() { proto.RegisterFile("plugin/plugin.proto", fileDescriptor_5f7429f2a742b54b) }

var fileDescriptor_5f7429f2a742b54b = []byte{
	// 400 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x90, 0xcf, 0xae, 0xd2, 0x40,
	0x14, 0xc6, 0x3b, 0x82, 0x55, 0x8e, 0x9a, 0xe8, 0x60, 0x48, 0x2d, 0xc9, 0x84, 0x74, 0x41, 0x58,
	0x98, 0xa2, 0xa0, 0x0b, 0x97, 0x6a, 0x4c, 0x34, 0xc1, 0xa4, 0x81, 0x07, 0x68, 0x4a, 0x3b, 0x4c,
	0x9b, 0x60, 0x67, 0x9c, 0x19, 0x0c, 0xb8, 0xf2, 0x11, 0x7c, 0x0c, 0x5f, 0xc1, 0x37, 0x70, 0xc9,
	0x92, 0xa5, 0x94, 0xcd, 0x5d, 0xf2, 0x08, 0x37, 0xfd, 0x77, 0x21, 0xb9, 0x97, 0xd5, 0xe4, 0x9c,
	0xef, 0xfb, 0x26, 0xe7, 0xfb, 0x41, 0x5b, 0x2c, 0x57, 0x2c, 0x49, 0x87, 0xe5, 0xe3, 0x0a, 0xc9,
	0x35, 0xc7, 0x66, 0x39, 0xd9, 0x5d, 0xc6, 0x39, 0x5b, 0xd2, 0x61, 0xb1, 0x9d, 0xaf, 0x16, 0x43,
	0xfa, 0x4d, 0xe8, 0x4d, 0x69, 0x72, 0xfa, 0x00, 0x13, 0xce, 0xbe, 0x52, 0xa5, 0x02, 0x46, 0xb1,
	0x05, 0x0f, 0x42, 0x9e, 0x6a, 0x9a, 0x6a, 0x0b, 0xf5, 0xd0, 0xa0, 0x35, 0xad, 0x47, 0xe7, 0x15,
	0x3c, 0xfb, 0x92, 0x26, 0x3a, 0x09, 0x96, 0xc9, 0x4f, 0x3a, 0xa5, 0xdf, 0x57, 0x54, 0x69, 0xdc,
	0x85, 0x56, 0xcc, 0x95, 0xf6, 0x83, 0x28, 0x92, 0x55, 0xe0, 0x61, 0xbe, 0x78, 0x1f, 0x45, 0xd2,
	0x11, 0x80, 0xcf, 0x13, 0x4a, 0xf0, 0x54, 0x51, 0xfc, 0x12, 0x70, 0xac, 0xb5, 0xf0, 0x45, 0xa0,
	0x63, 0x5f, 0x48, 0xba, 0x48, 0xd6, 0x54, 0x59, 0xa8, 0xd7, 0x18, 0xb4, 0xa6, 0x4f, 0x73, 0xc5,
	0x0b, 0x74, 0xec, 0x55, 0xfb, 0xdc, 0xcd, 0xa4, 0x08, 0x7d, 0x45, 0xe5, 0x8f, 0x24, 0xa4, 0xbe,
	0x90, 0x7c, 0xbd, 0xb1, 0xee, 0x95, 0xee, 0x5c, 0x99, 0x95, 0x82, 0x97, 0xef, 0x9d, 0x3e, 0x3c,
	0x9e, 0xe9, 0x40, 0xea, 0xfa, 0xbc, 0x0e, 0x98, 0x21, 0x4f, 0x17, 0x09, 0xab, 0x6e, 0xab, 0x26,
	0xe7, 0x09, 0x3c, 0x9a, 0x69, 0x2e, 0x2a, 0xdb, 0xe8, 0x1d, 0x34, 0x3f, 0x73, 0xa5, 0xf1, 0x6b,
	0x68, 0x4c, 0x38, 0xc3, 0xd8, 0xad, 0x28, 0x9e, 0xb8, 0xd8, 0x1d, 0xb7, 0x64, 0xe8, 0xd6, 0x0c,
	0xdd, 0x4f, 0x39, 0xc3, 0xd1, 0x5f, 0x04, 0xa6, 0x57, 0xb8, 0xf1, 0x47, 0x80, 0x53, 0x5d, 0xfc,
	0xa2, 0xfe, 0xe4, 0x16, 0x34, 0xdb, 0xbe, 0x4b, 0xaa, 0xe8, 0xbc, 0x85, 0xfb, 0x45, 0x03, 0xfc,
	0xbc, 0x36, 0x9d, 0x17, 0xba, 0x74, 0x06, 0x1e, 0x43, 0x33, 0x2f, 0x84, 0xdb, 0xa7, 0xd4, 0x4d,
	0xbd, 0x4b, 0xa1, 0x0f, 0x6f, 0xb6, 0x7b, 0x62, 0xec, 0xf6, 0xc4, 0x38, 0xee, 0x09, 0xfa, 0x95,
	0x11, 0xf4, 0x27, 0x23, 0xe8, 0x5f, 0x46, 0xd0, 0x36, 0x23, 0xe8, 0x7f, 0x46, 0xd0, 0x55, 0x46,
	0x8c, 0x63, 0x46, 0xd0, 0xef, 0x03, 0x31, 0xb6, 0x07, 0x62, 0xec, 0x0e, 0xc4, 0x98, 0x9b, 0xc5,
	0x2f, 0xe3, 0xeb, 0x00, 0x00, 0x00, 0xff, 0xff, 0x64, 0x8f, 0xee, 0xe7, 0x72, 0x02, 0x00, 0x00,
}

func (this *LogMessage) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*LogMessage)
	if !ok {
		that2, ok := that.(LogMessage)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Content != that1.Content {
		return false
	}
	return true
}
func (this *InitializeRequest) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*InitializeRequest)
	if !ok {
		that2, ok := that.(InitializeRequest)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.HostAddr != that1.HostAddr {
		return false
	}
	return true
}
func (this *InitializeResponse) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*InitializeResponse)
	if !ok {
		that2, ok := that.(InitializeResponse)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if len(this.HttpPathPrefixes) != len(that1.HttpPathPrefixes) {
		return false
	}
	for i := range this.HttpPathPrefixes {
		if this.HttpPathPrefixes[i] != that1.HttpPathPrefixes[i] {
			return false
		}
	}
	if len(this.GrpcServiceProxy) != len(that1.GrpcServiceProxy) {
		return false
	}
	for i := range this.GrpcServiceProxy {
		if this.GrpcServiceProxy[i] != that1.GrpcServiceProxy[i] {
			return false
		}
	}
	return true
}
func (this *StartRequest) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*StartRequest)
	if !ok {
		that2, ok := that.(StartRequest)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Config != that1.Config {
		return false
	}
	return true
}
func (this *StopRequest) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*StopRequest)
	if !ok {
		that2, ok := that.(StopRequest)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	return true
}
func (this *LogMessage) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&plugin.LogMessage{")
	s = append(s, "Content: "+fmt.Sprintf("%#v", this.Content)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *InitializeRequest) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&plugin.InitializeRequest{")
	s = append(s, "HostAddr: "+fmt.Sprintf("%#v", this.HostAddr)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *InitializeResponse) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 6)
	s = append(s, "&plugin.InitializeResponse{")
	s = append(s, "HttpPathPrefixes: "+fmt.Sprintf("%#v", this.HttpPathPrefixes)+",\n")
	s = append(s, "GrpcServiceProxy: "+fmt.Sprintf("%#v", this.GrpcServiceProxy)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *StartRequest) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&plugin.StartRequest{")
	s = append(s, "Config: "+fmt.Sprintf("%#v", this.Config)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *StopRequest) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 4)
	s = append(s, "&plugin.StopRequest{")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringPlugin(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// HostClient is the client API for Host service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type HostClient interface {
	Log(ctx context.Context, in *LogMessage, opts ...grpc.CallOption) (*empty.Empty, error)
}

type hostClient struct {
	cc *grpc.ClientConn
}

func NewHostClient(cc *grpc.ClientConn) HostClient {
	return &hostClient{cc}
}

func (c *hostClient) Log(ctx context.Context, in *LogMessage, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/plugin.Host/Log", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HostServer is the server API for Host service.
type HostServer interface {
	Log(context.Context, *LogMessage) (*empty.Empty, error)
}

func RegisterHostServer(s *grpc.Server, srv HostServer) {
	s.RegisterService(&_Host_serviceDesc, srv)
}

func _Host_Log_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HostServer).Log(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/plugin.Host/Log",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HostServer).Log(ctx, req.(*LogMessage))
	}
	return interceptor(ctx, in, info, handler)
}

var _Host_serviceDesc = grpc.ServiceDesc{
	ServiceName: "plugin.Host",
	HandlerType: (*HostServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Log",
			Handler:    _Host_Log_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "plugin/plugin.proto",
}

// PluginClient is the client API for Plugin service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type PluginClient interface {
	// Initialize is called by a Host when a plugin is first loaded. It should
	// not start the plugin.
	Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*InitializeResponse, error)
	// Stop is a control message to the plugin to start its service system.
	Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Stop is a controll message to the plugin to stop its service system. When
	// Stop is called, the plugin should shut down. It is valid to call Stop before
	// calling Start.
	Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*empty.Empty, error)
}

type pluginClient struct {
	cc *grpc.ClientConn
}

func NewPluginClient(cc *grpc.ClientConn) PluginClient {
	return &pluginClient{cc}
}

func (c *pluginClient) Initialize(ctx context.Context, in *InitializeRequest, opts ...grpc.CallOption) (*InitializeResponse, error) {
	out := new(InitializeResponse)
	err := c.cc.Invoke(ctx, "/plugin.Plugin/Initialize", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/plugin.Plugin/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/plugin.Plugin/Stop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PluginServer is the server API for Plugin service.
type PluginServer interface {
	// Initialize is called by a Host when a plugin is first loaded. It should
	// not start the plugin.
	Initialize(context.Context, *InitializeRequest) (*InitializeResponse, error)
	// Stop is a control message to the plugin to start its service system.
	Start(context.Context, *StartRequest) (*empty.Empty, error)
	// Stop is a controll message to the plugin to stop its service system. When
	// Stop is called, the plugin should shut down. It is valid to call Stop before
	// calling Start.
	Stop(context.Context, *StopRequest) (*empty.Empty, error)
}

func RegisterPluginServer(s *grpc.Server, srv PluginServer) {
	s.RegisterService(&_Plugin_serviceDesc, srv)
}

func _Plugin_Initialize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitializeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).Initialize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/plugin.Plugin/Initialize",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).Initialize(ctx, req.(*InitializeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/plugin.Plugin/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).Start(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/plugin.Plugin/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).Stop(ctx, req.(*StopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Plugin_serviceDesc = grpc.ServiceDesc{
	ServiceName: "plugin.Plugin",
	HandlerType: (*PluginServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Initialize",
			Handler:    _Plugin_Initialize_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _Plugin_Start_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _Plugin_Stop_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "plugin/plugin.proto",
}

func (m *LogMessage) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LogMessage) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Content) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintPlugin(dAtA, i, uint64(len(m.Content)))
		i += copy(dAtA[i:], m.Content)
	}
	return i, nil
}

func (m *InitializeRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InitializeRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.HostAddr) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintPlugin(dAtA, i, uint64(len(m.HostAddr)))
		i += copy(dAtA[i:], m.HostAddr)
	}
	return i, nil
}

func (m *InitializeResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InitializeResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.HttpPathPrefixes) > 0 {
		for _, s := range m.HttpPathPrefixes {
			dAtA[i] = 0xa
			i++
			l = len(s)
			for l >= 1<<7 {
				dAtA[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			dAtA[i] = uint8(l)
			i++
			i += copy(dAtA[i:], s)
		}
	}
	if len(m.GrpcServiceProxy) > 0 {
		for _, s := range m.GrpcServiceProxy {
			dAtA[i] = 0x12
			i++
			l = len(s)
			for l >= 1<<7 {
				dAtA[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			dAtA[i] = uint8(l)
			i++
			i += copy(dAtA[i:], s)
		}
	}
	return i, nil
}

func (m *StartRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *StartRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Config) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintPlugin(dAtA, i, uint64(len(m.Config)))
		i += copy(dAtA[i:], m.Config)
	}
	return i, nil
}

func (m *StopRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *StopRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	return i, nil
}

func encodeVarintPlugin(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *LogMessage) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Content)
	if l > 0 {
		n += 1 + l + sovPlugin(uint64(l))
	}
	return n
}

func (m *InitializeRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.HostAddr)
	if l > 0 {
		n += 1 + l + sovPlugin(uint64(l))
	}
	return n
}

func (m *InitializeResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.HttpPathPrefixes) > 0 {
		for _, s := range m.HttpPathPrefixes {
			l = len(s)
			n += 1 + l + sovPlugin(uint64(l))
		}
	}
	if len(m.GrpcServiceProxy) > 0 {
		for _, s := range m.GrpcServiceProxy {
			l = len(s)
			n += 1 + l + sovPlugin(uint64(l))
		}
	}
	return n
}

func (m *StartRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Config)
	if l > 0 {
		n += 1 + l + sovPlugin(uint64(l))
	}
	return n
}

func (m *StopRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovPlugin(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozPlugin(x uint64) (n int) {
	return sovPlugin(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *LogMessage) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&LogMessage{`,
		`Content:` + fmt.Sprintf("%v", this.Content) + `,`,
		`}`,
	}, "")
	return s
}
func (this *InitializeRequest) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&InitializeRequest{`,
		`HostAddr:` + fmt.Sprintf("%v", this.HostAddr) + `,`,
		`}`,
	}, "")
	return s
}
func (this *InitializeResponse) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&InitializeResponse{`,
		`HttpPathPrefixes:` + fmt.Sprintf("%v", this.HttpPathPrefixes) + `,`,
		`GrpcServiceProxy:` + fmt.Sprintf("%v", this.GrpcServiceProxy) + `,`,
		`}`,
	}, "")
	return s
}
func (this *StartRequest) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&StartRequest{`,
		`Config:` + fmt.Sprintf("%v", this.Config) + `,`,
		`}`,
	}, "")
	return s
}
func (this *StopRequest) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&StopRequest{`,
		`}`,
	}, "")
	return s
}
func valueToStringPlugin(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *LogMessage) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPlugin
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: LogMessage: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LogMessage: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Content", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlugin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPlugin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlugin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Content = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPlugin(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *InitializeRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPlugin
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InitializeRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InitializeRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field HostAddr", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlugin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPlugin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlugin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.HostAddr = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPlugin(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *InitializeResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPlugin
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InitializeResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InitializeResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field HttpPathPrefixes", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlugin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPlugin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlugin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.HttpPathPrefixes = append(m.HttpPathPrefixes, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field GrpcServiceProxy", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlugin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPlugin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlugin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.GrpcServiceProxy = append(m.GrpcServiceProxy, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPlugin(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *StartRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPlugin
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: StartRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: StartRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Config", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowPlugin
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthPlugin
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthPlugin
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Config = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipPlugin(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *StopRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowPlugin
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: StopRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: StopRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipPlugin(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthPlugin
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipPlugin(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowPlugin
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowPlugin
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowPlugin
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthPlugin
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthPlugin
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowPlugin
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipPlugin(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthPlugin
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthPlugin = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowPlugin   = fmt.Errorf("proto: integer overflow")
)
