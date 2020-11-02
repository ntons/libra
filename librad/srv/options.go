package srv

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

type serveOptions struct {
	grpcOnly                   bool
	grpcServerOptions          []grpc.ServerOption
	grpcGatewayServeMuxOptions []runtime.ServeMuxOption
	unixDomainSock             string
}

type ServeOption interface {
	apply(o *serveOptions)
}

type funcServeOption struct {
	fn func(o *serveOptions)
}

func (fo funcServeOption) apply(o *serveOptions) {
	fo.fn(o)
}

// force gateway and http disabled, only high performance grpc enabled
func WithGrpcOnly(ok bool) ServeOption {
	return funcServeOption{func(o *serveOptions) {
		o.grpcOnly = ok
	}}
}

// grpc server options
func WithGrpcServerOption(opts ...grpc.ServerOption) ServeOption {
	return funcServeOption{func(o *serveOptions) {
		o.grpcServerOptions = opts
	}}
}

// grpc-gateway serve options
func WithGrpcGatewayServeMuxOption(opts ...runtime.ServeMuxOption) ServeOption {
	return funcServeOption{func(o *serveOptions) {
		o.grpcGatewayServeMuxOptions = opts
	}}
}

// unix domain socket used to forward gateway request to
func WithUnixDomainSock(path string) ServeOption {
	return funcServeOption{func(o *serveOptions) {
		o.unixDomainSock = path
	}}
}
