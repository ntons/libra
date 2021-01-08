package echo

import (
	"context"
	"encoding/json"
	"time"

	grpcgw "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ntons/libra/librad/comm"
)

func init() { comm.RegisterService("echo", create) }

type config struct{}

type echoServer struct {
	v1.UnimplementedEchoServer

	ctx    context.Context
	cancel context.CancelFunc
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	cfg := &config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	srv := &echoServer{}
	srv.ctx, srv.cancel = context.WithCancel(context.Background())
	return srv, nil
}

func (*echoServer) Serve() {}
func (*echoServer) Close() {}

func (srv *echoServer) RegisterGrpc(s *grpc.Server) (err error) {
	v1.RegisterEchoServer(s, srv)
	return
}
func (srv *echoServer) RegisterGrpcGateway(
	cc *grpc.ClientConn, sm *grpcgw.ServeMux) (err error) {
	if err = v1.RegisterEchoHandler(srv.ctx, sm, cc); err != nil {
		return
	}
	return
}

func (srv *echoServer) Echo(
	ctx context.Context, req *v1.EchoRequest) (
	resp *v1.EchoResponse, err error) {
	md, _ := metadata.FromIncomingContext(ctx)
	log.Infof("echo request with metadata[%v]: %s", md, req.Content)
	return &v1.EchoResponse{Content: req.Content}, nil
}

func (srv *echoServer) Repeat(
	req *v1.EchoRepeatRequest, s v1.Echo_RepeatServer) (err error) {
	md, _ := metadata.FromIncomingContext(s.Context())
	log.Infof("repeat request with metadata[%v]: %s", md, req.Content)
	for seq := int32(0); seq < req.Count; seq++ {
		select {
		case <-srv.ctx.Done():
			return status.Errorf(codes.Unavailable, "server closed")
		case <-s.Context().Done():
			return
		case <-time.After(time.Duration(req.Interval) * time.Millisecond):
		}
		resp := &v1.EchoRepeatResponse{Content: req.Content, Seq: seq}
		if err = s.Send(resp); err != nil {
			return
		}
	}
	return
}
