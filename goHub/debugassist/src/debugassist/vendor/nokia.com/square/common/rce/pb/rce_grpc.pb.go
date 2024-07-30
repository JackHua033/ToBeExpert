// Copyright 2017 Square, Inc.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.24.0
// source: rce.proto

package pb

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
	RCEAgent_Start_FullMethodName     = "/rce.RCEAgent/Start"
	RCEAgent_Wait_FullMethodName      = "/rce.RCEAgent/Wait"
	RCEAgent_GetStatus_FullMethodName = "/rce.RCEAgent/GetStatus"
	RCEAgent_Stop_FullMethodName      = "/rce.RCEAgent/Stop"
	RCEAgent_Running_FullMethodName   = "/rce.RCEAgent/Running"
)

// RCEAgentClient is the client API for RCEAgent service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RCEAgentClient interface {
	// Start a command and immediately return its ID. Be sure to call Wait or Stop
	// to reap the command, else the agent will effectively leak memory by holding
	// unreaped commands. A command is considered running until reaped.
	Start(ctx context.Context, in *Command, opts ...grpc.CallOption) (*ID, error)
	// Wait for a command to complete or be stopped, reap it, and return its final status.
	Wait(ctx context.Context, in *ID, opts ...grpc.CallOption) (*Status, error)
	// Get the status of a command if it hasn't been reaped by calling Wait or Stop.
	GetStatus(ctx context.Context, in *ID, opts ...grpc.CallOption) (*Status, error)
	// Stop then reap a command by sending it a SIGTERM signal.
	Stop(ctx context.Context, in *ID, opts ...grpc.CallOption) (*Empty, error)
	// Return a list of all running (not reaped) commands by ID.
	Running(ctx context.Context, in *Empty, opts ...grpc.CallOption) (RCEAgent_RunningClient, error)
}

type rCEAgentClient struct {
	cc grpc.ClientConnInterface
}

func NewRCEAgentClient(cc grpc.ClientConnInterface) RCEAgentClient {
	return &rCEAgentClient{cc}
}

func (c *rCEAgentClient) Start(ctx context.Context, in *Command, opts ...grpc.CallOption) (*ID, error) {
	out := new(ID)
	err := c.cc.Invoke(ctx, RCEAgent_Start_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rCEAgentClient) Wait(ctx context.Context, in *ID, opts ...grpc.CallOption) (*Status, error) {
	out := new(Status)
	err := c.cc.Invoke(ctx, RCEAgent_Wait_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rCEAgentClient) GetStatus(ctx context.Context, in *ID, opts ...grpc.CallOption) (*Status, error) {
	out := new(Status)
	err := c.cc.Invoke(ctx, RCEAgent_GetStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rCEAgentClient) Stop(ctx context.Context, in *ID, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, RCEAgent_Stop_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rCEAgentClient) Running(ctx context.Context, in *Empty, opts ...grpc.CallOption) (RCEAgent_RunningClient, error) {
	stream, err := c.cc.NewStream(ctx, &RCEAgent_ServiceDesc.Streams[0], RCEAgent_Running_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &rCEAgentRunningClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type RCEAgent_RunningClient interface {
	Recv() (*ID, error)
	grpc.ClientStream
}

type rCEAgentRunningClient struct {
	grpc.ClientStream
}

func (x *rCEAgentRunningClient) Recv() (*ID, error) {
	m := new(ID)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// RCEAgentServer is the server API for RCEAgent service.
// All implementations must embed UnimplementedRCEAgentServer
// for forward compatibility
type RCEAgentServer interface {
	// Start a command and immediately return its ID. Be sure to call Wait or Stop
	// to reap the command, else the agent will effectively leak memory by holding
	// unreaped commands. A command is considered running until reaped.
	Start(context.Context, *Command) (*ID, error)
	// Wait for a command to complete or be stopped, reap it, and return its final status.
	Wait(context.Context, *ID) (*Status, error)
	// Get the status of a command if it hasn't been reaped by calling Wait or Stop.
	GetStatus(context.Context, *ID) (*Status, error)
	// Stop then reap a command by sending it a SIGTERM signal.
	Stop(context.Context, *ID) (*Empty, error)
	// Return a list of all running (not reaped) commands by ID.
	Running(*Empty, RCEAgent_RunningServer) error
	mustEmbedUnimplementedRCEAgentServer()
}

// UnimplementedRCEAgentServer must be embedded to have forward compatible implementations.
type UnimplementedRCEAgentServer struct {
}

func (UnimplementedRCEAgentServer) Start(context.Context, *Command) (*ID, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedRCEAgentServer) Wait(context.Context, *ID) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Wait not implemented")
}
func (UnimplementedRCEAgentServer) GetStatus(context.Context, *ID) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedRCEAgentServer) Stop(context.Context, *ID) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
func (UnimplementedRCEAgentServer) Running(*Empty, RCEAgent_RunningServer) error {
	return status.Errorf(codes.Unimplemented, "method Running not implemented")
}
func (UnimplementedRCEAgentServer) mustEmbedUnimplementedRCEAgentServer() {}

// UnsafeRCEAgentServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RCEAgentServer will
// result in compilation errors.
type UnsafeRCEAgentServer interface {
	mustEmbedUnimplementedRCEAgentServer()
}

func RegisterRCEAgentServer(s grpc.ServiceRegistrar, srv RCEAgentServer) {
	s.RegisterService(&RCEAgent_ServiceDesc, srv)
}

func _RCEAgent_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Command)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RCEAgentServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RCEAgent_Start_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RCEAgentServer).Start(ctx, req.(*Command))
	}
	return interceptor(ctx, in, info, handler)
}

func _RCEAgent_Wait_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RCEAgentServer).Wait(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RCEAgent_Wait_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RCEAgentServer).Wait(ctx, req.(*ID))
	}
	return interceptor(ctx, in, info, handler)
}

func _RCEAgent_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RCEAgentServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RCEAgent_GetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RCEAgentServer).GetStatus(ctx, req.(*ID))
	}
	return interceptor(ctx, in, info, handler)
}

func _RCEAgent_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ID)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RCEAgentServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RCEAgent_Stop_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RCEAgentServer).Stop(ctx, req.(*ID))
	}
	return interceptor(ctx, in, info, handler)
}

func _RCEAgent_Running_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(RCEAgentServer).Running(m, &rCEAgentRunningServer{stream})
}

type RCEAgent_RunningServer interface {
	Send(*ID) error
	grpc.ServerStream
}

type rCEAgentRunningServer struct {
	grpc.ServerStream
}

func (x *rCEAgentRunningServer) Send(m *ID) error {
	return x.ServerStream.SendMsg(m)
}

// RCEAgent_ServiceDesc is the grpc.ServiceDesc for RCEAgent service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RCEAgent_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rce.RCEAgent",
	HandlerType: (*RCEAgentServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Start",
			Handler:    _RCEAgent_Start_Handler,
		},
		{
			MethodName: "Wait",
			Handler:    _RCEAgent_Wait_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _RCEAgent_GetStatus_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _RCEAgent_Stop_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Running",
			Handler:       _RCEAgent_Running_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "rce.proto",
}