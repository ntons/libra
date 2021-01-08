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

// |- librad_config |- apps
//
// |- librad_app1   |- users
//                  |- roles
//
// |- librad_app2   |- users
//                  |- roles

const (
	// database name prefix
	dbNamePrefix = "librad_"
)

var (
	// global database instance
	db *database
)

// Schemes:
type xApp struct {
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

type xUser struct {
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

type xRole struct {
	Id string `bson:"_id"`
	// 角色序号，主要有一下几个用途
	// 1. 创建用户发生重试时保证只有唯一一个角色被成功创建
	// 2. 用来确定该用户的角色分类，比如分区分服
	Index uint32 `bson:"index,omitempty"`
	// 所属用户ID
	UserId string `bson:"user_id,omitempty"`
	// 创建时间
	CreateTime time.Time `bson:"create_time,omitempty"`
	// 上次登录时间
	SignInTime time.Time `bson:"sign_in_time,omitempty"`
	// 元数据
	Metadata map[string]string `bson:"metadata,omitempty"`
}

// app manager index apps by id and key
type xAppMgr struct {
	idIndex  map[string]*xApp
	keyIndex map[uint32]*xApp
}

func newAppMgr(list []*xApp) *xAppMgr {
	mgr := &xAppMgr{
		idIndex:  make(map[string]*xApp),
		keyIndex: make(map[uint32]*xApp),
	}
	for _, app := range list {
		// hash fingerprint to 32 bytes byte array, NewCipher must success
		hash := sha256.Sum256([]byte(app.Fingerprint))
		app.block, _ = aes.NewCipher(hash[:])
		// id and key must unique due to db index
		mgr.idIndex[app.Id] = app
		mgr.keyIndex[app.Key] = app
	}
	return mgr
}
func (mgr *xAppMgr) findById(appId string) *xApp {
	return mgr.idIndex[appId]
}
func (mgr *xAppMgr) findByKey(appKey uint32) *xApp {
	return mgr.keyIndex[appKey]
}

func tokenKey(userId string) string {
	return fmt.Sprintf("token:{%s}", userId)
}
func ticketKey(roleId string) string {
	return fmt.Sprintf("ticket:{%s}", roleId)
}

type database struct {
	// life-time context
	ctx context.Context
	// db handlers
	r *redis.Client
	m *mongo.Client
	// collection handlers
	appCollection     *mongo.Collection
	appUserCollection map[string]*mongo.Collection
	appRoleCollection map[string]*mongo.Collection
	// app map pointer
	apps unsafe.Pointer
}

func dialRedis(cfg *config) (_ *redis.Client, err error) {
	o, err := redis.ParseURL(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}
	return redis.NewClient(o), nil
}
func dialMongo(ctx context.Context, cfg *config) (_ *mongo.Client, err error) {
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
func dialDatabase(ctx context.Context, cfg *config) (_ *database, err error) {
	r, err := dialRedis(cfg)
	if err != nil {
		return
	}
	m, err := dialMongo(ctx, cfg)
	if err != nil {
		return
	}
	db := &database{
		ctx:               ctx,
		r:                 r,
		m:                 m,
		appUserCollection: make(map[string]*mongo.Collection),
		appRoleCollection: make(map[string]*mongo.Collection),
	}
	atomic.StorePointer(&db.apps, unsafe.Pointer(newAppMgr([]*xApp{})))
	return db, nil
}
func (db *database) Serve() {
	for {
		if err := db.loadApps(db.ctx); err != nil {
			log.Warnf("failed to load apps: %v", err)
		}
		jitter := time.Duration(rand.Int63n(int64(30 * time.Second)))
		select {
		case <-db.ctx.Done():
			return
		case <-time.After(45*time.Second + jitter): // [45s,75s)
		}
	}
}

// get app collection
func (db *database) getAppCollection(
	ctx context.Context) (*mongo.Collection, error) {
	if db.appCollection != nil {
		return db.appCollection, nil
	}
	const collectionName = "apps"
	dbName := dbNamePrefix + "config"
	collection := db.m.Database(dbName).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.M{"key": 1},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	db.appCollection = collection
	return collection, nil
}

// get user collection of app
func (db *database) getUserCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	const collectionName = "users"
	if collection, ok := db.appUserCollection[appId]; ok {
		return collection, nil
	}
	collection := db.m.Database(dbNamePrefix + appId).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.M{"acct_id": 1},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	db.appUserCollection[appId] = collection
	return collection, nil
}

// get role collection of app
func (db *database) getRoleCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	const collectionName = "roles"
	if collection, ok := db.appRoleCollection[appId]; ok {
		return collection, nil
	}
	collection := db.m.Database(dbNamePrefix + appId).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.M{"user_id": 1, "index": 1},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	db.appRoleCollection[appId] = collection
	return collection, nil
}

// load app configurations from database
func (db *database) loadApps(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection, err := db.getAppCollection(ctx)
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
	atomic.StorePointer(&db.apps, unsafe.Pointer(newAppMgr(res)))
	return
}

// get app by id or key, nil will be returned when not exists
func (db *database) getAppById(appId string) *xApp {
	return (*xAppMgr)(atomic.LoadPointer(&db.apps)).findById(appId)
}
func (db *database) getAppByKey(appKey uint32) *xApp {
	return (*xAppMgr)(atomic.LoadPointer(&db.apps)).findByKey(appKey)
}

// generate a new token
func (db *database) newToken(
	ctx context.Context, app *xApp, userId string) (_ string, err error) {
	id, ok := decId(userId)
	if !ok {
		return "", fmt.Errorf("failed to decode user id: %s", userId)
	}
	token := encCred(newCred(id, app.block))
	if err = db.r.Set(ctx, tokenKey(userId), token, 0).Err(); err != nil {
		return
	}
	return token, nil
}

// check whether a token is available and retrieve the associated data
func (db *database) checkToken(
	ctx context.Context, token string) (appId, userId string, err error) {
	cred, ok := decCred(token)
	if !ok {
		return "", "", errInvalidToken
	}
	app := db.getAppByKey(getAppKeyFromCred(cred))
	if app == nil {
		return "", "", errInvalidToken
	}
	id, ok := getIdFromCred(cred, app.block)
	if !ok {
		return "", "", errInvalidToken
	}
	appId, userId = app.Id, encId(id)
	if target, err := db.r.Get(ctx, tokenKey(userId)).Result(); err != nil {
		if err == redis.Nil {
			return "", "", errInvalidToken
		} else {
			log.Warnf("failed to get token from redis: %v", err)
			return "", "", errDatabaseUnavailable
		}
	} else if target != token {
		return "", "", errInvalidToken
	}
	return
}

// generate a new ticket
func (db *database) newTicket(
	ctx context.Context, appId, roleId string) (_ string, err error) {
	ticket := "" //newTicket(appId, roleId)
	if err = db.r.Set(ctx, ticketKey(roleId), ticket, 0).Err(); err != nil {
		return
	}
	return ticket, nil
}

// check whether a ticket is available and retrieve the associated data
func (db *database) checkTicket(
	ctx context.Context, ticket string) (appId, roleId string, err error) {
	cred, ok := decCred(ticket)
	if !ok {
		return "", "", errInvalidTicket
	}
	app := db.getAppByKey(getAppKeyFromCred(cred))
	if app == nil {
		return "", "", errInvalidTicket
	}
	id, ok := getIdFromCred(cred, app.block)
	if !ok {
		return "", "", errInvalidTicket
	}
	appId, roleId = app.Id, encId(id)
	if target, err := db.r.Get(ctx, ticketKey(roleId)).Result(); err != nil {
		if err == redis.Nil {
			return "", "", errInvalidToken
		} else {
			log.Warnf("failed to get ticket from redis: %v", err)
			return "", "", errDatabaseUnavailable
		}
	} else if target != ticket {
		return "", "", errInvalidTicket
	}
	return
}

// get user from database by id
func (db *database) getUser(
	ctx context.Context, appId, userId string) (_ *xUser, err error) {
	collection, err := db.getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	user := &xUser{}
	if err = collection.FindOne(
		ctx, bson.M{"_id": userId}).Decode(user); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errUserNotFound
		}
		return
	}
	return user, nil
}

