package pubsub

import (
	"context"
	"encoding/json"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redis"
	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
)

func init() { modularity.Register(&module{}) }

type module struct {
	modularity.Skeleton
}

func (module) Name() string { return "pubsub" }

func (module) Initialize(jb json.RawMessage) (err error) {
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

	server.RegisterGrpcService(&v1pb.PubSubService_ServiceDesc, newPubSubServer())

	return
}
