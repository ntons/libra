package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/log-go"
	"github.com/ntons/tongo/redis"
	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ntons/libra/librad/common/util"
)

func init() { modularity.Register(&cacheServer{}) }

type cacheServer struct {
	modularity.Skeleton
	v1pb.UnimplementedCacheServer
	cli redis.Client
}

func (cacheServer) Name() string { return "cache" }

func (srv *cacheServer) Initialize(jb json.RawMessage) (err error) {
	if err = json.Unmarshal(jb, &cfg); err != nil {
		return
	}
	if cfg.Redis == "" {
		return fmt.Errorf("require redis configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if srv.cli, err = redis.Dial(ctx, cfg.Redis); err != nil {
		log.Warnf("failed to connect redis: %v", err)
		return fmt.Errorf("failed to connect redis")
	}

	v1pb.RegisterCacheServer(server.Default, srv)
	return
}

func (srv *cacheServer) Get(
	ctx context.Context, req *v1pb.CacheSetRequest) (
	_ *v1pb.CacheGetResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	value, err := srv.cli.Get(
		ctx,
		getCacheKey(trusted.AppId, req.Key),
	).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, status.Errorf(codes.NotFound, "not found")
		} else {
			log.Warnf("failed to get from redis: %v", err)
			return nil, status.Errorf(codes.Unavailable, "redis error")
		}
	}
	return &v1pb.CacheGetResponse{
		Value: util.StringToBytes(value),
	}, nil
}

func (srv *cacheServer) Set(
	ctx context.Context, req *v1pb.CacheSetRequest) (
	_ *v1pb.CacheSetResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if err = srv.cli.Set(
		ctx,
		getCacheKey(trusted.AppId, req.Key),
		req.Value,
		getTimeout(req.Options),
	).Err(); err != nil {
		log.Warnf("failed to set to redis: %v", err)
		return nil, status.Errorf(codes.Unavailable, "redis error")
	}
	return &v1pb.CacheSetResponse{}, nil
}

func (srv *cacheServer) Add(
	ctx context.Context, req *v1pb.CacheAddRequest) (
	_ *v1pb.CacheAddResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	ok, err := srv.cli.SetNX(
		ctx,
		getCacheKey(trusted.AppId, req.Key),
		req.Value,
		getTimeout(req.Options)).Result()
	if err != nil {
		log.Warnf("failed to set to redis: %v", err)
		return nil, status.Errorf(codes.Unavailable, "redis error")
	}
	if !ok {
		return nil, status.Errorf(codes.AlreadyExists, "already exists")
	}
	return &v1pb.CacheAddResponse{}, nil
}
