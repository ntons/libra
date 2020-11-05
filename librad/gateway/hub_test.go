package gateway

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	md "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"
)

type mockStream struct {
}

func (s mockStream) Send(data *anypb.Any) error {
	fmt.Println("send: ", data)
	return nil
}
func (s mockStream) SetHeader(md.MD) error       { return nil }
func (s mockStream) SendHeader(md.MD) error      { return nil }
func (s mockStream) SetTrailer(md.MD)            {}
func (s mockStream) Context() context.Context    { return context.Background() }
func (s mockStream) SendMsg(m interface{}) error { return nil }
func (s mockStream) RecvMsg(m interface{}) error { return nil }

func TestHub(t *testing.T) {
	cli := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newHub(ctx, cli)

	wg.Add(1)
	go func() { defer wg.Done(); h.Serve() }()

	s1 := newSession(ctx, "s1", &mockStream{})
	wg.Add(1)
	go func() { defer wg.Done(); s1.Serve() }()

	s2 := newSession(ctx, "s2", &mockStream{})
	wg.Add(1)
	go func() { defer wg.Done(); s2.Serve() }()

	h.Subscribe(ctx, s1, "foo1")
	h.Subscribe(ctx, s2, "foo2")

	h.Broadcast(ctx, "foo1", &anypb.Any{TypeUrl: "foo1"})
	h.Broadcast(ctx, "foo2", &anypb.Any{TypeUrl: "foo2"})

	h.Unsubscribe(ctx, s1, "foo1")
	h.Unsubscribe(ctx, s1, "foo2")

	h.Broadcast(ctx, "foo1", &anypb.Any{TypeUrl: "foo1"})
	h.Broadcast(ctx, "foo2", &anypb.Any{TypeUrl: "foo2"})

	time.Sleep(time.Second)
}

/*
func TestSubscribe(t *testing.T) {
	cli := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := cli.Subscribe(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-sub.Channel():
				fmt.Println(msg)
			}
		}
	}()

	fmt.Println(sub.Subscribe(ctx, "test"))
	fmt.Println(sub.Subscribe(ctx, "test"))
	fmt.Println(cli.Publish(ctx, "test", "xxoo"))
	time.Sleep(time.Second)

	fmt.Println(sub.Unsubscribe(ctx, "test"))
	fmt.Println(sub.Unsubscribe(ctx, "test"))
	fmt.Println(cli.Publish(ctx, "test", "xxoo"))
	time.Sleep(time.Second)
}
*/
