package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

const Nil = redis.Nil

type Client interface {
	Close() error
	Get(context.Context, string) *redis.StringCmd
	Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd
	SetNX(context.Context, string, interface{}, time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	EvalSha(context.Context, string, []string, ...interface{}) *redis.Cmd
	ScriptLoad(context.Context, string) *redis.StringCmd
}

func Dial(ctx context.Context, url string) (_ Client, err error) {
	ropt, err := redis.ParseURL(url)
	if err != nil {
		return
	}
	rdb := redis.NewClient(ropt)
	if err = rdb.Ping(ctx).Err(); err != nil {
		return
	}
	return rdb, nil
}
