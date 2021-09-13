package registry

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"

	"github.com/ntons/log-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ntons/libra/librad/internal/redis"
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
	dbAppCollection  *mongo.Collection
	dbUserCollection = make(map[string]*mongo.Collection)
	dbRoleCollection = make(map[string]*mongo.Collection)

	// app cache loaded from database
	xApps = newAppIndex(nil)

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

type xApp struct {
	// 应用ID
	Id string `bson:"_id"`
	// 数值形式的应用ID
	Key uint32 `bson:"key"`
	// 应用签名密钥，授权访问
	Secret string `bson:"secret,omitempty"`
	// 应用指纹指纹，特异化应用数据，增加安全性
	Fingerprint string `bson:"fingerprint,omitempty"`
	// 允许的服务
	Permissions []*xPermission `bson:"permissions,omitempty"`
	// AES密钥，由Fingerprint生成
	block cipher.Block
}

func (x *xApp) parse() (err error) {
	// check permission expression
	for _, p := range x.Permissions {
		if err = p.parse(); err != nil {
			return
		}
	}
	// hash fingerprint to 32 bytes byte array, NewCipher must success
	hash := sha256.Sum256([]byte(x.Fingerprint))
	x.block, _ = aes.NewCipher(hash[:])
	return
}
func (x *xApp) isPermitted(path string) bool {
	for _, p := range cfg.CommonPermissions {
		if p.isPermitted(path) {
			return true
		}
	}
	for _, p := range x.Permissions {
		if p.isPermitted(path) {
			return true
		}
	}
	return false
}

// App collection with index
type xAppIndex struct {
	idIndex  map[string]*xApp
	keyIndex map[uint32]*xApp
}

func newAppIndex(apps []*xApp) *xAppIndex {
	var (
		idIndex  = make(map[string]*xApp)
		keyIndex = make(map[uint32]*xApp)
	)
	for _, a := range apps {
		idIndex[a.Id] = a
		keyIndex[a.Key] = a
	}
	return &xAppIndex{idIndex: idIndex, keyIndex: keyIndex}
}

func findAppById(id string) *xApp {
	a, _ := xApps.idIndex[id]
	return a
}
func findAppByKey(key uint32) *xApp {
	a, _ := xApps.keyIndex[key]
	return a
}

// 会话缓存数据
type dbSessData struct {
	RoleId    string `msgpack:"roleId"`
	RoleIndex uint32 `msgpack:"roleIndex"`
}
type dbSess struct {
	AppId  string     `msgpack:"-"`
	UserId string     `msgpack:"-"`
	Token  string     `msgpack:"token"`
	Data   dbSessData `msgpack:"data"`
	//// 中转数据
	app *xApp `msgpack:"-"`
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
	// load app configurations from database
	loadApps := func(ctx context.Context) (err error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		collection, err := getAppCollection(ctx)
		if err != nil {
			return fmt.Errorf("failed to get app collection: %w", err)
		}
		cursor, err := collection.Find(ctx, bson.D{})
		if err != nil {
			return fmt.Errorf("failed to query apps: %w", err)
		}
		var res []*xApp
		if err = cursor.All(ctx, &res); err != nil {
			return
		}
		for _, a := range res {
			if err = a.parse(); err != nil {
				return
			}
		}
		xApps = newAppIndex(res)
		return
	}
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
