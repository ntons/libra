package pubsub

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/log-go"
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
	if strings.ContainsRune(appId, ':') {
		panic("invalid app id")
	}
	return fmt.Sprintf("%s:%s", appId, topic)
}

func toTopic(stream string) string {
	i := strings.IndexRune(stream, ':')
	if i < 0 {
		panic("invalid stream")
	}
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

func (*pubSubServer) Subscribe(
	req *v1pb.PubSub_SubscribeRequest,
	stream v1pb.PubSubService_SubscribeServer) (err error) {

	appId, err := getAppId(stream.Context())
	if err != nil {
		return
	}

	var (
		wg sync.WaitGroup // reading goroutine waitgroup
		mu sync.Mutex     // sending mutex
	)
	defer wg.Wait()

	// 多个topic可能存在不同散列，需要分开read
	for topic, start := range req.TopicStart {
		id := fmt.Sprintf("%d-0", start.SinceMilliTimestamp)
		if start.AfterId != "" {
			a := strings.SplitN(start.AfterId, "-", 2)
			if v, err := strconv.ParseInt(a[0], 10, 64); err != nil {
				return newInvalidArgumentError(
					"invalid start after id: %v", start.AfterId)
			} else if v >= start.SinceMilliTimestamp {
				id = start.AfterId
			}
		}
		wg.Add(1)
		go func(args *redis.XReadArgs) {
			defer wg.Done()
			for {
				r, err := cli.XRead(stream.Context(), args).Result()
				if err != nil {
					log.Warnf("failed to read pubsub message: %v", err)
					return
				}
				var resp = &v1pb.PubSub_SubscribeResponse{}
				for _, e := range r {
					for _, m := range e.Messages {
						args.Streams[1] = m.ID
						v, ok := m.Values["PubSubMsg"]
						if !ok {
							log.Warnf("failed to parse pubsub message: %v", err)
							continue
						}
						var b []byte
						if b, err = base64.StdEncoding.DecodeString(v.(string)); err != nil {
							log.Warnf("failed to decode pubsub message: %v", err)
							continue
						}
						msg := &v1pb.PubSub_Message{}
						if err = proto.Unmarshal(b, msg); err != nil {
							log.Warnf("failed to unmarshal pubsub message: %v", err)
							continue
						}
						msg.Topic = toTopic(e.Stream)
						msg.Id = m.ID
						resp.Msgs = append(resp.Msgs, msg)
					}
				}
				if len(resp.Msgs) == 0 {
					continue
				}
				mu.Lock()
				err = stream.Send(resp)
				mu.Unlock()
				if err != nil {
					log.Warnf("failed to send pubsub response: %v", err)
					return
				}
			}
		}(&redis.XReadArgs{
			Streams: []string{toStream(appId, topic), id},
			Count:   int64(req.BatchCount),
		})
	}

	return
}
