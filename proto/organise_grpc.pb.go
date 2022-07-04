// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// OrganiserServiceClient is the client API for OrganiserService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type OrganiserServiceClient interface {
	AddLocation(ctx context.Context, in *AddLocationRequest, opts ...grpc.CallOption) (*AddLocationResponse, error)
	GetOrganisation(ctx context.Context, in *GetOrganisationRequest, opts ...grpc.CallOption) (*GetOrganisationResponse, error)
	UpdateLocation(ctx context.Context, in *UpdateLocationRequest, opts ...grpc.CallOption) (*UpdateLocationResponse, error)
	Locate(ctx context.Context, in *LocateRequest, opts ...grpc.CallOption) (*LocateResponse, error)
	GetQuota(ctx context.Context, in *QuotaRequest, opts ...grpc.CallOption) (*QuotaResponse, error)
	AddExtractor(ctx context.Context, in *AddExtractorRequest, opts ...grpc.CallOption) (*AddExtractorResponse, error)
	GetCache(ctx context.Context, in *GetCacheRequest, opts ...grpc.CallOption) (*GetCacheResponse, error)
}

type organiserServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewOrganiserServiceClient(cc grpc.ClientConnInterface) OrganiserServiceClient {
	return &organiserServiceClient{cc}
}

func (c *organiserServiceClient) AddLocation(ctx context.Context, in *AddLocationRequest, opts ...grpc.CallOption) (*AddLocationResponse, error) {
	out := new(AddLocationResponse)
	err := c.cc.Invoke(ctx, "/recordsorganiser.OrganiserService/AddLocation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organiserServiceClient) GetOrganisation(ctx context.Context, in *GetOrganisationRequest, opts ...grpc.CallOption) (*GetOrganisationResponse, error) {
	out := new(GetOrganisationResponse)
	err := c.cc.Invoke(ctx, "/recordsorganiser.OrganiserService/GetOrganisation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organiserServiceClient) UpdateLocation(ctx context.Context, in *UpdateLocationRequest, opts ...grpc.CallOption) (*UpdateLocationResponse, error) {
	out := new(UpdateLocationResponse)
	err := c.cc.Invoke(ctx, "/recordsorganiser.OrganiserService/UpdateLocation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organiserServiceClient) Locate(ctx context.Context, in *LocateRequest, opts ...grpc.CallOption) (*LocateResponse, error) {
	out := new(LocateResponse)
	err := c.cc.Invoke(ctx, "/recordsorganiser.OrganiserService/Locate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organiserServiceClient) GetQuota(ctx context.Context, in *QuotaRequest, opts ...grpc.CallOption) (*QuotaResponse, error) {
	out := new(QuotaResponse)
	err := c.cc.Invoke(ctx, "/recordsorganiser.OrganiserService/GetQuota", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organiserServiceClient) AddExtractor(ctx context.Context, in *AddExtractorRequest, opts ...grpc.CallOption) (*AddExtractorResponse, error) {
	out := new(AddExtractorResponse)
	err := c.cc.Invoke(ctx, "/recordsorganiser.OrganiserService/AddExtractor", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *organiserServiceClient) GetCache(ctx context.Context, in *GetCacheRequest, opts ...grpc.CallOption) (*GetCacheResponse, error) {
	out := new(GetCacheResponse)
	err := c.cc.Invoke(ctx, "/recordsorganiser.OrganiserService/GetCache", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OrganiserServiceServer is the server API for OrganiserService service.
// All implementations should embed UnimplementedOrganiserServiceServer
// for forward compatibility
type OrganiserServiceServer interface {
	AddLocation(context.Context, *AddLocationRequest) (*AddLocationResponse, error)
	GetOrganisation(context.Context, *GetOrganisationRequest) (*GetOrganisationResponse, error)
	UpdateLocation(context.Context, *UpdateLocationRequest) (*UpdateLocationResponse, error)
	Locate(context.Context, *LocateRequest) (*LocateResponse, error)
	GetQuota(context.Context, *QuotaRequest) (*QuotaResponse, error)
	AddExtractor(context.Context, *AddExtractorRequest) (*AddExtractorResponse, error)
	GetCache(context.Context, *GetCacheRequest) (*GetCacheResponse, error)
}

// UnimplementedOrganiserServiceServer should be embedded to have forward compatible implementations.
type UnimplementedOrganiserServiceServer struct {
}

func (UnimplementedOrganiserServiceServer) AddLocation(context.Context, *AddLocationRequest) (*AddLocationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddLocation not implemented")
}
func (UnimplementedOrganiserServiceServer) GetOrganisation(context.Context, *GetOrganisationRequest) (*GetOrganisationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetOrganisation not implemented")
}
func (UnimplementedOrganiserServiceServer) UpdateLocation(context.Context, *UpdateLocationRequest) (*UpdateLocationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateLocation not implemented")
}
func (UnimplementedOrganiserServiceServer) Locate(context.Context, *LocateRequest) (*LocateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Locate not implemented")
}
func (UnimplementedOrganiserServiceServer) GetQuota(context.Context, *QuotaRequest) (*QuotaResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetQuota not implemented")
}
func (UnimplementedOrganiserServiceServer) AddExtractor(context.Context, *AddExtractorRequest) (*AddExtractorResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddExtractor not implemented")
}
func (UnimplementedOrganiserServiceServer) GetCache(context.Context, *GetCacheRequest) (*GetCacheResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCache not implemented")
}

// UnsafeOrganiserServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to OrganiserServiceServer will
// result in compilation errors.
type UnsafeOrganiserServiceServer interface {
	mustEmbedUnimplementedOrganiserServiceServer()
}

func RegisterOrganiserServiceServer(s grpc.ServiceRegistrar, srv OrganiserServiceServer) {
	s.RegisterService(&_OrganiserService_serviceDesc, srv)
}

func _OrganiserService_AddLocation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddLocationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganiserServiceServer).AddLocation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/recordsorganiser.OrganiserService/AddLocation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganiserServiceServer).AddLocation(ctx, req.(*AddLocationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganiserService_GetOrganisation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetOrganisationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganiserServiceServer).GetOrganisation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/recordsorganiser.OrganiserService/GetOrganisation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganiserServiceServer).GetOrganisation(ctx, req.(*GetOrganisationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganiserService_UpdateLocation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateLocationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganiserServiceServer).UpdateLocation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/recordsorganiser.OrganiserService/UpdateLocation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganiserServiceServer).UpdateLocation(ctx, req.(*UpdateLocationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganiserService_Locate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LocateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganiserServiceServer).Locate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/recordsorganiser.OrganiserService/Locate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganiserServiceServer).Locate(ctx, req.(*LocateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganiserService_GetQuota_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuotaRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganiserServiceServer).GetQuota(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/recordsorganiser.OrganiserService/GetQuota",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganiserServiceServer).GetQuota(ctx, req.(*QuotaRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganiserService_AddExtractor_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddExtractorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganiserServiceServer).AddExtractor(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/recordsorganiser.OrganiserService/AddExtractor",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganiserServiceServer).AddExtractor(ctx, req.(*AddExtractorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrganiserService_GetCache_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetCacheRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrganiserServiceServer).GetCache(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/recordsorganiser.OrganiserService/GetCache",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrganiserServiceServer).GetCache(ctx, req.(*GetCacheRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _OrganiserService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "recordsorganiser.OrganiserService",
	HandlerType: (*OrganiserServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddLocation",
			Handler:    _OrganiserService_AddLocation_Handler,
		},
		{
			MethodName: "GetOrganisation",
			Handler:    _OrganiserService_GetOrganisation_Handler,
		},
		{
			MethodName: "UpdateLocation",
			Handler:    _OrganiserService_UpdateLocation_Handler,
		},
		{
			MethodName: "Locate",
			Handler:    _OrganiserService_Locate_Handler,
		},
		{
			MethodName: "GetQuota",
			Handler:    _OrganiserService_GetQuota_Handler,
		},
		{
			MethodName: "AddExtractor",
			Handler:    _OrganiserService_AddExtractor_Handler,
		},
		{
			MethodName: "GetCache",
			Handler:    _OrganiserService_GetCache_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "organise.proto",
}
