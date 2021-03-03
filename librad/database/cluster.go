package database

import (
	"context"
	"hash/crc32"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/ntons/libra/librad/comm/util"
)

type redisCluster []*redis.Client

func (cluster redisCluster) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return cluster[cluster.hash(key)].SetNX(ctx, key, value, expiration)
}

func (cluster redisCluster) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (cmd *redis.Cmd) {
	if len(keys) == 0 {
		panic("no key to hash")
	} else if len(keys) == 1 {
		return cluster[cluster.hash(keys[0])].EvalSha(
			ctx, sha1, keys, args...)
	} else {
		for n, keys := range cluster.hashKeys(keys) {
			if len(keys) == 0 {
				continue
			}
			if cmd = cluster[n].EvalSha(
				ctx, sha1, keys, args...); cmd.Err() != nil {
				return
			}
		}
	}
	return
}

func (cluster redisCluster) ScriptLoad(ctx context.Context, script string) (cmd *redis.StringCmd) {
	for _, cli := range cluster {
		if cmd = cli.ScriptLoad(ctx, script); cmd.Err() != nil {
			return
		}
	}
	return
}

func (cluster redisCluster) hash(key string) uint32 {
	return crc32.ChecksumIEEE(util.StringToBytes(key)) % uint32(len(cluster))
}
func (cluster redisCluster) hashKeys(keys []string) [][]string {
	hashedKeys := make([][]string, len(cluster))
	for _, key := range keys {
		n := cluster.hash(key)
		hashedKeys[n] = append(hashedKeys[n], key)
	}
	return hashedKeys
}
