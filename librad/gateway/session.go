package gateway

import (
	"context"
	"sync"

	"github.com/ntons/libra-go/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

// 通过Access接入用户会话
type session struct {
	// attached role id
	Id string
	//
	ctx    context.Context
	cancel context.CancelFunc
	// push stream
	stream v1.Gateway_ConnectServer
	// push chan
	c chan *anypb.Any
	// be kicked by other session
	kicked bool
	//
	hub *hub
	// subscribed channels
	mu  sync.Mutex
	chs map[string]struct{}
}

func newSession(
	ctx context.Context, id string, stream v1.Gateway_ConnectServer) *session {
	s := &session{
		Id:     id,
		stream: stream,
		c:      make(chan *anypb.Any, 16),
		chs:    make(map[string]struct{}),
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	return s
}

func (s *session) Kick() { s.kicked = true; s.cancel() }

func (s *session) Serve() error {
	defer s.cancel()
	for {
		select {
		case msg := <-s.c:
			// not safe to call send from goroutines
			if err := s.stream.Send(msg); err != nil {
				return err
			}
		case <-s.ctx.Done():
			if s.kicked {
				return status.Errorf(codes.AlreadyExists, "kicked")
			} else {
				return status.Errorf(codes.Unavailable, "server closed")
			}
		case <-s.stream.Context().Done():
			return nil
		}
	}
}

func (s *session) Send(msg *anypb.Any) (err error) {
	select {
	case s.c <- msg:
	default:
		return status.Error(codes.Unavailable, "queue full")
	}
	return
}

func (s *session) Subscribe(ctx context.Context, keys ...string) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(keys) == 0 {
		return
	}
	select {
	case <-s.ctx.Done():
	}

	if err = s.hub.Subscribe(ctx, s, keys...); err != nil {
		return
	}
	for _, key := range keys {
		s.chs[key] = struct{}{}
	}
	return
}

func (s *session) Unsubscribe(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	s.mu.Unlock()
	defer s.mu.Unlock()
	s.hub.Unsubscribe(ctx, s, keys...)
	for _, key := range keys {
		delete(s.chs, key)
	}
}
