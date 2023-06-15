package indexing

import (
	"context"
	"encoding/json"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/redis"

	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
)

func init() { modularity.Register(&indexing{}) }

type indexing struct {
	modularity.Skeleton
}

func (indexing) Name() string { return "indexing" }

func (indexing) Initialize(jb json.RawMessage) (err error) {
	if jb == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var cfg config
	if err = json.Unmarshal(jb, &cfg); err != nil {
		return
	}

	if cli, err = redis.Dial(ctx, cfg.Redis, redis.WithPingTest()); err != nil {
		return
	}

	// 子串索引
	server.RegisterGrpcService(
		&v1pb.SubstrIndex1Service_ServiceDesc,
		newSubstrIndex1Server())

	return
}
