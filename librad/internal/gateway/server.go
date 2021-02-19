package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/internal/comm"
)

func init() {
	comm.RegisterService("gateway", create)
}

type server struct {
	v1.UnimplementedGatewayServer
	//
	ctx  context.Context
	stop context.CancelFunc
	//
	mu sync.RWMutex
	m  map[string]*session
	//
	hub *hub
}

func create(b json.RawMessage) (s comm.Service, err error) {
	var cfg config
	if err = json.Unmarshal(b, &cfg); err != nil {
		return
	}
	ropts, err := redis.ParseURL(cfg.Broadcast.Redis)
	if err != nil {
		return nil, fmt.Errorf("bad redis address: %v", cfg.Broadcast.Redis)
	}
	gw := &server{m: make(map[string]*session)}
	gw.ctx, gw.stop = context.WithCancel(context.Background())
	gw.hub = newHub(gw.ctx, redis.NewClient(ropts))
	return gw, nil
}

// implement comm.Service
func (gw *server) RegisterGrpc(s *grpc.Server) (err error) {
	v1.RegisterGatewayServer(s, gw)
	return
}
func (gw *server) Serve() { gw.hub.Serve() }
func (gw *server) Stop()  { gw.stop() }

// implement gwapi.Connect
func (gw *server) Connect(
	req *v1.GatewayConnectRequest, stream v1.Gateway_ConnectServer) error {
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

func (gw *server) Send(
	ctx context.Context, req *v1.GatewaySendRequest) (
	*v1.GatewaySendResponse, error) {
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
	return &v1.GatewaySendResponse{}, nil
}

func (gw *server) Subscribe(
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

func (gw *server) Unsubscribe(
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

func (gw *server) Broadcast(
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
