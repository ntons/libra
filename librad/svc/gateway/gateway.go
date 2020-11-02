package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/srv"
)

func init() {
	srv.RegisterService("gateway", create)
}

type gateway struct {
	srv.UnimplementedServer
	v1.UnimplementedGatewayServer
	//
	ctx    context.Context
	cancel context.CancelFunc
	//
	mu sync.RWMutex
	m  map[string]*session
	//
	hub *hub
}

func create(b json.RawMessage) (s srv.Service, err error) {
	var cfg config
	if err = json.Unmarshal(b, &cfg); err != nil {
		return
	}
	ropts, err := redis.ParseURL(cfg.Broadcast.Redis)
	if err != nil {
		return nil, fmt.Errorf("bad redis address: %v", cfg.Broadcast.Redis)
	}
	gw := &gateway{m: make(map[string]*session)}
	gw.ctx, gw.cancel = context.WithCancel(context.Background())
	gw.hub = newHub(gw.ctx, redis.NewClient(ropts))
	return gw, nil
}

// implement comm.Service
func (gw *gateway) RegisterGrpc(grpcSrv *srv.GrpcServer) (err error) {
	v1.RegisterGatewayServer(grpcSrv, gw)
	return
}

func (gw *gateway) Serve() { gw.hub.Serve() }

func (gw *gateway) Stop() { gw.cancel() }

// implement gwapi.Access
func (gw *gateway) Access(
	req *v1.GatewayAccessRequest, stream v1.Gateway_AccessServer) error {
	roleId, err := xLibraRoleId(stream.Context())
	if err != nil {
		return err
	}
	// reply a empty message immediately
	if err := stream.Send(&anypb.Any{}); err != nil {
		return err
	}
	// run session
	s := newSession(gw.ctx, roleId, stream)
	gw.mu.Lock()
	if s := gw.m[roleId]; s != nil {
		s.Kick()
	}
	gw.m[roleId] = s
	gw.mu.Unlock()
	log.Infof("sign in: %v", roleId)
	defer func() {
		gw.mu.Lock()
		delete(gw.m, roleId)
		gw.mu.Unlock()
		log.Infof("sign out: %v", roleId)
	}()
	return s.Serve()
}

func (gw *gateway) Push(
	ctx context.Context, req *v1.GatewayPushRequest) (
	*v1.GatewayPushResponse, error) {
	roleId, err := xLibraRoleId(ctx)
	if err != nil {
		return nil, err
	}
	gw.mu.RLock()
	s := gw.m[roleId]
	gw.mu.RUnlock()
	if s == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	if err := s.Send(req.Data); err != nil {
		return nil, err
	}
	return &v1.GatewayPushResponse{}, nil
}

func (gw *gateway) Subscribe(
	ctx context.Context, req *v1.GatewaySubscribeRequest) (
	*v1.GatewaySubscribeResponse, error) {
	roleId, err := xLibraRoleId(ctx)
	if err != nil {
		return nil, err
	}
	gw.mu.RLock()
	s := gw.m[roleId]
	gw.mu.RUnlock()
	if s == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	if err := gw.hub.Subscribe(ctx, s, req.Keys...); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}
	return &v1.GatewaySubscribeResponse{}, nil
}

func (gw *gateway) Unsubscribe(
	ctx context.Context, req *v1.GatewayUnsubscribeRequest) (
	_ *v1.GatewayUnsubscribeResponse, err error) {
	roleId, err := xLibraRoleId(ctx)
	if err != nil {
		return nil, err
	}
	gw.mu.RLock()
	s := gw.m[roleId]
	gw.mu.RUnlock()
	if s == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	gw.hub.Unsubscribe(ctx, s, req.Keys...)
	return &v1.GatewayUnsubscribeResponse{}, nil
}

func (gw *gateway) Broadcast(
	ctx context.Context, req *v1.GatewayBroadcastRequest) (
	*v1.GatewayBroadcastResponse, error) {
	if err := gw.hub.Broadcast(ctx, req.Key, req.Data); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}
	return &v1.GatewayBroadcastResponse{}, nil
}

// get roleId from context
func xLibraRoleId(ctx context.Context) (roleId string, err error) {
	if md, ok := metadata.FromIncomingContext(ctx); !ok {
		err = status.Error(codes.Unauthenticated, "no role id")
	} else if v := md.Get("x-libra-role-id"); len(v) == 0 {
		err = status.Error(codes.Unauthenticated, "no role id")
	} else if roleId = v[0]; roleId == "" {
		err = status.Error(codes.Unauthenticated, "no role id")
	}
	return
}
