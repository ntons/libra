package syncer

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/remon"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ntons/libra/librad/internal/comm"
)

func init() {
	comm.RegisterService("syncer", create)
}

type server struct {
	ctx  context.Context
	stop context.CancelFunc
	clis []*remon.SyncClient
}

func create(b json.RawMessage) (_ comm.Service, err error) {
	if err = json.Unmarshal(b, cfg); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var clis []*remon.SyncClient

	// dail to database remons
	m1, err := mongo.NewClient(options.Client().ApplyURI(cfg.Database.Mongo))
	if err != nil {
		return
	}
	if err = m1.Connect(ctx); err != nil {
		return
	}
	for _, url := range cfg.Database.Redis {
		var o *redis.Options
		if o, err = redis.ParseURL(url); err != nil {
			return
		}
		clis = append(clis, remon.NewSync(redis.NewClient(o), m1))
	}

	// dial to mailbox remons
	m2, err := mongo.NewClient(options.Client().ApplyURI(cfg.MailBox.Mongo))
	if err != nil {
		return
	}
	if err = m2.Connect(ctx); err != nil {
		return
	}
	for _, url := range cfg.MailBox.Redis {
		var o *redis.Options
		if o, err = redis.ParseURL(url); err != nil {
			return
		}
		clis = append(clis, remon.NewSync(redis.NewClient(o), m2))
	}

	ctx, stop := context.WithCancel(context.Background())
	return &server{ctx: ctx, stop: stop, clis: clis}, nil
}

func (srv *server) Serve() {
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, cli := range srv.clis {
		wg.Add(1)
		go func(cli *remon.SyncClient) {
			defer wg.Done()
			cli.Serve()
		}(cli)
		defer cli.Stop()
	}
	<-srv.ctx.Done()
}

func (srv *server) Stop() { srv.stop() }
