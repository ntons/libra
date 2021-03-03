package syncer

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/log-go"
	"github.com/ntons/remon"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type task struct {
	name    string
	quit    chan struct{}
	syncers []*remon.Syncer
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

	t = &task{
		name: name,
		quit: make(chan struct{}, 1),
	}
	for i, rdb := range rdbs {
		t.syncers = append(t.syncers,
			remon.NewSyncer(
				rdb,
				mdb,
				remon.OnSyncSave(t.onSave),
				remon.OnSyncIdle(t.onIdle),
				remon.OnSyncError(t.onError),
			),
		)
		log.Infof("%s: %s => %s", name, urls[i], url)
	}

	return
}

func (t *task) Serve() {
	var wg sync.WaitGroup
	defer wg.Wait()

	for _, s := range t.syncers {
		wg.Add(1)
		go func(s *remon.Syncer) {
			defer wg.Done()
			s.Serve()
		}(s)
		defer s.Stop()
	}

	<-t.quit
}
func (t *task) Stop() {
	select {
	case t.quit <- struct{}{}:
	default:
	}
}

func (t *task) onSave(key string) time.Duration {
	log.Debugf("OnSave: %s, %s", t.name, key)
	return 0
}
func (t *task) onIdle() time.Duration {
	return time.Second
}
func (t *task) onError(err error) time.Duration {
	log.Warnf("OnError: %s, %s", t.name, err)
	return time.Second
}
