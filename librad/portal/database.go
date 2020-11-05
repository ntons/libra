package portal

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/go-redis/redis/v8"
	"github.com/ntons/log-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mdMap map[string]string

// Schemes:
type dbApp struct {
	// 应用ID
	Id string `bson:"_id"`
	// 数值形式的应用ID
	Key uint32 `bson:"key"`
	// 应用签名密钥，授权访问
	Secret string `bson:"secret,omitempty"`
	// 应用指纹指纹，特异化应用数据，增加安全性
	Fingerprint string `bson:"fingerprint,omitempty"`

	// AES密钥，由Fingerprint生成
	block cipher.Block
}

type dbAppMap map[string]*dbApp

type dbUser struct {
	Id string `bson:"_id"`
	// 用户账号列表，其中任意一个匹配都可以认定为该用户
	// 常见的用例为：
	// 1. 游客账号/正式账号
	// 2. 平台账号/第三方账号
	AcctId []string `bson:"acct_id,omitempty"`
	// 创建时间
	CreateTime time.Time `bson:"create_time,omitempty"`
	// 上次登录时间
	LoginTime time.Time `bson:"login_time,omitempty"`
	// 元数据
	Metadata map[string]string `bson:"metadata,omitempty"`
}

type dbRole struct {
	Id string `bson:"_id"`
	// 角色序号，主要有一下几个用途
	// 1. 创建用户发生重试时保证只有唯一一个角色被成功创建
	// 2. 用来确定该用户的角色分类，比如分区分服
	Index int32 `bson:"index,omitempty"`
	// 所属用户ID
	UserId string `bson:"user_id,omitempty"`
	// 创建时间
	CreateTime time.Time `bson:"create_time,omitempty"`
	// 上次登录时间
	SignInTime time.Time `bson:"sign_in_time,omitempty"`
	// 元数据
	Metadata map[string]string `bson:"metadata,omitempty"`
}

type database struct {
	r *redis.Client
	m *mongo.Client
	// database name prefix
	dbNamePrefix string
	//
	ctx    context.Context
	cancel context.CancelFunc
	//
	appCollection     *mongo.Collection
	appUserCollection map[string]*mongo.Collection
	appRoleCollection map[string]*mongo.Collection
	// app map pointer
	apps unsafe.Pointer
}

func dialRedis(cfg *config) (_ *redis.Client, err error) {
	o, err := redis.ParseURL(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %v", err)
	}
	return redis.NewClient(o), nil
}
func dialMongo(ctx context.Context, cfg *config) (_ *mongo.Client, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cli, err := mongo.NewClient(options.Client().ApplyURI(cfg.Mongo))
	if err != nil {
		return nil, fmt.Errorf("failed to new mongo client: %v", err)
	}
	if err = cli.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect mongo: %v", err)
	}
	return cli, nil
}
func newDatabase(ctx context.Context, cfg *config) (_ *database, err error) {
	r, err := dialRedis(cfg)
	if err != nil {
		return
	}
	m, err := dialMongo(ctx, cfg)
	if err != nil {
		return
	}
	db := &database{
		r:                 r,
		m:                 m,
		dbNamePrefix:      "librad_",
		ctx:               ctx,
		appUserCollection: make(map[string]*mongo.Collection),
		appRoleCollection: make(map[string]*mongo.Collection),
	}
	apps := make(map[string]*dbApp)
	atomic.StorePointer(&db.apps, unsafe.Pointer(&apps))
	return db, nil
}
func (db *database) Serve() {
	for {
		if err := db.loadApps(db.ctx); err != nil {
			log.Warn("failed to load apps: %v", err)
		}
		jitter := time.Duration(rand.Int63n(int64(30 * time.Second)))
		select {
		case <-db.ctx.Done():
			return
		case <-time.After(45*time.Second + jitter):
			// [45s,75s)
		}
	}
}

// database and collection dispatch
func (db *database) getDatabase(appId string) *mongo.Database {
	return db.m.Database(db.dbNamePrefix + appId)
}
func (db *database) getAppCollection(
	ctx context.Context) (*mongo.Collection, error) {
	if db.appCollection != nil {
		return db.appCollection, nil
	}
	const collectionName = "apps"
	dbName := db.dbNamePrefix + "config"
	collection := db.m.Database(dbName).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.M{"key": 1},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %v", err)
	}
	db.appCollection = collection
	return collection, nil
}
func (db *database) getUserCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	const collectionName = "users"
	if collection, ok := db.appUserCollection[appId]; ok {
		return collection, nil
	}
	collection := db.getDatabase(appId).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.M{"acct_id": 1},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %v", err)
	}
	db.appUserCollection[appId] = collection
	return collection, nil
}
func (db *database) getRoleCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	const collectionName = "roles"
	if collection, ok := db.appRoleCollection[appId]; ok {
		return collection, nil
	}
	collection := db.getDatabase(appId).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.M{"user_id": 1, "index": 1},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %v", err)
	}
	db.appRoleCollection[appId] = collection
	return collection, nil
}

// load apps from database
func (db *database) loadApps(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection, err := db.getAppCollection(ctx)
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to query apps: %v", err)
	}
	var res []*dbApp
	if err = cursor.All(ctx, &res); err != nil {
		return
	}
	index := make(map[string]*dbApp)
	for _, app := range res {
		key := sha256.Sum256([]byte(app.Fingerprint)) // 32 bytes/AES-256
		if app.block, err = aes.NewCipher(key[:]); err != nil {
			return fmt.Errorf("invalid aes key")
		}
		index[app.Id] = app
	}
	atomic.StorePointer(&db.apps, unsafe.Pointer(&index))
	return
}
func (db *database) getApp(appId string) (app *dbApp, err error) {
	app, ok := (*(*dbAppMap)(atomic.LoadPointer(&db.apps)))[appId]
	if !ok {
		return nil, errAppIdNotFound
	}
	return
}

