package db

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpdateAcctDetail(
	ctx context.Context, appId, acctId, detail string) (err error) {
	collection, err := getAcctCollection(ctx, appId)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	if _, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": acctId},
		bson.M{"$set": bson.M{"detail": parseAcctDetail(detail)}},
		options.Update().SetUpsert(true),
	); err != nil {
		return fmt.Errorf("failed to update acct detail: %w", err)
	}
	return
}

func getAcctCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	dbAcctCollectionMu.Lock()
	defer dbAcctCollectionMu.Unlock()

	if collection, ok := dbAcctCollection[appId]; ok {
		return collection, nil
	}

	const tblName = "libra.accts"
	dbName := getAppDBName(appId)

	collection := mdb.Database(dbName).Collection(tblName)
	dbAcctCollection[appId] = collection
	return collection, nil
}

func parseAcctDetail(s string) interface{} {
	s = strings.TrimSpace(s)
	if v := assumeJsonString(s); v != nil {
		return v
	}
	if v := assumeQueryString(s); v != nil {
		return v
	}
	return s // 默认返回原始字符串
}
func assumeJsonString(s string) interface{} {
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		var structured map[string]interface{}
		if json.Unmarshal([]byte(s), &structured) == nil {
			return structured
		}
	}
	return nil
}
func assumeQueryString(s string) interface{} {
	if values, err := url.ParseQuery(s); err == nil {
		return values
	}
	return nil
}
