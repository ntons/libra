package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	_ "github.com/ntons/grpc-compressor/lz4" // register lz4 compressor
	"github.com/ntons/log-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	_ "github.com/ntons/libra/librad/database"
	_ "github.com/ntons/libra/librad/gateway"
	"github.com/ntons/libra/librad/internal/comm"
	_ "github.com/ntons/libra/librad/ranking"
	_ "github.com/ntons/libra/librad/registry"
	_ "github.com/ntons/libra/librad/syncer"
)

const (
	// alian serving status for short
	xStatusUnknown    = grpc_health_v1.HealthCheckResponse_UNKNOWN
	xStatusServing    = grpc_health_v1.HealthCheckResponse_SERVING
	xStatusNotServing = grpc_health_v1.HealthCheckResponse_NOT_SERVING
)

var (
	quit      = make(chan struct{}, 1)
	healthsrv = health.NewServer()
)

// intercept unary calls
// logging request and providing service-wide intercepting
func interceptUnary(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {
	if info.Server == healthsrv {
		return handler(ctx, req)
	}
	md, _ := metadata.FromIncomingContext(ctx)
	log.Debugw("unary call", "method", info.FullMethod, "metadata", md)
	if inter, ok := info.Server.(comm.GrpcUnaryInterceptor); ok {
		return inter.InterceptUnary(ctx, req, info, handler)
	}
	resp, err = handler(ctx, req)
	if x := status.Code(err); x != codes.OK && x != codes.NotFound {
		log.Warnw("unary call error",
			"method", info.FullMethod,
			"error", err,
		)
	}
	return
}

// intercept stream calls
// logging request and providing service-wide intercepting
func interceptStream(
	srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) (err error) {
	if srv == healthsrv {
		return handler(srv, ss)
	}
	md, _ := metadata.FromIncomingContext(ss.Context())
	log.Debugw("stream call", "method", info.FullMethod, "metadata", md)
	if inter, ok := srv.(comm.GrpcStreamInterceptor); ok {
		return inter.InterceptStream(srv, ss, info, handler)
	}
	err = handler(srv, ss)
	if x := status.Code(err); x != codes.OK && x != codes.NotFound {
		log.Warnw("stream call error",
			"method", info.FullMethod,
			"error", err,
		)
	}
	return
}

// start serving
func serve() (err error) {
	var wg sync.WaitGroup
	defer wg.Wait() // make sure all go routine exit

	grpcsrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptUnary),
		grpc.ChainStreamInterceptor(interceptStream),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime: 30 * time.Second,
			},
		),
	)

	// register services
	for name, b := range comm.Config.Services {
		var svc comm.Service
		if svc, err = comm.CreateService(name, b); err != nil {
			return
		}
		if grpcsvc, ok := svc.(comm.GrpcService); ok {
			if err = grpcsvc.RegisterGrpc(grpcsrv); err != nil {
				return
			}
		}
		wg.Add(1)
		go func(svc comm.Service) { defer wg.Done(); svc.Serve() }(svc)
		defer svc.Stop()
		healthsrv.SetServingStatus(name, xStatusServing)
		log.Infow("service registered", "name", name)
	}
	for name := range grpcsrv.GetServiceInfo() {
		log.Infow("grpc service registered", "name", name)
	}

	helloworld.RegisterGreeterServer(grpcsrv, &greeterServer{})

	grpc_health_v1.RegisterHealthServer(grpcsrv, healthsrv)
	healthsrv.SetServingStatus("", xStatusServing)

	lis, err := net.Listen("tcp", comm.Config.Bind)
	if err != nil {
		return fmt.Errorf("failed to listen at %s: %w", comm.Config.Bind, err)
	}
	defer lis.Close()

	wg.Add(1)
	go func() { defer wg.Done(); grpcsrv.Serve(lis) }()
	defer grpcsrv.GracefulStop()

	defer healthsrv.Shutdown() // mark status first when terminating

	<-quit // waiting for terminating
	return
}

// shutdown server
func shutdown() {
	select {
	case quit <- struct{}{}:
	default:
	}
}
