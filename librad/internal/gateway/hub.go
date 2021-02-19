package gateway

import (
	"context"
	"errors"
	"sync"

	"github.com/go-redis/redis/v8"
	log "github.com/ntons/log-go"
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/internal/comm"
)

var errHubClosed = errors.New("hub closed")

// 进程内置订阅分发点，防止用户过多对redis产生过大的压力
type hub struct {
	ctx context.Context
	// redis
	cli *redis.Client
	sub *redis.PubSub
	// channel manager
	mu sync.RWMutex
	m  map[string]*channel
	wg sync.WaitGroup
}

func newHub(ctx context.Context, cli *redis.Client) *hub {
	return &hub{
		ctx: ctx,
		cli: cli,
		sub: cli.Subscribe(ctx),
		m:   make(map[string]*channel),
	}
}

func (h *hub) Serve() {
	defer h.wg.Wait()
	for {
		select {
		case <-h.ctx.Done():
			return
		case msg := <-h.sub.Channel():
			if msg == nil {
				break
			}
			h.mu.RLock()
			ch := h.m[msg.Channel]
			h.mu.RUnlock()
			if ch == nil {
				log.Warnf("unhandled broadcast channel %s", msg.Channel)
				break
			}
			ch.Broadcast(msg)
		}
	}
}

func (h *hub) Subscribe(
	ctx context.Context, s *session, keys ...string) (err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	select {
	case <-h.ctx.Done():
		return
	default:
	}

	var tmp []string
	for _, key := range keys {
		if _, ok := h.m[key]; !ok {
			tmp = append(tmp, key)
		}
	}
	if len(tmp) > 0 {
		if err = h.sub.Subscribe(ctx, tmp...); err != nil {
			log.Warnf("failed to subscribe: %v", err)
			return
		}
	}
	for _, key := range keys {
		ch := h.m[key]
		if ch == nil {
			ch = newChannel(h.ctx)
			h.m[key] = ch
			h.wg.Add(1)
			go func() { defer h.wg.Done(); ch.Serve() }()
		}
		ch.Add(s)
	}
	return
}

func (h *hub) Unsubscribe(ctx context.Context, s *session, keys ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	select {
	case <-h.ctx.Done():
		return
	default:
	}

	var tmp []string
	for _, key := range keys {
		ch := h.m[key]
		if ch == nil {
			continue
		}
		if n := ch.Del(s); n == 0 {
			ch.Close()
			delete(h.m, key)
			tmp = append(tmp, key)
		}
	}
	if len(tmp) > 0 {
		if err := h.sub.Unsubscribe(ctx, tmp...); err != nil {
			log.Warnf("failed to unsubscribed: %v", err)
		}
	}
}

func (h *hub) Broadcast(
	ctx context.Context, key string, data *anypb.Any) (err error) {
	b, err := pb.Marshal(data)
	if err != nil {
		return
	}
	if err = h.cli.Publish(ctx, key, comm.B2S(b)).Err(); err != nil {
		return
	}
	return
}

type channel struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	m      map[string]*session
	c      chan *redis.Message
}

func newChannel(ctx context.Context) *channel {
	ch := &channel{
		m: make(map[string]*session),
		c: make(chan *redis.Message, 16),
	}
	ch.ctx, ch.cancel = context.WithCancel(ctx)
	return ch
}
func (ch *channel) Close() { ch.cancel() }

func (ch *channel) Serve() {
	for {
		select {
		case <-ch.ctx.Done():
			return
		case msg := <-ch.c:
			data := &anypb.Any{}
			if err := pb.Unmarshal(comm.S2B(msg.Payload), data); err != nil {
				log.Warn("failed to broadcast: bad message")
				break
			}
			ch.mu.Lock()
			for _, s := range ch.m {
				if err := s.Send(data); err != nil {
					log.Warnf("failed to broadcast to %s: %s", s.Id, err)
				}
			}
			ch.mu.Unlock()
		}
	}
}

func (ch *channel) Broadcast(msg *redis.Message) {
	select {
	case ch.c <- msg:
	default:
		log.Warn("failed to broadcast: queue full")
	}
}
func (ch *channel) Add(s *session) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.m[s.Id] = s
}
func (ch *channel) Del(s *session) int {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	delete(ch.m, s.Id)
	return len(ch.m)
}
