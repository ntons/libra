package redis

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sigurn/crc16"

	"github.com/ntons/libra/librad/internal/util"
)

type clusterOptions struct {
	hashTag bool
}

type ClusterOption interface {
	apply(o *clusterOptions)
}

type funcClusterOption struct {
	fn func(o *clusterOptions)
}

func (f funcClusterOption) apply(o *clusterOptions) {
	if f.fn != nil {
		f.fn(o)
	}
}

func WithHashTag() ClusterOption {
	return funcClusterOption{func(o *clusterOptions) { o.hashTag = true }}
}

var _ Client = (*Cluster)(nil)

type Cluster struct {
	p []Client
	o clusterOptions
}

func DialCluster(
	ctx context.Context, urls []string, opts ...ClusterOption) (
	_ *Cluster, err error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("one node at lease")
	} else if len(urls) > math.MaxUint16 {
		return nil, fmt.Errorf("%d node at most", math.MaxUint16)
	}

	cluster := &Cluster{p: make([]Client, 0, len(urls))}
	for _, opt := range opts {
		opt.apply(&cluster.o)
	}

	ropts := make([]*redis.Options, len(urls))
	for i, url := range urls {
		if ropts[i], err = redis.ParseURL(url); err != nil {
			return
		}
	}
	defer func() {
		if err != nil {
			cluster.Close()
		}
	}()
	for _, ropt := range ropts {
		rdb := redis.NewClient(ropt)
		if err = rdb.Ping(ctx).Err(); err != nil {
			return
		}
		cluster.p = append(cluster.p, rdb)
	}
	return cluster, nil
}

func (cluster *Cluster) Close() error {
	for _, cli := range cluster.p {
		cli.Close()
	}
	cluster.p = nil
	return nil
}

func (cluster *Cluster) Get(ctx context.Context, key string) *redis.StringCmd {
	return cluster.hashTo(key).Get(ctx, key)
}
func (cluster *Cluster) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return cluster.hashTo(key).Set(ctx, key, value, expiration)
}

func (cluster *Cluster) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return cluster.hashTo(key).SetNX(ctx, key, value, expiration)
}
func (cluster *Cluster) Del(ctx context.Context, keys ...string) (cmd *redis.IntCmd) {
	if len(keys) == 0 {
		panic("no key to hash")
	} else if len(keys) == 1 {
		return cluster.hashTo(keys[0]).Del(ctx, keys...)
	} else {
		for n, keys := range cluster.hashKeys(keys) {
			if len(keys) == 0 {
				continue
			}
			if c := cluster.p[n].Del(ctx, keys...); c.Err() != nil && c == nil {
				cmd = c
			}
		}
	}
	return
}

func (cluster *Cluster) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (cmd *redis.Cmd) {
	if len(keys) == 0 {
		panic("no key to hash")
	} else if len(keys) == 1 {
		return cluster.hashTo(keys[0]).EvalSha(ctx, sha1, keys, args...)
	} else {
		for n, keys := range cluster.hashKeys(keys) {
			if len(keys) == 0 {
				continue
			}
			if cmd = cluster.p[n].EvalSha(ctx, sha1, keys, args...); cmd.Err() != nil {
				return
			}
		}
	}
	return
}

func (cluster *Cluster) ScriptLoad(ctx context.Context, script string) (cmd *redis.StringCmd) {
	for _, cli := range cluster.p {
		if cmd = cli.ScriptLoad(ctx, script); cmd.Err() != nil {
			return
		}
	}
	return
}

var xmodem = crc16.MakeTable(crc16.CRC16_XMODEM)

func (cluster *Cluster) hashKey(key string) uint16 {
	if cluster.o.hashTag {
		if i := strings.IndexByte(key, '{'); i >= 0 && i < len(key)-2 {
			if j := strings.IndexByte(key[i+1:], '}'); j > 0 {
				key = key[i+1 : i+1+j]
			}
		}
	}
	return crc16.Checksum(
		util.StringToBytes(key), xmodem) % uint16(len(cluster.p))
}

func (cluster *Cluster) hashKeys(keys []string) [][]string {
	hashedKeys := make([][]string, len(cluster.p))
	for _, key := range keys {
		n := cluster.hashKey(key)
		hashedKeys[n] = append(hashedKeys[n], key)
	}
	return hashedKeys
}

func (cluster *Cluster) hashTo(key string) Client {
	return cluster.p[cluster.hashKey(key)]
}
