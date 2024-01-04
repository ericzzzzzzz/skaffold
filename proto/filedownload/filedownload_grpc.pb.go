// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.1
// source: proto/filedownload/filedownload.proto

package filedownload

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
	FileService_DownloadFile_FullMethodName = "/proto.filedownload.FileService/DownloadFile"
	FileService_Watch_FullMethodName        = "/proto.filedownload.FileService/Watch"
)

// FileServiceClient is the client API for FileService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FileServiceClient interface {
	DownloadFile(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (FileService_DownloadFileClient, error)
	Watch(ctx context.Context, in *FileWatchRequest, opts ...grpc.CallOption) (FileService_WatchClient, error)
}

type fileServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFileServiceClient(cc grpc.ClientConnInterface) FileServiceClient {
	return &fileServiceClient{cc}
}

func (c *fileServiceClient) DownloadFile(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (FileService_DownloadFileClient, error) {
	stream, err := c.cc.NewStream(ctx, &FileService_ServiceDesc.Streams[0], FileService_DownloadFile_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &fileServiceDownloadFileClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type FileService_DownloadFileClient interface {
	Recv() (*DownloadResponse, error)
	grpc.ClientStream
}

type fileServiceDownloadFileClient struct {
	grpc.ClientStream
}

func (x *fileServiceDownloadFileClient) Recv() (*DownloadResponse, error) {
	m := new(DownloadResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *fileServiceClient) Watch(ctx context.Context, in *FileWatchRequest, opts ...grpc.CallOption) (FileService_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &FileService_ServiceDesc.Streams[1], FileService_Watch_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &fileServiceWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type FileService_WatchClient interface {
	Recv() (*FileEvent, error)
	grpc.ClientStream
}

type fileServiceWatchClient struct {
	grpc.ClientStream
}

func (x *fileServiceWatchClient) Recv() (*FileEvent, error) {
	m := new(FileEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// FileServiceServer is the server API for FileService service.
// All implementations must embed UnimplementedFileServiceServer
// for forward compatibility
type FileServiceServer interface {
	DownloadFile(*DownloadRequest, FileService_DownloadFileServer) error
	Watch(*FileWatchRequest, FileService_WatchServer) error
	mustEmbedUnimplementedFileServiceServer()
}

// UnimplementedFileServiceServer must be embedded to have forward compatible implementations.
type UnimplementedFileServiceServer struct {
}

func (UnimplementedFileServiceServer) DownloadFile(*DownloadRequest, FileService_DownloadFileServer) error {
	return status.Errorf(codes.Unimplemented, "method DownloadFile not implemented")
}
func (UnimplementedFileServiceServer) Watch(*FileWatchRequest, FileService_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
func (UnimplementedFileServiceServer) mustEmbedUnimplementedFileServiceServer() {}

// UnsafeFileServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FileServiceServer will
// result in compilation errors.
type UnsafeFileServiceServer interface {
	mustEmbedUnimplementedFileServiceServer()
}

func RegisterFileServiceServer(s grpc.ServiceRegistrar, srv FileServiceServer) {
	s.RegisterService(&FileService_ServiceDesc, srv)
}

func _FileService_DownloadFile_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(DownloadRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(FileServiceServer).DownloadFile(m, &fileServiceDownloadFileServer{stream})
}

type FileService_DownloadFileServer interface {
	Send(*DownloadResponse) error
	grpc.ServerStream
}

type fileServiceDownloadFileServer struct {
	grpc.ServerStream
}

func (x *fileServiceDownloadFileServer) Send(m *DownloadResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _FileService_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(FileWatchRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(FileServiceServer).Watch(m, &fileServiceWatchServer{stream})
}

type FileService_WatchServer interface {
	Send(*FileEvent) error
	grpc.ServerStream
}

type fileServiceWatchServer struct {
	grpc.ServerStream
}

func (x *fileServiceWatchServer) Send(m *FileEvent) error {
	return x.ServerStream.SendMsg(m)
}

// FileService_ServiceDesc is the grpc.ServiceDesc for FileService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FileService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.filedownload.FileService",
	HandlerType: (*FileServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "DownloadFile",
			Handler:       _FileService_DownloadFile_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "Watch",
			Handler:       _FileService_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/filedownload/filedownload.proto",
}
