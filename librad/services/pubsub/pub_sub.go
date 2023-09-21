package pubsub

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/libra/librad/common/util"
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

func fromXMessage(m redis.XMessage) (*v1pb.PubSub_Msg, error) {
	v, ok := m.Values["pubsub"]
	if !ok {
		return nil, fmt.Errorf("failed to get pubsub message value")
	}
	b, err := base64.StdEncoding.DecodeString(v.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to decode pubsub message: %e", err)
	}
	r := &v1pb.PubSub_Msg{}
	if err = proto.Unmarshal(b, r); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pubsub message: %v", err)
	}
	r.Id = m.ID
	return r, nil
}

func (srv *pubSubServer) Publish(
	ctx context.Context, req *v1pb.PubSub_PublishRequest) (
	_ *v1pb.PubSub_PublishResponse, err error) {

	var appId string
	if trusted := libra.RequireAuthBySecret(ctx); trusted == nil {
		return nil, errUnauthenticated
	} else {
		appId = trusted.AppId
	}

	var delayTs int64 = 0
	if req.Delay > 0 {
		delayTs = time.Now().Unix() + req.Delay
	}
	for _, pub := range req.Publications {
		if pub.AppId == "" {
			pub.AppId = appId
		}
		if pub.Topic == "" {
			return nil, newInvalidArgumentError("bad topic")
		}
		if delayTs > 0 {
			if err = srv.delayPublish(ctx, delayTs, pub); err != nil {
				log.Warnf("failed to delay publish: %v", err)
				return
			}
		} else {
			if err = srv.publish(ctx, pub); err != nil {
				log.Warnf("failed to publish: %v", err)
				return
			}
		}
	}

	return &v1pb.PubSub_PublishResponse{}, nil
}

func (*pubSubServer) publish(ctx context.Context, pub *v1pb.PubSub_Publication) (err error) {
	args := redis.XAddArgs{
		Stream:     toStream(pub.AppId, pub.Topic),
		NoMkStream: !pub.AutoCreateTopic,
		MaxLen:     int64(pub.MaxLen),
		Approx:     true,
		ID:         "*",
	}
	if pub.MaxTtl > 0 {
		ttl := time.Duration(pub.MaxTtl) * time.Millisecond
		args.MinID = fmt.Sprintf("%d", time.Now().Add(-ttl).UnixMilli())
	}
	for _, msg := range pub.Msgs {
		msg.Topic, msg.Id, msg.GroupId = "", "", 0 // clear fields
		var b []byte
		if b, err = proto.Marshal(msg); err != nil {
			return newInvalidArgumentError("bad msg")
		}
		args.Values = append([]any{}, "pubsub", base64.StdEncoding.EncodeToString(b))
		if err = cli.XAdd(ctx, &args).Err(); err != nil {
			if err == redis.Nil { // stream not found
				return newNotFoundError("stream not found")
			} else {
				log.Warnf("failed to add msg: %v", err)
				return newUnavailableError("db error")
			}
		}
	}
	return
}

func (*pubSubServer) Subscribe(
	req *v1pb.PubSub_SubscribeRequest,
	sess v1pb.PubSubService_SubscribeServer) error {

	var appId string
	if trusted := libra.RequireAuthBySecret(sess.Context()); trusted == nil {
		return errUnauthenticated
	} else {
		appId = trusted.AppId
	}

	var (
		wg sync.WaitGroup // reading goroutine waitgroup
		mu sync.Mutex     // sending mutex
	)
	defer wg.Wait()

	// 多个topic可能存在不同散列，需要分开read
	for _, sub := range req.Subscriptions {
		id := fmt.Sprintf("%d-0", sub.SinceMilliTimestamp)
		if sub.AfterId != "" {
			a := strings.SplitN(sub.AfterId, "-", 2)
			if v, err := strconv.ParseInt(a[0], 10, 64); err != nil {
				return newInvalidArgumentError(
					"invalid subscription after id: %v", sub.AfterId)
			} else if v >= sub.SinceMilliTimestamp {
				id = sub.AfterId
			}
		}
		wg.Add(1)
		go func(args *redis.XReadArgs) {
			defer wg.Done()
			for {
				r, err := cli.XRead(sess.Context(), args).Result()
				if err != nil {
					log.Warnf("failed to read pubsub message: %v", err)
					return
				}
				var resp = &v1pb.PubSub_SubscribeResponse{}
				for _, e := range r {
					topic := toTopic(e.Stream)
					for _, m := range e.Messages {
						args.Streams[1] = m.ID
						msg, err := fromXMessage(m)
						if err != nil {
							log.Warnf("%v", err)
							continue
						}
						msg.Topic = topic
						resp.Msgs = append(resp.Msgs, msg)
					}
				}
				if len(resp.Msgs) == 0 {
					continue
				}
				mu.Lock()
				err = sess.Send(resp)
				mu.Unlock()
				if err != nil {
					log.Warnf("failed to send pubsub response: %v", err)
					return
				}
			}
		}(&redis.XReadArgs{
			Streams: []string{toStream(appId, sub.Topic), id},
			Count:   int64(sub.BatchSize),
		})
	}

	return nil
}

