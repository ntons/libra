package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 通用封禁

type BlockData struct {
	Key string `bson:"_id"`
	// 封禁时间
	BanAt time.Time `bson:"ban_at,omitempty"`
	// 封禁时间
	BanTo time.Time `bson:"ban_to,omitempty"`
	// 封禁原因
	BanFor string `bson:"ban_for,omitempty"`
}

func Block(ctx context.Context, appId string, keys []string, banTo time.Time, banFor string) (err error) {
	coll, err := getBlockCollection(ctx, appId)
	if err != nil {
		return
	}
	if _, err = coll.UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": keys}},
		bson.M{"$set": bson.M{
			"ban_at":  time.Now(),
			"ban_to":  banTo,
			"ban_for": banFor,
		}},
		options.Update().SetUpsert(true),
	); err != nil {
		return
	}
	return
}

func Allow(ctx context.Context, appId string, keys []string) (err error) {
	coll, err := getBlockCollection(ctx, appId)
	if err != nil {
		return
	}
	if _, err = coll.DeleteMany(
		ctx,
		bson.M{"_id": bson.M{"$in": keys}},
	); err != nil {
		return
	}
	return
}

func IsBlocked(ctx context.Context, appId string, keys []string) (_ *BlockData, err error) {
	coll, err := getBlockCollection(ctx, appId)
	if err != nil {
		return
	}
	cur, err := coll.Find(
		ctx,
		bson.M{"_id": bson.M{"$in": keys}},
	)
	if err != nil {
		return
	}

	var blocks []*BlockData
	if err = cur.All(ctx, &blocks); err != nil {
		return
	}

	var (
		now         = time.Now()
		latestBlock *BlockData
		latestBanAt time.Time
	)
	for _, block := range blocks {
		if block.BanTo.After(now) &&
			(latestBlock == nil || block.BanAt.After(latestBanAt)) {
			latestBlock = block
			latestBanAt = block.BanAt
		}
	}

	return latestBlock, nil
}

func getBlockCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	dbBlockCollectionMu.Lock()
	defer dbBlockCollectionMu.Unlock()

	if collection, ok := dbBlockCollection[appId]; ok {
		return collection, nil
	}

	const tblName = "libra.blocks"
	dbName := getAppDBName(appId)

	collection := mdb.Database(dbName).Collection(tblName)
	dbBlockCollection[appId] = collection
	return collection, nil
}
