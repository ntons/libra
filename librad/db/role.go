package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ntons/log-go"
	"github.com/ntons/redis"
	"github.com/vmihailenco/msgpack/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Role struct {
	Id string `bson:"_id"`
	// 角色序号，主要有一下几个用途
	// 1. 创建用户发生重试时保证只有唯一一个角色被成功创建
	// 2. 用来确定该用户的角色分类，比如分区分服
	Index uint32 `bson:"index,omitempty"`
	// 所属用户ID
	UserId string `bson:"user_id,omitempty"`
	// 创建时间
	CreateAt time.Time `bson:"create_at,omitempty"`
	// 上次登录时间
	SignInAt time.Time `bson:"sign_in_at,omitempty"`
	// 元数据
	Metadata map[string]string `bson:"metadata,omitempty"`
}

// get role collection of app
func getRoleCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	dbRoleCollectionMu.Lock()
	defer dbRoleCollectionMu.Unlock()

	if collection, ok := dbRoleCollection[appId]; ok {
		return collection, nil
	}

	const tblName = "libra.roles"
	dbName := getAppDBName(appId)

	collection := mdb.Database(dbName).Collection(tblName)
	if _, err := collection.Indexes().CreateOne(
		ctx,
		mongo.IndexModel{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "index", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}
	dbRoleCollection[appId] = collection
	return collection, nil
}

func GetRole(
	ctx context.Context, appId /*, userId*/, roleId string) (_ *Role, err error) {
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &Role{}
	if err = collection.FindOne(
		ctx,
		// role必须属于user才算找到
		bson.M{"_id": roleId /*, "user_id": userId*/},
	).Decode(role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ErrRoleNotFound
		} else {
			err = ErrDatabaseUnavailable
		}
		return
	}
	return role, nil
}

func GetRoles(
	ctx context.Context, appId string, roleIds []string) (
	_ []*Role, err error) {
	if len(roleIds) == 0 {
		return
	}
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	cursor, err := collection.Find(
		ctx,
		bson.M{"_id": bson.M{"$in": roleIds}},
	)
	if err != nil {
		return
	}
	var roles []*Role
	if err = cursor.All(ctx, &roles); err != nil {
		return
	}
	return roles, nil
}

func GetRolesByUserId(
	ctx context.Context, appId string, userIds []string) (
	_ []*Role, err error) {
	if len(userIds) == 0 {
		return
	}
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	cursor, err := collection.Find(
		ctx,
		bson.M{"user_id": bson.M{"$in": userIds}},
	)
	if err != nil {
		return
	}
	var roles []*Role
	if err = cursor.All(ctx, &roles); err != nil {
		return
	}
	return roles, nil
}

func ListRoles(
	ctx context.Context, appId, userId string) (_ []*Role, err error) {
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	cur, err := collection.Find(ctx, bson.M{"user_id": userId})
	if err != nil {
		return
	}
	var roles []*Role
	if err = cur.All(ctx, &roles); err != nil {
		return
	}
	return roles, nil
}

func CreateRole(
	ctx context.Context, appId, userId string, index uint32) (
	_ *Role, err error) {
	app := FindAppById(appId)
	if app == nil {
		return nil, ErrInvalidAppId
	}
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &Role{
		Id:       newRoleId(app.Key),
		UserId:   userId,
		Index:    index,
		CreateAt: time.Now(),
	}
	if _, err = collection.InsertOne(ctx, role); err != nil {
		return
	}
	return role, nil
}

func SignInRole(
	ctx context.Context, appId /*,userId*/, roleId string) (err error) {
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	var role Role
	if err = collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": roleId /*, "user_id": userId*/},
		bson.M{"$set": bson.M{"sign_in_at": time.Now()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = ErrRoleNotFound
		} else {
			err = ErrDatabaseUnavailable
		}
		return
	}
	// update sess data
	b, _ := msgpack.Marshal(&SessData{
		RoleId:    roleId,
		RoleIndex: role.Index,
	})
	if err = luaUpdateSessData.Run(
		ctx, rdbAuth, []string{role.UserId}, b).Err(); err != nil {
		if err == redis.Nil {
			return ErrInvalidToken
		} else {
			log.Warnf("failed to update session: %v", err)
			return ErrDatabaseUnavailable
		}
	}
	return
}

func SetRoleMetadata(
	ctx context.Context, appId /*, userId*/, roleId string,
	md map[string]string) (err error) {
	collection, err := getRoleCollection(ctx, appId)
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
			bson.M{"_id": roleId /*, "user_id": userId*/},
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
