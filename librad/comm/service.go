package comm

import (
	"context"
	"net/http"

	grpcgw "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

type GrpcStreamInterceptor interface {
	InterceptStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error
}
type GrpcUnaryInterceptor interface {
	InterceptUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error)
}

type GrpcService interface {
	RegisterGrpc(gsrv *grpc.Server) error
}
type GrpcGatewayService interface {
	RegisterGrpcGateway(gcc *grpc.ClientConn, gwmux *grpcgw.ServeMux) error
}
type HttpService interface {
	RegisterHttp(hmux *http.ServeMux) error
}
type Service interface {
	Serve()
	Close()
}
