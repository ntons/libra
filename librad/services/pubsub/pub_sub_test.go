package pubsub

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	logcfg "github.com/ntons/log-go/config"
	"github.com/ntons/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	initTestOnce sync.Once
)

func tryInitTest(t *testing.T) {
	initTestOnce.Do(func() {
		logcfg.DefaultZapJsonConfig.Use()
		var err error
		if cli, err = redis.Dial(
			context.Background(),
			"redis://localhost:6379/1",
			redis.WithPingTest(),
		); err != nil {
			t.Fatalf("failed to dial db: %v", err)
		}
	})
}

func getTestSrv(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", "localhost:5000")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	v1pb.RegisterPubSubServiceServer(s, newPubSubServer())
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	return s
}

func getTestCli(t *testing.T) (*grpc.ClientConn, v1pb.PubSubServiceClient) {
	conn, err := grpc.Dial("localhost:5000", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	return conn, v1pb.NewPubSubServiceClient(conn)
}

func TestSubscribe(t *testing.T) {
	tryInitTest(t)

	srv := getTestSrv(t)
	defer srv.GracefulStop()

	conn, cli := getTestCli(t)
	defer conn.Close()

	var (
		err error
		ctx = context.Background()
	)

	if _, err := cli.Publish(
		metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
			"x-libra-trusted-auth-by": "secret",
			"x-libra-trusted-app-id":  "myapp",
		})),
		&v1pb.PubSub_PublishRequest{
			Publications: []*v1pb.PubSub_Publication{
				&v1pb.PubSub_Publication{
					Topic: "test",
					Msgs: []*v1pb.PubSub_Msg{
						&v1pb.PubSub_Msg{
							Value: &v1pb.PubSub_Msg_Str{
								Str: "test",
							},
						},
					},
				},
			},
		}); err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	sub, err := cli.Subscribe(
		metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
			"x-libra-trusted-auth-by": "secret",
			"x-libra-trusted-app-id":  "myapp",
		})),
		&v1pb.PubSub_SubscribeRequest{
			Subscriptions: []*v1pb.PubSub_Subscription{
				&v1pb.PubSub_Subscription{Topic: "test", AfterId: "0"},
			},
		})
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	for {
		resp, err := sub.Recv()
		if err != nil {
			t.Fatalf("failed to recv: %v", err)
		}
		fmt.Printf("%v\n", resp)
	}
}

func TestConsume(t *testing.T) {
	tryInitTest(t)

	srv := getTestSrv(t)
	defer srv.GracefulStop()

	conn, cli := getTestCli(t)
	defer conn.Close()

	var (
		err  error
		ctx  = context.Background()
		resp *v1pb.PubSub_ConsumeResponse
	)

	var wg sync.WaitGroup
	defer wg.Wait()

	// consumers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				var req = &v1pb.PubSub_ConsumeRequest{
					Consumptions: []*v1pb.PubSub_Consumption{
						&v1pb.PubSub_Consumption{Topic: "test", AckTimeoutMilli: 1000},
					},
				}
				if resp != nil {
					for _, msg := range resp.Msgs {
						req.Acks = append(req.Acks, &v1pb.PubSub_Ack{
							Topic:  msg.Topic,
							MsgIds: []string{msg.Id},
						})
					}
				}
				fmt.Printf("[%d]consuming: %v\n", i, req)
				if resp, err = cli.Consume(
					metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
						"x-libra-trusted-auth-by": "secret",
						"x-libra-trusted-app-id":  "myapp",
					})),
					req,
				); err != nil {
					t.Fatalf("failed to consume: %v", err)
				}
				fmt.Printf("[%d]: %v\n", i, resp)
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if _, err := cli.Publish(
				metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
					"x-libra-trusted-auth-by": "secret",
					"x-libra-trusted-app-id":  "myapp",
				})),
				&v1pb.PubSub_PublishRequest{
					Publications: []*v1pb.PubSub_Publication{
						&v1pb.PubSub_Publication{
							Topic: "test",
							Msgs: []*v1pb.PubSub_Msg{
								&v1pb.PubSub_Msg{
									Value: &v1pb.PubSub_Msg_Str{
										Str: "test",
									},
								},
							},
						},
					},
				}); err != nil {
				t.Fatalf("failed to send: %v", err)
			}
			time.Sleep(3 * time.Second)
		}
	}()
}