// login user by acctId
func (db *database) loginUser(
	ctx context.Context, app *xApp, acctId []string) (_ *xUser, err error) {
	collection, err := db.getUserCollection(ctx, app.Id)
	if err != nil {
		return
	}
	now := time.Now()
	user := &xUser{
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
		log.Warnf("failed to access mongo: %v", err)
		return nil, errDatabaseUnavailable
	}
	return user, nil
}

// bind a new acctId to existed user
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
			return errUserNotFound
		} else {
			log.Warnf("failed to access mongo: %v", err)
			return errDatabaseUnavailable
		}
	}
	return
}

// set user's metadata
func (db *database) setUserMetadata(
	ctx context.Context, appId, userId string,
	md map[string]string) (err error) {
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
		log.Warnf("failed to access user: %v", err)
		return errDatabaseUnavailable
	}
	return
}

// get role by id
func (db *database) getRole(
	ctx context.Context, appId, roleId string) (_ *xRole, err error) {
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &xRole{}
	if err = collection.FindOne(
		ctx, bson.M{"_id": roleId}).Decode(role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errRoleNotFound
		}
		return
	}
	return role, nil
}

// list all roles of user
func (db *database) listRoles(
	ctx context.Context, appId, userId string) (_ []*xRole, err error) {
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	cur, err := collection.Find(ctx, bson.M{"user_id": userId})
	if err != nil {
		return
	}
	var roles []*xRole
	if err = cur.All(ctx, &roles); err != nil {
		return
	}
	return roles, nil
}

// create a new role of user
func (db *database) createRole(
	ctx context.Context, appId, userId string, index uint32) (
	_ *xRole, err error) {
	app := db.getAppById(appId)
	if app == nil {
		return nil, errInvalidAppId
	}
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &xRole{
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

// sign-in a role
func (db *database) signInRole(
	ctx context.Context, appId, roleId string) (_ *xRole, err error) {
	collection, err := db.getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &xRole{}
	if err = collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": roleId},
		bson.M{"$set": bson.M{"sign_in_time": time.Now()}},
	).Decode(&role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errRoleNotFound
		}
		return
	}
	return
}

// get role's metadata
func (db *database) setRoleMetadata(
	ctx context.Context, appId, userId, roleId string,
	md map[string]string) (err error) {
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
