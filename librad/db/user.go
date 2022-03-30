package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/log-go"
	"github.com/ntons/redis"
	"github.com/vmihailenco/msgpack/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ntons/libra/librad/common/util"
)

type User struct {
	Id string `bson:"_id"`
	// 用户账号列表，其中任意一个匹配都可以认定为该用户
	// 常见的用例为：
	// 1. 游客账号/正式账号
	// 2. 平台账号/第三方账号
	AcctIds []string `bson:"acct_ids,omitempty"`
	// 创建时间
	CreateAt time.Time `bson:"create_at,omitempty"`
	// 创建时IP
	CreateIp string `bson:"create_ip,omitempty"`
	// 上次登录时间
	LoginAt time.Time `bson:"login_at,omitempty"`
	// 上次登录时IP
	LoginIp string `bson:"login_ip,omitempty"`
	// 封号时间
	BanAt time.Time `bson:"ban_at,omitempty"`
	// 封号时间
	BanTo time.Time `bson:"ban_to,omitempty"`
	// 封号原因
	BanFor string `bson:"ban_for,omitempty"`
	// 元数据
	Metadata map[string]string `bson:"metadata,omitempty"`
}

// get user collection of app
func getUserCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	dbUserCollectionMu.Lock()
	defer dbUserCollectionMu.Unlock()

	if collection, ok := dbUserCollection[appId]; ok {
		return collection, nil
	}

	const tblName = "libra.users"
	dbName := getAppDBName(appId)

	collection := mdb.Database(dbName).Collection(tblName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys:    bson.D{{Key: "acct_ids", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	dbUserCollection[appId] = collection
	return collection, nil
}

func GetUser(
	ctx context.Context, appId, userId string) (_ *User, err error) {
	collection, err := getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	user := &User{}
	if err = collection.FindOne(
		ctx,
		bson.M{"_id": userId},
	).Decode(user); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ErrRoleNotFound
		} else {
			err = ErrDatabaseUnavailable
		}
		return
	}
	return user, nil
}

func GetUsersByAcctId(
	ctx context.Context, appId string, acctIds ...string) (_ []*User, err error) {
	collection, err := getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	cursor, err := collection.Find(
		ctx,
		bson.M{"acct_ids": bson.M{"$in": acctIds}},
	)
	if err != nil {
		return nil, ErrDatabaseUnavailable
	}
	var users []*User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, ErrDatabaseUnavailable
	}
	return users, nil
}

func GetUsers(
	ctx context.Context, appId string, userIds []string) (
	_ []*User, err error) {
	return getUsersWithOption(ctx, appId, userIds, nil)
}

// 批量拉取时的性能考虑
func GetUsersWithFields(
	ctx context.Context, appId string, userIds, fields []string) (
	_ []*User, err error) {
	proj := bson.M{"_id": 1}
	for _, field := range fields {
		proj[field] = 1
	}
	return getUsersWithOption(
		ctx, appId, userIds, options.Find().SetProjection(proj))
}

func getUsersWithOption(
	ctx context.Context, appId string, userIds []string,
	opt *options.FindOptions) (_ []*User, err error) {
	if len(userIds) == 0 {
		return
	}
	collection, err := getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	var opts []*options.FindOptions
	if opt != nil {
		opts = append(opts, opt)
	}
	cursor, err := collection.Find(
		ctx,
		bson.M{"_id": bson.M{"$in": userIds}},
		opts...,
	)
	if err != nil {
		return
	}
	var users []*User
	if err = cursor.All(ctx, &users); err != nil {
		return
	}
	return users, nil
}

