package syncer

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/log-go"
	"github.com/ntons/redmon"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type task struct {
	name string
	dbs  []*redmon.Client
}

func dial(name, url string, urls []string) (t *task, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mdb *mongo.Client
	if mdb, err = mongo.NewClient(options.Client().ApplyURI(url)); err != nil {
		return
	}
	if err = mdb.Connect(ctx); err != nil {
		return
	}

	var rdbs = make([]*redis.Client, 0, len(urls))
	for _, url := range urls {
		var opts *redis.Options
		if opts, err = redis.ParseURL(url); err != nil {
			return
		}
		rdb := redis.NewClient(opts)
		if err = rdb.Ping(ctx).Err(); err != nil {
			return
		}
		rdbs = append(rdbs, rdb)
	}

	t = &task{name: name}
	for i, rdb := range rdbs {
		t.dbs = append(t.dbs,
			redmon.NewClient(
				rdb,
				mdb,
				redmon.OnSyncSave(t.onSave),
				redmon.OnSyncIdle(t.onIdle),
				redmon.OnSyncFail(t.onFail),
			),
		)
		log.Infof("%s: %s => %s", name, urls[i], url)
	}

	return
}

func (t *task) Serve(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()

	for _, db := range t.dbs {
		wg.Add(1)
		go func(db *redmon.Client) {
			defer wg.Done()
			db.Sync(ctx)
		}(db)
	}
}

func (t *task) onSave(key string) time.Duration {
	log.Debugf("OnSave: %s, %s", t.name, key)
	return 0
}
func (t *task) onIdle() time.Duration {
	return time.Second
}
func (t *task) onFail(err error) time.Duration {
	log.Warnf("OnFail: %s, %s", t.name, err)
	return time.Second
}
