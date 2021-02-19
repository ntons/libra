package comm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

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
	Stop()
}

/// Service factory
var (
	svcFactoryMu sync.Mutex
	svcFactories = make(map[string]ServiceFactory)
)

type ServiceFactory func(cfg json.RawMessage) (Service, error)

func RegisterService(name string, factory ServiceFactory) {
	svcFactoryMu.Lock()
	defer svcFactoryMu.Unlock()
	svcFactories[name] = factory
}

func CreateService(name string, cfg json.RawMessage) (Service, error) {
	svcFactoryMu.Lock()
	defer svcFactoryMu.Unlock()
	if factory, ok := svcFactories[name]; !ok {
		return nil, fmt.Errorf("unregistered service: %s", name)
	} else {
		return factory(cfg)
	}
}
