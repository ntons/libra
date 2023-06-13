package pubsub

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redis"
	"google.golang.org/protobuf/proto"
)

type pubSubServer struct {
	v1pb.UnimplementedPubSubServiceServer
}

func newPubSubServer() *pubSubServer { return &pubSubServer{} }

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

func (*pubSubServer) Publish(
	ctx context.Context, req *v1pb.PubSub_PublishRequest) (
	_ *v1pb.PubSub_PublishResponse, err error) {

	appId, err := getAppId(ctx)
	if err != nil {
		return
	}

	args := &redis.XAddArgs{
		Approx: true,
		ID:     "*",
	}
	if req.Opts != nil {
		args.NoMkStream = !req.Opts.CreateTopic
		args.MaxLen = req.Opts.MaxLen
		args.MinID = req.Opts.MinId
	}

	for _, msg := range req.Msgs {
		var b []byte
		if b, err = proto.Marshal(msg); err != nil {
			err = newInvalidArgumentError("bad msg")
			return
		}
		args.Stream = toStream(appId, msg.Topic)
		args.Values = append([]any{}, "PubSubMsg", base64.StdEncoding.EncodeToString(b))
		if err = cli.XAdd(ctx, args).Err(); err != nil {
			return
		}
	}

	return &v1pb.PubSub_PublishResponse{}, nil
}

func (*pubSubServer) Subscribe(req *v1pb.PubSub_SubscribeRequest, stream v1pb.PubSubService_SubscribeServer) (err error) {
	appId, err := getAppId(stream.Context())
	if err != nil {
		return
	}

	// 当前会话的读取位置，重建会话时需要调用者提供起始位置
	cursors := make(map[string]string)
	for topic, start := range req.TopicStart {
		var id string
		switch v := start.At.(type) {
		case *v1pb.PubSub_SubscribeRequest_Start_AfterId:
			id = v.AfterId
		case *v1pb.PubSub_SubscribeRequest_Start_SinceTimestampMillis:
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

		var resp = &v1pb.PubSub_SubscribeResponse{}

		var r []redis.XStream
		if r, err = cli.XRead(stream.Context(), args).Result(); err != nil {
			return
		}
		for _, e := range r {
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
				m := &v1pb.PubSub_Message{}
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
