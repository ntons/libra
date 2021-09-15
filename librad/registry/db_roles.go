package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/ntons/log-go"
	"github.com/vmihailenco/msgpack/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ntons/libra/librad/internal/redis"
)

type dbRole struct {
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
	const collectionName = "roles"
	if collection, ok := dbRoleCollection[appId]; ok {
		return collection, nil
	}
	collection := mdb.Database(getAppDBName(appId)).Collection(collectionName)
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

func getRole(
	ctx context.Context, appId, userId, roleId string) (_ *dbRole, err error) {
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &dbRole{}
	if err = collection.FindOne(
		ctx,
		// role必须属于user才算找到
		bson.M{"_id": roleId, "user_id": userId},
	).Decode(role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errRoleNotFound
		} else {
			err = errDatabaseUnavailable
		}
		return
	}
	return role, nil
}

func getRoles(
	ctx context.Context, appId string, roleIds []string) (
	_ []*dbRole, err error) {
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
	var roles []*dbRole
	if err = cursor.All(ctx, &roles); err != nil {
		return
	}
	return roles, nil
}

func getRolesByUserId(
	ctx context.Context, appId string, userIds []string) (
	_ []*dbRole, err error) {
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
	var roles []*dbRole
	if err = cursor.All(ctx, &roles); err != nil {
		return
	}
	return roles, nil
}

func listRoles(
	ctx context.Context, appId, userId string) (_ []*dbRole, err error) {
	collection, err := getRoleCollection(ctx, appId)
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

func createRole(
	ctx context.Context, appId, userId string, index uint32) (
	_ *dbRole, err error) {
	app := findAppById(appId)
	if app == nil {
		return nil, errInvalidAppId
	}
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	role := &dbRole{
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

func signInRole(
	ctx context.Context, appId, userId, roleId string) (err error) {
	collection, err := getRoleCollection(ctx, appId)
	if err != nil {
		return
	}
	var role dbRole
	if err = collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": roleId, "user_id": userId},
		bson.M{"$set": bson.M{"sign_in_at": time.Now()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&role); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errRoleNotFound
		} else {
			err = errDatabaseUnavailable
		}
		return
	}
	// update sess data
	b, _ := msgpack.Marshal(&dbSessData{
		RoleId:    roleId,
		RoleIndex: role.Index,
	})
	if err = luaUpdateSessData.Run(
		ctx, rdbAuth, []string{userId}, b).Err(); err != nil {
		if err == redis.Nil {
			return errInvalidToken
		} else {
			log.Warnf("failed to update session: %v", err)
			return errDatabaseUnavailable
		}
	}
	return
}

func setRoleMetadata(
	ctx context.Context, appId, userId, roleId string,
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
			bson.M{"_id": roleId, "user_id": userId},
			update,
		); err != nil {
			log.Warnf("failed to access user: %v", err)
			return errDatabaseUnavailable
		} else if r.MatchedCount == 0 {
			return errUserNotFound
		}
	}
	return
}
