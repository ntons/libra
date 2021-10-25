package db

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ntons/log-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ntons/libra/librad/common/redis"
)

// |- config |- apps
//
// |- app1   |- users
//           |- roles
//
// |- app2   |- users
//           |- roles

const (
	dbMaxAcctPerUser = 100
	// 会话最长生命周期，即使一直在线，也会强制清除
	dbSessTTL = 24 * time.Hour
)

var (
	mdb *mongo.Client

	rdbAuth  redis.Client
	rdbNonce redis.Client

	// cached collection
	dbAppCollection   *mongo.Collection
	dbAdminCollection *mongo.Collection
	dbUserCollection  = make(map[string]*mongo.Collection)
	dbRoleCollection  = make(map[string]*mongo.Collection)

	// app cache loaded from database
	xApps = newAppIndex(nil)
	// admin cache loaded from database
	xAdmins = newAdminIndex(nil)

	// 只更新会话数据
	luaUpdateSessData = redis.NewScript(`
local b = redis.call("GET", KEYS[1])
if not b then return Nil end
local d = cmsgpack.unpack(b)
d.data = cmsgpack.unpack(ARGV[1])
return redis.call("SETEX", KEYS[1], %d, cmsgpack.pack(d))`,
		dbSessTTL/time.Second,
	)
)

// 会话缓存数据
type SessData struct {
	RoleId    string `msgpack:"roleId"`
	RoleIndex uint32 `msgpack:"roleIndex"`
}
type Sess struct {
	AppId  string   `msgpack:"-"`
	UserId string   `msgpack:"-"`
	Token  string   `msgpack:"token"`
	Data   SessData `msgpack:"data"`
	//// 中转数据
	App *App `msgpack:"-"`
}

func dialMongo(ctx context.Context) (_ *mongo.Client, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cli, err := mongo.NewClient(options.Client().ApplyURI(cfg.Mongo))
	if err != nil {
		return nil, fmt.Errorf("failed to new mongo client: %w", err)
	}
	if err = cli.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect mongo: %w", err)
	}
	return cli, nil
}

func dialDatabase(ctx context.Context) (err error) {
	if rdbAuth, err = redis.DialCluster(ctx, cfg.Auth.Redis); err != nil {
		return
	}
	if rdbNonce, err = redis.DialCluster(ctx, cfg.Nonce.Redis); err != nil {
		return
	}
	if mdb, err = dialMongo(ctx); err != nil {
		return
	}
	return
}

func dbServe(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := loadApps(ctx); err != nil {
				log.Warnf("failed to load apps: %v", err)
			}
			jitter := time.Duration(rand.Int63n(int64(30 * time.Second)))
			select {
			case <-ctx.Done():
				return
			case <-time.After(45*time.Second + jitter): // [45s,75s)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := loadAdmins(ctx); err != nil {
				log.Warnf("failed to load apps: %v", err)
			}
			jitter := time.Duration(rand.Int63n(int64(30 * time.Second)))
			select {
			case <-ctx.Done():
				return
			case <-time.After(45*time.Second + jitter): // [45s,75s)
			}
		}
	}()

	<-ctx.Done()
}

// get app collection
func getAppCollection(ctx context.Context) (*mongo.Collection, error) {
	if dbAppCollection != nil {
		return dbAppCollection, nil
	}
	const collectionName = "apps"
	collection := mdb.Database(cfg.ConfigDBName).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	dbAppCollection = collection
	return collection, nil
}

func getAdminCollection(ctx context.Context) (*mongo.Collection, error) {
	if dbAdminCollection != nil {
		return dbAdminCollection, nil
	}
	const collectionName = "admins"
	collection := mdb.Database(cfg.ConfigDBName).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	dbAdminCollection = collection
	return collection, nil
}
