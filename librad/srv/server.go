package srv

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

func randString(n int) string {
	const (
		S = "abcdefghijklmnopqrstuvwxyz"
		N = len(S)
	)
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(S[rand.Intn(N)])
	}
	return sb.String()
}

func isContentType(r *http.Request, value string) bool {
	return strings.Contains(r.Header.Get("Content-Type"), value)
}

type Server struct {
	// health service
	health *health.Server
	// life-time control
	ctx    context.Context
	cancel context.CancelFunc
	// services
	svcs []Service
}

func New() (srv *Server) {
	srv = &Server{health: health.NewServer()}
	// an empty service name stands for entire server status
	srv.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	srv.ctx, srv.cancel = context.WithCancel(context.Background())
	return
}

// register must be invoked before server
func (srv *Server) RegisterService(name string, svc Service) {
	srv.svcs = append(srv.svcs, svc)
	srv.health.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
}

func (srv *Server) ListenAndServe(addr string, opts ...ServeOption) (err error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	defer lis.Close()
	return srv.Serve(lis, opts...)
}

// For serving mutilple protocol service on the same port, x/net/http2
// is used instead of grpc http/2 implementation.
// The x/net/htt2 impact the performance by about 50% lower on some cases.
// If only grpc services registered, prefer grpc http/2 to x/net/http2
func (srv *Server) Serve(lis net.Listener, opts ...ServeOption) (err error) {
	var o = &serveOptions{
		unixDomainSock: fmt.Sprintf("/tmp/%s.sock", randString(10)),
	}
	for _, opt := range opts {
		opt.apply(o)
	}

	var wg sync.WaitGroup
	defer wg.Wait() // make sure all go routine exit

	// listen on unix domain socket for gateway
	uds, err := net.Listen("unix", o.unixDomainSock)
	if err != nil {
		return err
	}
	defer uds.Close()

	// connect to unix domain socket
	grpcConn, err := grpc.Dial("unix://"+o.unixDomainSock, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer grpcConn.Close()

	// create grpc server
	grpcSrv := grpc.NewServer(
		append([]grpc.ServerOption{
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime: 30 * time.Second,
			}),
		}, o.grpcServerOptions...)...)
	defer grpcSrv.GracefulStop()

	// dispatch request between grpc/gateway/http
	grpcGatewayMux := runtime.NewServeMux(o.grpcGatewayServeMuxOptions...)
	httpMux := http.NewServeMux()
	httpSrv := http.Server{
		Handler: h2c.NewHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.ProtoMajor == 2 {
					if isContentType(r, "application/grpc") {
						grpcSrv.ServeHTTP(w, r)
					} else {
						w.WriteHeader(http.StatusBadRequest)
					}
				} else if r.ProtoMajor == 1 {
					if isContentType(r, "application/grpc-gateway") {
						grpcGatewayMux.ServeHTTP(w, r)
					} else {
						httpMux.ServeHTTP(w, r)
					}
				}
			}),
			&http2.Server{},
		),
	}
	defer httpSrv.Shutdown(context.Background())

	// services must be registered here because of defer sequence
	withGrpcGateway, withHttp := false, false
	for _, svc := range srv.svcs {
		if err = svc.RegisterGrpc(grpcSrv); err != nil {
			if err != ErrUnimplemented {
				return
			}
		}
		if err = svc.RegisterGrpcGateway(grpcConn, grpcGatewayMux); err != nil {
			if err != ErrUnimplemented {
				return
			}
		} else {
			withGrpcGateway = true
		}
		if err = svc.RegisterHttp(httpMux); err != nil {
			if err != ErrUnimplemented {
				return
			}
		} else {
			withHttp = true
		}
		err = nil
		wg.Add(1)
		go func(svc Service) { defer wg.Done(); svc.Serve() }(svc)
		defer svc.Stop()
	}
	// health service
	healthpb.RegisterHealthServer(grpcSrv, srv.health)
	defer srv.health.Shutdown()

	// serving, grpcGateway cannot be served without grpc
	if o.grpcOnly {
		wg.Add(1)
		go func() { defer wg.Done(); grpcSrv.Serve(lis) }()
	} else {
		if withGrpcGateway {
			wg.Add(1)
			go func() { defer wg.Done(); grpcSrv.Serve(uds) }()
		}
		if withGrpcGateway || withHttp {
			wg.Add(1)
			go func() { defer wg.Done(); httpSrv.Serve(lis) }()
		} else {
			wg.Add(1)
			go func() { defer wg.Done(); grpcSrv.Serve(lis) }()
		}
	}

	<-srv.ctx.Done()
	return
}

func (srv *Server) Shutdown() {
	srv.cancel()
}

func (srv *Server) WaitForTerm() {
	sig := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGPIPE)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	select {
	case <-sig:
	case <-srv.ctx.Done():
	}
	return
}

func (srv *Server) SetServingStatus(
	name string, status healthpb.HealthCheckResponse_ServingStatus) {
	srv.health.SetServingStatus(name, status)
}