func (srv *pubSubServer) Consume(
	ctx context.Context, req *v1pb.PubSub_ConsumeRequest) (
	_ *v1pb.PubSub_ConsumeResponse, err error) {

	var appId string
	if trusted := libra.RequireAuthBySecret(ctx); trusted == nil {
		return nil, errUnauthenticated
	} else {
		appId = trusted.AppId
	}

	if err = srv.ack(ctx, appId, req.Acks); err != nil {
		return
	}

	var (
		wg   sync.WaitGroup // reading goroutine waitgroup
		mu   sync.Mutex     // sending mutex
		resp = &v1pb.PubSub_ConsumeResponse{}
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, con := range req.Consumptions {
		wg.Add(1)
		go func(con *v1pb.PubSub_Consumption) {
			defer wg.Done()
			defer cancel()

			msgs, err := srv.readGroup(ctx, appId, con)
			if err != nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()
			resp.Msgs = append(resp.Msgs, msgs...)
		}(con)
	}

	wg.Wait() // waiting all reading goroutine join

	return resp, nil
}

func (*pubSubServer) ack(
	ctx context.Context, appId string, acks []*v1pb.PubSub_Ack) error {
	for _, ack := range acks {
		var (
			stream = toStream(appId, ack.Topic)
			group  = fmt.Sprintf("%d", ack.GroupId)
		)
		if err := cli.XAck(ctx, stream, group, ack.MsgIds...).Err(); err != nil {
			log.Warnf("failed to ack: %v", err)
			return newUnavailableError("db error")
		}
	}
	return nil
}

func (*pubSubServer) createGroup(
	ctx context.Context, stream, group string) (err error) {
	if err = cli.XGroupCreateMkStream(
		ctx, stream, group, "0-0").Err(); isBusyGroupError(err) {
		err = nil // 有可能同时被其他消费者创建
	}
	return
}

func (srv *pubSubServer) readGroup(
	ctx context.Context, appId string, con *v1pb.PubSub_Consumption) (
	msgs []*v1pb.PubSub_Msg, err error) {
	var (
		stream  = toStream(appId, con.Topic)
		group   = fmt.Sprintf("%d", con.GroupId)
		timeout = time.Duration(con.AckTimeout) * time.Millisecond

		xAutoClaimArgs = &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			MinIdle:  timeout,
			Start:    "0-0",
			Count:    1,
			Consumer: group,
		}

		xReadGroupArgs = &redis.XReadGroupArgs{
			Group:    group,
			Consumer: group,
			Streams:  []string{stream, ">"},
			Count:    1,
			Block:    timeout, // 轮询autoclaim
		}
	)

	for len(msgs) == 0 {
		var r []redis.XMessage

	xAutoClaim:
		if r, _, err = cli.XAutoClaim(ctx, xAutoClaimArgs).Result(); err != nil {
			if isNoGroupError(err) {
				if err = srv.createGroup(ctx, stream, group); err == nil {
					goto xAutoClaim
				}
			}
			if !isCanceledError(err) {
				log.Warnf("failed to claim: %v", err)
			}
			return
		}

	xReadGroup:
		if len(r) == 0 {
			var r1 []redis.XStream
			r1, err = cli.XReadGroup(ctx, xReadGroupArgs).Result()
			if err != nil && err != redis.Nil {
				if isNoGroupError(err) {
					if err = srv.createGroup(ctx, stream, group); err == nil {
						goto xReadGroup
					}
				}
				if !isCanceledError(err) {
					log.Warnf("failed to read group: %v", err)
				}
				return
			}
			for _, e := range r1 {
				if e.Stream != stream {
					log.Errorf("read group got mismatched stream: %v, %v", e.Stream, stream)
					continue
				}
				r = append(r, e.Messages...)
			}
		}

		for _, m := range r {
			if msg, err := fromXMessage(m); err != nil {
				log.Warnf("%v", err)
			} else {
				msg.Topic = con.Topic
				msg.GroupId = con.GroupId
				msgs = append(msgs, msg)
			}
		}
	}
	return
}

// 延迟发送消息
func (srv *pubSubServer) delayPublish(ctx context.Context, ts int64, pub *v1pb.PubSub_Publication) (err error) {
	b, err := proto.Marshal(pub)
	if err != nil {
		log.Warnf("failed to marshal delay pub: %v", err)
		return
	}
	if err = luaDelay.Run(ctx, cli, []string{"$PUBSUB_DELAY$"}, "add", ts, util.BytesToString(b)).Err(); err != nil {
		log.Warnf("failed to add delay pub: %v", err)
		return
	}
	return
}

func (srv *pubSubServer) tryPopDelay(ctx context.Context) (list []*v1pb.PubSub_Publication, err error) {
	r, err := luaDelay.Run(ctx, cli, []string{"$PUBSUB_DELAY$"}, "try_pop", time.Now().Unix()).Result()
	if err != nil {
		log.Warnf("failed to pop delay pub: %v", err)
		return
	}

	for _, v := range r.([]interface{}) {
		pub := &v1pb.PubSub_Publication{}
		if err = proto.Unmarshal(util.StringToBytes(v.(string)), pub); err != nil {
			log.Warnf("failed to unmarshal delay pub: %v", err)
			continue
		}
		list = append(list, pub)
	}
	return
}

func (srv *pubSubServer) tryDelayPublish(ctx context.Context) {
	list, err := srv.tryPopDelay(ctx)
	if err != nil {
		log.Warnf("failed to pop delay: %v", err)
		return
	}
	for _, pub := range list {
		if err := srv.publish(ctx, pub); err != nil {
			log.Warnf("failed to publish: %v", err)
		}
	}
}