func LoginUser(
	ctx context.Context, app *App, clientIp, deviceId string,
	acctIds []string, createIfNotFound bool) (
	_ *User, _ *Sess, err error) {
	if len(acctIds) > dbMaxAcctPerUser {
		err = newInvalidArgumentError("too many acct ids")
		return
	}

	collection, err := getUserCollection(ctx, app.Id)
	if err != nil {
		return
	}
	now := time.Now()
	user := &User{
		Id:       newUserId(app.Key),
		CreateAt: now,
		CreateIp: clientIp,
	}
	// 这里正确执行隐含了一个前置条件，acct_ids字段必须是索引。
	// 当给进来的acct_ids列表可以映射到多个User的时候addToSet必然会失败，
	// 从而可以保证参数 acct *---1 User 的映射关系成立。
	if err = collection.FindOneAndUpdate(
		ctx,
		bson.M{
			"acct_ids": bson.M{
				"$elemMatch": bson.M{
					"$in": acctIds,
				},
			},
		},
		bson.M{
			"$set": bson.M{
				"login_at": now,
				"login_ip": clientIp,
			},
			"$addToSet": bson.M{
				"acct_ids": bson.M{
					"$each": acctIds,
				},
			},
			"$setOnInsert": user,
		},
		options.FindOneAndUpdate().SetUpsert(createIfNotFound),
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(user); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ErrUserNotFound
		} else {
			log.Warnf("failed to access mongo: %v", err)
			err = ErrDatabaseUnavailable
		}
		return
	}
	if !containAcctIdPlaceholder(user.AcctIds) {
		if _, err = collection.UpdateOne(
			ctx,
			bson.M{
				"_id": user.Id,
			},
			bson.M{
				"$addToSet": bson.M{
					"acct_ids": "x$" + user.Id,
				},
			},
		); err != nil {
			log.Warnf("failed to access mongo: %v", err)
			return nil, nil, ErrDatabaseUnavailable
		}
	}

	limitUserAcctCount(ctx, collection, user)

	// 检查封禁状态
	if user.BanTo.After(now) {
		return nil, nil, newPermissionDeniedError(newErrorDetail(
			v1pb.ErrorCode_ErrorCodeBan,
			&v1pb.BanErrorDetail{
				UserId: user.Id,
				BanTo:  int32(user.BanTo.Unix()),
				BanFor: user.BanFor,
			},
		))
	}

	// 检查通用封禁
	var keys = append([]string{}, user.AcctIds...)
	if clientIp != "" {
		keys = append(keys, clientIp)
	}
	if deviceId != "" {
		keys = append(keys, deviceId)
	}
	if len(keys) > 0 {
		var block *BlockData
		if block, err = IsBlocked(ctx, app.Id, keys); err != nil {
			return nil, nil, ErrDatabaseUnavailable
		}
		if block != nil {
			return nil, nil, newPermissionDeniedError(newErrorDetail(
				v1pb.ErrorCode_ErrorCodeBan,
				&v1pb.BanErrorDetail{
					UserId: user.Id,
					BanTo:  int32(block.BanTo.Unix()),
					BanFor: block.BanFor,
				},
			))
		}
	}

	// 创建会话
	sess, err := newSess(ctx, app, user.Id)
	if err != nil {
		return
	}

	return user, sess, nil
}

func LogoutUser(ctx context.Context, userIds ...string) (err error) {
	if len(userIds) > 0 {
		if err = rdbAuth.Del(ctx, userIds...).Err(); err != nil {
			log.Warnf("failed to revoke token from redis: %v", err)
			return ErrDatabaseUnavailable
		}
	}
	return
}

