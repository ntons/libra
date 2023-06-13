package pubsub

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestPubSubServer(t *testing.T) {
	var (
		err error
		ctx = context.Background()
	)

	if cli, err = redis.Dial(ctx, "redis://localhost:6379/1", redis.WithPingTest()); err != nil {
		t.Fatalf("failed to dial db: %v", err)
	}

	lis, err := net.Listen("tcp", "localhost:5000")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	s := grpc.NewServer()
	v1pb.RegisterPubSubServiceServer(s, newPubSubServer())
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()
	defer s.GracefulStop()

	conn, err := grpc.Dial("localhost:5000", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	cli := v1pb.NewPubSubServiceClient(conn)

	if _, err := cli.Publish(
		metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
			"x-libra-trusted-auth-by": "secret",
			"x-libra-trusted-app-id":  "myapp",
		})),
		&v1pb.PubSub_PublishRequest{
			Msgs: []*v1pb.PubSub_Message{
				&v1pb.PubSub_Message{
					Topic: "test",
					Value: &v1pb.PubSub_Message_Str{
						Str: "test",
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
			TopicStart: map[string]*v1pb.PubSub_SubscribeRequest_Start{
				"test": &v1pb.PubSub_SubscribeRequest_Start{
					At: &v1pb.PubSub_SubscribeRequest_Start_AfterId{
						AfterId: "0",
					},
				},
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
