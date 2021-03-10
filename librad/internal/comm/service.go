package comm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"google.golang.org/grpc"
)

type GrpcStreamInterceptor interface {
	InterceptStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error
}
type GrpcUnaryInterceptor interface {
	InterceptUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error)
}

type Service interface {
	Serve()
	Stop()
}

type GrpcService interface {
	RegisterGrpc(*grpc.Server) error
}

var (
	serviceMutex    sync.Mutex
	serviceCreators = make(map[string]ServiceCreator)
)

type ServiceCreator func(json.RawMessage) (Service, error)

func RegisterService(name string, creator ServiceCreator) {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()
	serviceCreators[name] = creator
}

func CreateService(name string, cfg json.RawMessage) (Service, error) {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()
	if creator, ok := serviceCreators[name]; !ok {
		return nil, fmt.Errorf("unregistered service: %s", name)
	} else {
		return creator(cfg)
	}
}
