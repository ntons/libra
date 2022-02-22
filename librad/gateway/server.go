package gateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/log-go"
	"github.com/ntons/redis"
	"github.com/ntons/redmq"
	"github.com/onemoreteam/httpframework/modularity"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	modularity.Register(std)
}

var std = newGatewayServer()

type gatewayServer struct {
	modularity.Skeleton
	v1pb.UnimplementedGatewayServer

	mq redmq.Client

	quit context.CancelFunc
}

func newGatewayServer() *gatewayServer {
	return &gatewayServer{}
}

func (srv *gatewayServer) Name() string { return "gateway" }

func (srv *gatewayServer) Initialize(jb json.RawMessage) (err error) {
	if jb == nil {
		return
	}
	if err = json.Unmarshal(jb, &cfg); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	db, err := redis.Dial(ctx, cfg.Redis, redis.WithPingTest())
	if err != nil {
		return
	}

	srv.mq = redmq.New(db)

	return
}

func (srv *gatewayServer) Serve() error {
	var ctx context.Context
	ctx, srv.quit = context.WithCancel(context.Background())
	srv.mq.Serve(ctx)
	srv.quit = nil
	return nil
}

func (srv *gatewayServer) Shutdown() {
	if srv.quit != nil {
		srv.quit()
	}
}

func (srv *gatewayServer) Watch(stream v1pb.Gateway_WatchServer) (err error) {
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		var req *v1pb.GatewayWatchRequest
		if req, err = stream.Recv(); err != nil {
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if err := srv.mq.Watch(
					stream.Context(),
					req.Topic,
					req.Cursor,
					func(msg *redmq.Msg) {
						stream.Send(&v1pb.GatewayMessage{
							Topic:        msg.Topic,
							Id:           msg.Id,
							ProducerName: msg.ProducerName,
							EventTime:    msg.EventTime,
							PublishTime:  msg.PublishTime,
							Properties:   msg.Properties,
							Value:        msg.GetProtoValue(),
						})
					},
				); err != nil {
					if stream.Context().Err() != nil {
						return
					}
					log.Warnf("gateway Watch error: %v", err)
					select {
					case <-stream.Context().Done():
						return
					case <-time.After(time.Second):
					}
				}
			}
		}()
	}
	return
}

func (srv *gatewayServer) Send(
	ctx context.Context, req *v1pb.GatewaySendRequest) (
	_ *v1pb.GatewaySendResponse, err error) {
	if req.Msg == nil || req.Msg.Topic == "" || req.Msg.Value == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid message")
	}
	if err = srv.mq.Send(
		ctx,
		&redmq.Msg{
			Topic:        req.Msg.Topic,
			ProducerName: req.Msg.ProducerName,
			EventTime:    req.Msg.EventTime,
			Properties:   req.Msg.Properties,
			Value:        &redmq.Msg_ProtoValue{ProtoValue: req.Msg.Value},
		},
		redmq.WithMaxLen(req.MaxLen),
		redmq.WithAutoCreate(req.AutoCreate),
	); err != nil {
		log.Warnf("gateway Send error: %v", err)
		return
	}
	return
}
