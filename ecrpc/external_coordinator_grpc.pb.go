// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: ecrpc/external_coordinator.proto

package ecrpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ExternalCoordinator_RegisterMissionControl_FullMethodName        = "/ecrpc.ExternalCoordinator/RegisterMissionControl"
	ExternalCoordinator_QueryAggregatedMissionControl_FullMethodName = "/ecrpc.ExternalCoordinator/QueryAggregatedMissionControl"
)

// ExternalCoordinatorClient is the client API for ExternalCoordinator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ExternalCoordinatorClient interface {
	// RegisterMissionControl registers mission control data.
	RegisterMissionControl(ctx context.Context, in *RegisterMissionControlRequest, opts ...grpc.CallOption) (*RegisterMissionControlResponse, error)
	// QueryAggregatedMissionControl queries aggregated mission control data.
	QueryAggregatedMissionControl(ctx context.Context, in *QueryAggregatedMissionControlRequest, opts ...grpc.CallOption) (ExternalCoordinator_QueryAggregatedMissionControlClient, error)
}

type externalCoordinatorClient struct {
	cc grpc.ClientConnInterface
}

func NewExternalCoordinatorClient(cc grpc.ClientConnInterface) ExternalCoordinatorClient {
	return &externalCoordinatorClient{cc}
}

func (c *externalCoordinatorClient) RegisterMissionControl(ctx context.Context, in *RegisterMissionControlRequest, opts ...grpc.CallOption) (*RegisterMissionControlResponse, error) {
	out := new(RegisterMissionControlResponse)
	err := c.cc.Invoke(ctx, ExternalCoordinator_RegisterMissionControl_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *externalCoordinatorClient) QueryAggregatedMissionControl(ctx context.Context, in *QueryAggregatedMissionControlRequest, opts ...grpc.CallOption) (ExternalCoordinator_QueryAggregatedMissionControlClient, error) {
	stream, err := c.cc.NewStream(ctx, &ExternalCoordinator_ServiceDesc.Streams[0], ExternalCoordinator_QueryAggregatedMissionControl_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &externalCoordinatorQueryAggregatedMissionControlClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ExternalCoordinator_QueryAggregatedMissionControlClient interface {
	Recv() (*QueryAggregatedMissionControlResponse, error)
	grpc.ClientStream
}

type externalCoordinatorQueryAggregatedMissionControlClient struct {
	grpc.ClientStream
}

func (x *externalCoordinatorQueryAggregatedMissionControlClient) Recv() (*QueryAggregatedMissionControlResponse, error) {
	m := new(QueryAggregatedMissionControlResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ExternalCoordinatorServer is the server API for ExternalCoordinator service.
// All implementations must embed UnimplementedExternalCoordinatorServer
// for forward compatibility
type ExternalCoordinatorServer interface {
	// RegisterMissionControl registers mission control data.
	RegisterMissionControl(context.Context, *RegisterMissionControlRequest) (*RegisterMissionControlResponse, error)
	// QueryAggregatedMissionControl queries aggregated mission control data.
	QueryAggregatedMissionControl(*QueryAggregatedMissionControlRequest, ExternalCoordinator_QueryAggregatedMissionControlServer) error
	mustEmbedUnimplementedExternalCoordinatorServer()
}

// UnimplementedExternalCoordinatorServer must be embedded to have forward compatible implementations.
type UnimplementedExternalCoordinatorServer struct {
}

func (UnimplementedExternalCoordinatorServer) RegisterMissionControl(context.Context, *RegisterMissionControlRequest) (*RegisterMissionControlResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterMissionControl not implemented")
}
func (UnimplementedExternalCoordinatorServer) QueryAggregatedMissionControl(*QueryAggregatedMissionControlRequest, ExternalCoordinator_QueryAggregatedMissionControlServer) error {
	return status.Errorf(codes.Unimplemented, "method QueryAggregatedMissionControl not implemented")
}
func (UnimplementedExternalCoordinatorServer) mustEmbedUnimplementedExternalCoordinatorServer() {}

// UnsafeExternalCoordinatorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ExternalCoordinatorServer will
// result in compilation errors.
type UnsafeExternalCoordinatorServer interface {
	mustEmbedUnimplementedExternalCoordinatorServer()
}

func RegisterExternalCoordinatorServer(s grpc.ServiceRegistrar, srv ExternalCoordinatorServer) {
	s.RegisterService(&ExternalCoordinator_ServiceDesc, srv)
}

func _ExternalCoordinator_RegisterMissionControl_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterMissionControlRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExternalCoordinatorServer).RegisterMissionControl(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ExternalCoordinator_RegisterMissionControl_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExternalCoordinatorServer).RegisterMissionControl(ctx, req.(*RegisterMissionControlRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ExternalCoordinator_QueryAggregatedMissionControl_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(QueryAggregatedMissionControlRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ExternalCoordinatorServer).QueryAggregatedMissionControl(m, &externalCoordinatorQueryAggregatedMissionControlServer{stream})
}

type ExternalCoordinator_QueryAggregatedMissionControlServer interface {
	Send(*QueryAggregatedMissionControlResponse) error
	grpc.ServerStream
}

type externalCoordinatorQueryAggregatedMissionControlServer struct {
	grpc.ServerStream
}

func (x *externalCoordinatorQueryAggregatedMissionControlServer) Send(m *QueryAggregatedMissionControlResponse) error {
	return x.ServerStream.SendMsg(m)
}

// ExternalCoordinator_ServiceDesc is the grpc.ServiceDesc for ExternalCoordinator service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ExternalCoordinator_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ecrpc.ExternalCoordinator",
	HandlerType: (*ExternalCoordinatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterMissionControl",
			Handler:    _ExternalCoordinator_RegisterMissionControl_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "QueryAggregatedMissionControl",
			Handler:       _ExternalCoordinator_QueryAggregatedMissionControl_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "ecrpc/external_coordinator.proto",
}
