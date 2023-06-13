package pubsub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redis"
	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func init() { modularity.Register(newPubSubServer()) }

type pubSubServer struct {
	modularity.Skeleton
	v1pb.UnimplementedPubSubServiceServer
}

func newPubSubServer() *pubSubServer { return &pubSubServer{} }

func (*pubSubServer) Name() string { return "pubsub" }

func (srv *pubSubServer) Initalize(jb json.RawMessage) (err error) {
	if jb == nil {
		return
	}

	var cfg = struct {
		Redis string `json:"redis"`
	}{}
	if err = json.Unmarshal(jb, &cfg); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if cli, err = redis.Dial(ctx, cfg.Redis, redis.WithPingTest()); err != nil {
		return
	}

	server.RegisterService(&v1pb.PubSubService_ServiceDesc, srv)

	return
}

func getAppId(ctx context.Context) (appId string, err error) {
	if trusted := libra.RequireAuthBySecret(ctx); trusted == nil {
		return "", errUnauthenticated
	} else {
		return trusted.AppId, nil
	}
}

func toStream(appId, topic string) string {
	return fmt.Sprintf("%s:%s", appId, topic)
}

func toTopic(stream string) string {
	i := strings.LastIndexByte(stream, ':')
	return stream[i+1:]
}

func (*pubSubServer) Send(ctx context.Context, req *v1pb.PubSub_SendRequest) (_ *v1pb.PubSub_SendResponse, err error) {
	appId, err := getAppId(ctx)
	if err != nil {
		return
	}
	if req.Msg == nil {
		err = newInvalidArgumentError("bad msg")
		return
	}
	var b []byte
	if b, err = proto.Marshal(req.Msg); err != nil {
		err = newInvalidArgumentError("bad msg")
		return
	}
	args := &redis.XAddArgs{
		Stream: toStream(appId, req.Msg.Topic),
		Approx: true,
		ID:     "*",
		Values: append([]any{}, "PubSubMsg", base64.StdEncoding.EncodeToString(b)),
	}
	if req.Opts != nil {
		args.NoMkStream = !req.Opts.CreateTopic
		args.MaxLen = req.Opts.MaxLen
		args.MinID = req.Opts.MinId
	}
	if err = cli.XAdd(ctx, args).Err(); err != nil {
		return
	}
	return &v1pb.PubSub_SendResponse{}, nil
}

func (*pubSubServer) Read(req *v1pb.PubSub_ReadRequest, stream v1pb.PubSubService_ReadServer) (err error) {
	appId, err := getAppId(stream.Context())
	if err != nil {
		return
	}
	// 当前会话的读取位置，重建会话时需要调用者提供起始位置
	cursors := make(map[string]string)
	for topic, start := range req.TopicStart {
		var id string
		switch v := start.At.(type) {
		case *v1pb.PubSub_ReadRequest_Start_AfterId:
			id = v.AfterId
		case *v1pb.PubSub_ReadRequest_Start_SinceTimestampMillis:
			id = fmt.Sprintf("%d-0", v.SinceTimestampMillis)
		default:
			id = "0-0"
		}
		cursors[toStream(appId, topic)] = id
	}
	for {
		args := &redis.XReadArgs{}
		for stream, _ := range cursors {
			args.Streams = append(args.Streams, stream)
		}
		for _, cursor := range cursors {
			args.Streams = append(args.Streams, cursor)
		}
		fmt.Printf("%v\n", args)

		var resp = &v1pb.PubSub_ReadResponse{}

		var a []redis.XStream
		if a, err = cli.XRead(stream.Context(), args).Result(); err != nil {
			return
		}
		fmt.Printf("%v\n", a)
		for _, e := range a {
			for _, m := range e.Messages {
				cursors[e.Stream] = m.ID
				v, ok := m.Values["PubSubMsg"]
				if !ok {
					continue
				}
				var b []byte
				if b, err = base64.StdEncoding.DecodeString(v.(string)); err != nil {
					continue
				}
				m := &v1pb.PubSub_Msg{}
				if err = proto.Unmarshal(b, m); err != nil {
					continue
				}
				m.Topic = toTopic(e.Stream)
				m.Id = m.Id
				resp.Msgs = append(resp.Msgs, m)
			}
		}
		if len(resp.Msgs) == 0 {
			continue
		}
		if err = stream.Send(resp); err != nil {
			return
		}
	}
	return
}

func (*pubSubServer) Consume(stream v1pb.PubSubService_ConsumeServer) (err error) {
	return status.Errorf(codes.Unimplemented, "method Consume not implemented")
}