// token
func (db *database) newToken(
	ctx context.Context, app *dbApp, userId string) (_ string, err error) {
	token := newToken(app, userId)
	if err = db.r.Set(ctx, tokenKey(userId), token, 0).Err(); err != nil {
		return
	}
	return token, nil
}
func (db *database) checkToken(
	ctx context.Context, app *dbApp, token string) (userId string, err error) {
	if userId, err = decToken(app, token); err != nil {
		return
	}
	value, err := db.r.Get(ctx, tokenKey(userId)).Result()
	if err != nil {
		return
	}
	if value != token {
		return "", errInvalidToken
	}
	return
}

// ticket
func (db *database) newTicket(
	ctx context.Context, app *dbApp, roleId string) (_ string, err error) {
	ticket := newTicket(app, roleId)
	if err = db.r.Set(ctx, ticketKey(roleId), ticket, 0).Err(); err != nil {
		return
	}
	return ticket, nil
}
func (db *database) checkTicket(
	ctx context.Context, app *dbApp, ticket string) (roleId string, err error) {
	if roleId, err = decTicket(app, ticket); err != nil {
		return
	}
	value, err := db.r.Get(ctx, ticketKey(roleId)).Result()
	if err != nil {
		return
	}
	if value != ticket {
		return "", errInvalidTicket
	}
	return
}

// user
func (db *database) getUser(
	ctx context.Context, appId, userId string) (_ *dbUser, err error) {
	collection, err := db.getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	user := &dbUser{}
	if err = collection.FindOne(
		ctx, bson.M{"_id": userId}).Decode(user); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errUserNotFound
		}
		return
	}
	return user, nil
}
func (db *database) loginUser(
	ctx context.Context, app *dbApp, acctId []string) (_ *dbUser, err error) {
	collection, err := db.getUserCollection(ctx, app.Id)
	if err != nil {
		return
	}
	now := time.Now()
	user := &dbUser{
		Id:         newUserId(app.Key),
		CreateTime: now,
	}
	if err = collection.FindOneAndUpdate(
		ctx,
		bson.M{"acct_id": bson.M{"$elemMatch": bson.M{"$in": acctId}}},
		bson.M{
			"$set":         bson.M{"login_time": now},
			"$addToSet":    bson.M{"acct_id": bson.M{"$each": acctId}},
			"$setOnInsert": user,
		},
		options.FindOneAndUpdate().SetUpsert(true),
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(user); err != nil {
		return
	}
	return user, nil
}

func (db *database) bindAcctIdToUser(
	ctx context.Context, appId, userId, acctId string) (err error) {
	collection, err := db.getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	if _, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": userId},
		bson.M{"$addToSet": bson.M{"acct_id": acctId}},
	); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errRoleNotFound
		}
		return
	}
	return
}

// role
func (db *database) getRole(
	ctx context.Context, appId, roleId string) (_ *dbRole, err error) {
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &dbRole{}
	if err = collection.FindOne(
		ctx, bson.M{"_id": roleId}).Decode(role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errRoleNotFound
		}
		return
	}
	return role, nil
}
func (db *database) listRoles(
	ctx context.Context, appId, userId string) (_ []*dbRole, err error) {
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	cur, err := collection.Find(ctx, bson.M{"user_id": userId})
	if err != nil {
		return
	}
	var roles []*dbRole
	if err = cur.All(ctx, &roles); err != nil {
		return
	}
	return roles, nil
}
func (db *database) createRole(
	ctx context.Context, app *dbApp, userId string, index int32) (
	_ *dbRole, err error) {
	collection, err := db.getRoleCollection(ctx, app.Id)
	if err != nil {
		return
	}
	role := &dbRole{
		Id:         newRoleId(app.Key),
		UserId:     userId,
		Index:      index,
		CreateTime: time.Now(),
	}
	if _, err = collection.InsertOne(ctx, role); err != nil {
		return
	}
	return role, nil
}
func (db *database) signInRole(
	ctx context.Context, appId, roleId string) (_ *dbRole, err error) {
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	now := time.Now()
	role := &dbRole{}
	if err = collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": roleId},
		bson.M{"$set": bson.M{"sign_in_time": now}},
	).Decode(&role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errRoleNotFound
		}
		return
	}
	return
}

func (db *database) setUserMetadata(
	ctx context.Context, appId, userId string, md map[string]string) (
	err error) {
	collection, err := db.getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	set, unset := bson.M{}, bson.M{}
	for key, val := range md {
		if val != "" {
			set["metadata."+key] = val
		} else {
			unset["metadata."+key] = 1
		}
	}
	if _, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": userId},
		bson.M{"$set": set, "$unset": unset},
	); err != nil {
		return
	}
	return
}
func (db *database) setRoleMetadata(
	ctx context.Context, appId, userId, roleId string, md map[string]string) (
	err error) {
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	set, unset := bson.M{}, bson.M{}
	for key, val := range md {
		if val != "" {
			set["metadata."+key] = val
		} else {
			unset["metadata."+key] = 1
		}
	}
	if _, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": roleId, "user_id": userId}, // role must belong to user
		bson.M{"$set": set, "$unset": unset},
	); err != nil {
		return
	}
	return
}