func BindAcctIdToUser(
	ctx context.Context, appId, userId string,
	acctIds []string, takeOverIfDuplicated bool) (
	_ []string, err error) {
	if len(acctIds) > dbMaxAcctPerUser {
		err = newInvalidArgumentError("too many acct ids")
		return
	}
	collection, err := getUserCollection(ctx, appId)
	if err != nil {
		return
	}

	user := &User{}

	findAndBind := func(ctx context.Context) (err error) {
		if err = collection.FindOneAndUpdate(
			ctx,
			bson.M{
				"_id": userId,
			},
			bson.M{
				"$addToSet": bson.M{
					"acct_ids": bson.M{
						"$each": acctIds,
					},
				},
			},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(user); err != nil {
			if err == mongo.ErrNoDocuments {
				return ErrUserNotFound
			} else if mongo.IsDuplicateKeyError(err) {
				return ErrAcctAlreadyExists
			} else {
				log.Warnf("failed to access mongo: %v", err)
				return ErrDatabaseUnavailable
			}
		}
		return
	}

	if takeOverIfDuplicated {
		// 账号转移要在会话中执行，保证解绑和绑定操作的原子性
		if err = func() (err error) {
			var sess mongo.Session
			if sess, err = mdb.StartSession(); err != nil {
				log.Warnf("failed to start db session: %v", err)
				return ErrDatabaseUnavailable
			}
			defer sess.EndSession(ctx)
			// Transaction不能在Standalone中执行
			// 4.0 只能使用 replica set
			// 4.2 replica set 或 cluster
			if _, err = sess.WithTransaction(
				ctx,
				func(ctx mongo.SessionContext) (_ interface{}, err error) {
					// 解除已被绑定的账号
					if _, err = collection.UpdateMany(
						ctx,
						bson.M{
							"acct_ids": bson.M{
								"$elemMatch": bson.M{
									"$in": acctIds,
								},
							},
						},
						bson.M{
							"$pullAll": bson.M{
								"acct_ids": acctIds,
							},
						},
					); err != nil {
						log.Warnf("failed to access mongo: %v", err)
						return nil, ErrDatabaseUnavailable
					}
					// 绑定到当前用户
					if err = findAndBind(ctx); err != nil {
						return
					}
					return
				},
			); err != nil {
				return
			}
			return
		}(); err != nil {
			return
		}
	} else {
		// 直接绑定到当前用户
		if err = findAndBind(ctx); err != nil {
			return
		}
	}

	limitUserAcctCount(ctx, collection, user)

	return user.AcctIds, nil
}

func UnbindAcctIdFromUser(
	ctx context.Context, appId, userId string, acctIds []string) (
	_ []string, err error) {
	if containAcctIdPlaceholder(acctIds) {
		// 不允许解绑x$
		return nil, ErrInvalidAcctId
	}

	collection, err := getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	user := &User{}
	if err = collection.FindOneAndUpdate(
		ctx,
		bson.M{
			"_id": userId,
		},
		bson.M{
			"$pullAll": bson.M{
				"acct_ids": acctIds,
			},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(user); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUserNotFound
		} else {
			log.Warnf("failed to access mongo: %v", err)
			return nil, ErrDatabaseUnavailable
		}
	}
	return user.AcctIds, nil
}

// 成不成功无所谓，尽可能保证即可
func limitUserAcctCount(
	ctx context.Context, collection *mongo.Collection, user *User) {
	if len(user.AcctIds) > dbMaxAcctPerUser {
		if err := collection.FindOneAndUpdate(
			ctx,
			bson.M{
				"_id": user.Id,
			},
			bson.M{
				"$push": bson.M{
					"acct_ids": bson.M{
						"$each":  []string{},
						"$slice": -dbMaxAcctPerUser,
					},
				},
			},
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(user); err != nil {
			log.Warnf("failed to slice acct ids: %v, %v, %v",
				user.Id, len(user.AcctIds), err)
		}
	}
}

func SetUserMetadata(
	ctx context.Context, appId, userId string,
	md map[string]string) (err error) {
	collection, err := getUserCollection(ctx, appId)
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
	if len(set) > 0 || len(unset) > 0 {
		update := bson.M{}
		if len(set) > 0 {
			update["$set"] = set
		}
		if len(unset) > 0 {
			update["$unset"] = unset
		}
		if r, err := collection.UpdateOne(
			ctx,
			bson.M{"_id": userId},
			update,
		); err != nil {
			log.Warnf("failed to access user: %v", err)
			return ErrDatabaseUnavailable
		} else if r.MatchedCount == 0 {
			return ErrUserNotFound
		}
	}
	return
}

func BanUsers(
	ctx context.Context, appId string, userIds []string,
	banTo time.Time, banFor string) (err error) {
	collection, err := getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	if _, err = collection.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": userIds}},
		bson.M{"$set": bson.M{
			"ban_at":  time.Now(),
			"ban_to":  banTo,
			"ban_for": banFor,
		}},
	); err != nil {
		return
	}
	return
}

func UnbanUsers(
	ctx context.Context, appId string, userIds []string) (err error) {
	collection, err := getUserCollection(ctx, appId)
	if err != nil {
		return
	}
	if _, err = collection.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": userIds}},
		bson.M{"$unset": bson.M{
			"ban_at":  1,
			"ban_to":  1,
			"ban_for": 1,
		}},
	); err != nil {
		return
	}
	return
}

func containAcctIdPlaceholder(acctIds []string) bool {
	for _, acctId := range acctIds {
		if strings.HasPrefix(acctId, "x$") {
			return true
		}
	}
	return false
}

func newSess(
	ctx context.Context, app *App, userId string) (_ *Sess, err error) {
	token, err := newToken(app, userId)
	if err != nil {
		return
	}
	s := &Sess{
		Token:  token,
		AppId:  app.Id,
		UserId: userId,
	}
	b, _ := msgpack.Marshal(&s)
	if err = rdbAuth.Set(
		ctx, userId, util.BytesToString(b), dbSessTTL).Err(); err != nil {
	}
	return s, nil
}

func CheckToken(ctx context.Context, token string) (_ *Sess, err error) {
	app, userId, err := decToken(token)
	if err != nil {
		log.Warnf("failed to decode token: %v", err)
		return nil, ErrInvalidToken
	}
	b, err := rdbAuth.Get(ctx, userId).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrInvalidToken
		} else {
			log.Warnf("failed to get token from redis: %v", err)
			return nil, ErrDatabaseUnavailable
		}
	}
	s := &Sess{
		AppId:  app.Id,
		UserId: userId,
		App:    app,
	}
	if err = msgpack.Unmarshal(b, &s); err != nil {
		log.Warnf("failed to decode SessData: %v", err)
		return nil, ErrMalformedSessData
	}
	if s.Token != token {
		return nil, ErrInvalidToken
	}
	return s, nil
}

func CheckNonce(ctx context.Context, appId, nonce string) (err error) {
	if len(nonce) > 32 {
		return ErrInvalidNonce
	}
	key := fmt.Sprintf("%s$%s", appId, nonce)
	ok, err := rdbNonce.SetNX(ctx, key, "", cfg.Nonce.timeout).Result()
	if err != nil {
		return ErrDatabaseUnavailable
	}
	if !ok {
		return ErrInvalidNonce
	}
	return
}
