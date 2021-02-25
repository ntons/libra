package database

import (
	"context"
	"hash/crc32"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/remon"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ntons/libra/librad/internal/comm"
)

var _ remon.Client = (*client)(nil)

// 分布式ReMon客户端，目前只允许Redis集群
type client struct {
	pool []remon.Client
}

func dial(ctx context.Context, redisurls []string, mongourl string) (_ *client, err error) {
	m, err := mongo.NewClient(options.Client().ApplyURI(mongourl))
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err = m.Connect(ctx); err != nil {
		return
	}
	var pool []remon.Client
	for _, url := range redisurls {
		var o *redis.Options
		if o, err = redis.ParseURL(url); err != nil {
			m.Disconnect(ctx)
			return
		}
		pool = append(pool, remon.New(redis.NewClient(o), m))
	}
	return &client{pool: pool}, nil
}

func (cli *client) Stat() remon.Stat {
	return remon.Stat{}
}

func (cli *client) Get(ctx context.Context, key string, opts ...remon.GetOption) (rev int64, val string, err error) {
	return cli.hash(key).Get(ctx, key, opts...)
}

func (cli *client) GetBytes(ctx context.Context, key string, opts ...remon.GetOption) (int64, []byte, error) {
	return cli.hash(key).GetBytes(ctx, key, opts...)
}

func (cli *client) Set(ctx context.Context, key, val string) (rev int64, err error) {
	return cli.hash(key).Set(ctx, key, val)
}

func (cli *client) SetBytes(ctx context.Context, key string, val []byte) (rev int64, err error) {
	return cli.hash(key).SetBytes(ctx, key, val)
}

func (cli *client) Add(ctx context.Context, key, val string) (err error) {
	return cli.hash(key).Add(ctx, key, val)
}

func (cli *client) AddBytes(ctx context.Context, key string, val []byte) (err error) {
	return cli.hash(key).AddBytes(ctx, key, val)
}

func (cli *client) Eval(ctx context.Context, script *remon.Script, key string, args ...interface{}) (cmd *redis.Cmd) {
	return cli.hash(key).Eval(ctx, script, key, args...)
}

func (cli *client) hash(key string) remon.Client {
	return cli.pool[int(crc32.ChecksumIEEE(comm.S2B(key)))%len(cli.pool)]
}
