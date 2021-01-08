package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	grpcgw "github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/ntons/log-go"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/ntons/libra/librad/comm"
	_ "github.com/ntons/libra/librad/database"
	_ "github.com/ntons/libra/librad/echo"
	_ "github.com/ntons/libra/librad/gateway"
	_ "github.com/ntons/libra/librad/portal"
	_ "github.com/ntons/libra/librad/ranking"
)

const (
	xLibra  = "x-libra-"
	xCookie = "x-cookie-x-libra-"
)

var (
	quit           = make(chan struct{}, 1)
	grpcServer     = newGrpcServer()
	grpcGatewayMux = newGrpcGatewayServeMux()
	httpMux        = http.NewServeMux()
)

func newGrpcServer() *grpc.Server {
	return grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime: 30 * time.Second,
			},
		),
		grpc.ChainUnaryInterceptor(func(
			ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler) (resp interface{}, err error) {
			if inter, ok := info.Server.(comm.GrpcUnaryInterceptor); ok {
				return inter.InterceptUnary(ctx, req, info, handler)
			}
			return handler(ctx, req)
		}),
		grpc.ChainStreamInterceptor(func(
			srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
			handler grpc.StreamHandler) (err error) {
			if inter, ok := srv.(comm.GrpcStreamInterceptor); ok {
				return inter.InterceptStream(srv, ss, info, handler)
			}
			return handler(srv, ss)
		}),
	)
}

func newGrpcGatewayServeMux() *grpcgw.ServeMux {
	return grpcgw.NewServeMux(
		grpcgw.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			return key, strings.HasPrefix(strings.ToLower(key), xLibra)
		}),
		grpcgw.WithOutgoingHeaderMatcher(func(key string) (string, bool) {
			return key, strings.HasPrefix(strings.ToLower(key), xLibra)
		}),
		grpcgw.WithMetadata(func(
			ctx context.Context, req *http.Request) metadata.MD {
			md := make(metadata.MD)
			for _, cookie := range req.Cookies() {
				if strings.HasPrefix(strings.ToLower(cookie.Name), xLibra) {
					md.Set(cookie.Name, cookie.Value)
				}
			}
			return md
		}),
		grpcgw.WithForwardResponseOption(func(
			ctx context.Context, w http.ResponseWriter,
			_ proto.Message) (_ error) {
			md, ok := grpcgw.ServerMetadataFromContext(ctx)
			if !ok {
				return
			}
			for key, vals := range md.HeaderMD {
				if strings.HasPrefix(strings.ToLower(key), xCookie) {
					http.SetCookie(w, &http.Cookie{
						Name:  key[len(xCookie)-len(xLibra):],
						Value: vals[0],
					})
				}
			}
			return
		}),
	)
}

func newHttpServer() *http.Server {
	isContentType := func(r *http.Request, v string) bool {
		return strings.Contains(r.Header.Get("Content-Type"), v)
	}
	return &http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.ProtoMajor == 2 {
					if isContentType(r, "application/grpc") {
						grpcServer.ServeHTTP(w, r)
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
}

// For serving mutilple protocol service on the same port, x/net/http2
// is used instead of grpc http/2 implementation.
// The x/net/htt2 impact the performance by about 50% lower on some cases.
// If only grpc services registered, prefer grpc http/2 to x/net/http2
func serve(cfg *Config) (err error) {
	var (
		wg       sync.WaitGroup
		grpcOnly = true
		grpcConn *grpc.ClientConn
	)
	defer wg.Wait() // make sure all go routine exit

	// register services
	registerHealthServer()
	setStatus("", xStatusServing)
	for name, b := range cfg.Services {
		var svc comm.Service
		if svc, err = comm.CreateService(name, b); err != nil {
			return
		}
		if _svc, ok := svc.(comm.GrpcService); ok {
			if err = _svc.RegisterGrpc(grpcServer); err != nil {
				return
			}
		}
		if _svc, ok := svc.(comm.GrpcGatewayService); ok {
			if grpcConn == nil {
				if grpcConn, err = grpc.Dial(
					"unix://"+cfg.UnixDomainSock,
					grpc.WithInsecure(),
				); err != nil {
					return err
				}
				defer grpcConn.Close()
				log.Infof("connect to %s", cfg.UnixDomainSock)
			}
			if err = _svc.RegisterGrpcGateway(
				grpcConn, grpcGatewayMux); err != nil {
				return
			}
			grpcOnly = false
		}
		if _svc, ok := svc.(comm.HttpService); ok {
			if err = _svc.RegisterHttp(httpMux); err != nil {
				return
			}
			grpcOnly = false
		}
		wg.Add(1)
		go func(svc comm.Service) { defer wg.Done(); svc.Serve() }(svc)
		defer svc.Close()
		setStatus(name, xStatusServing)
		log.Infow("service registered", "name", name)
	}

	// primary listener
	lis, err := net.Listen("tcp", cfg.Bind)
	if err != nil {
		return fmt.Errorf("failed to listen at %s: %w", cfg.Bind, err)
	}
	defer lis.Close()
	log.Infof("listen at %s", cfg.Bind)

	// start serving
	if grpcOnly || cfg.GrpcOnly {
		wg.Add(1)
		go func() {
			defer wg.Done()
			grpcServer.Serve(lis)
			log.Infof("grpc server has shutdown")
		}()
		defer grpcServer.GracefulStop()
		log.Infof("grpc server serve at %s", lis.Addr().String())
	} else {
		if grpcConn != nil {
			ulis, err := net.Listen("unix", cfg.UnixDomainSock)
			if err != nil {
				return err
			}
			defer ulis.Close()
			log.Infof("listen at %s", cfg.UnixDomainSock)

			wg.Add(1)
			go func() {
				defer wg.Done()
				grpcServer.Serve(ulis)
				log.Infof("grpc server has shutdown")
			}()
			defer grpcServer.GracefulStop()
			log.Infof("grpc server serve at %s", ulis.Addr().String())
		}
		httpServer := newHttpServer()
		wg.Add(1)
		go func() {
			defer wg.Done()
			httpServer.Serve(lis)
			log.Infof("http server has shutdown")
		}()
		defer httpServer.Shutdown(context.Background())
		log.Infof("http server serve at %s", lis.Addr().String())
	}

	// mark status first when terminate
	defer shutdownHealthServer()

	// waiting for terminating
	<-quit
	return
}

// shutdown server
func shutdown() {
	select {
	case quit <- struct{}{}:
	default:
	}
}
