package srv

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

var (
	ErrUnimplemented = errors.New("unimplemented")
)

// alias some type for avoiding import by user
type GrpcServer = grpc.Server
type GrpcClientConn = grpc.ClientConn
type GrpcGatewayServeMux = runtime.ServeMux
type HttpServeMux = http.ServeMux

//
type Service interface {
	// register grpc service
	RegisterGrpc(*GrpcServer) error
	// register grpc-gateway service
	// the grpc implementation should be registered by RegisterGrpc
	RegisterGrpcGateway(*GrpcClientConn, *GrpcGatewayServeMux) error
	// register raw http service
	RegisterHttp(*HttpServeMux) error
	// waiting for service internal goroutine join
	Serve()
	// Stop service, invoked before server shutdown
	Stop()
}

// unimplemented server which provides services
type UnimplementedServer struct {
}

func (x UnimplementedServer) RegisterGrpc(grpcSrv *GrpcServer) error {
	return ErrUnimplemented
}
func (x UnimplementedServer) RegisterGrpcGateway(
	grpcConn *GrpcClientConn, grpcGatewayMux *GrpcGatewayServeMux) error {
	return ErrUnimplemented
}
func (x UnimplementedServer) RegisterHttp(httpMux *HttpServeMux) error {
	return ErrUnimplemented
}
func (x UnimplementedServer) Serve() {}
func (x UnimplementedServer) Stop()  {}

// service type registration
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

func createService(name string, cfg json.RawMessage) (Service, error) {
	svcFactoryMu.Lock()
	defer svcFactoryMu.Unlock()
	if factory, ok := svcFactories[name]; !ok {
		return nil, fmt.Errorf("unregistered service: %s", name)
	} else {
		return factory(cfg)
	}
}
