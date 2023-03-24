package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Gift struct {
	// ID
	Id string `bson:"_id,omitempty"`
	// 更新时间
	UpdateAt time.Time `bson:"update_at,omitempty"`
	// 过期时间
	ExpireAt time.Time `bson:"expire_at,omitempty"`
	// 有效载荷
	Payload []byte `bson:"payload,omitempty"`
}

type GiftCode struct {
	// 礼包码
	Code string `bson:"_id,omitempty"`
	// 礼包ID
	GiftId string `bson:"gift_id,omitempty"`
	// 已兑换
	Redeemed bool `bson:"redeemed,omitempty"`
}

func getGiftCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	dbGiftCollectionMu.Lock()
	defer dbGiftCollectionMu.Unlock()

	collection, ok := dbGiftCollection[appId]
	if !ok {
		const tblName = "libra.gifts"
		dbName := getAppDBName(appId)
		collection = mdb.Database(dbName).Collection(tblName)
		dbGiftCollection[appId] = collection
	}

	return collection, nil
}

func getGiftCodeCollection(
	ctx context.Context, appId string) (*mongo.Collection, error) {
	dbGiftCodeCollectionMu.Lock()
	defer dbGiftCodeCollectionMu.Unlock()

	collection, ok := dbGiftCodeCollection[appId]
	if !ok {
		const tblName = "libra.giftcodes"
		dbName := getAppDBName(appId)
		collection = mdb.Database(dbName).Collection(tblName)
		if _, err := collection.Indexes().CreateOne(
			ctx,
			mongo.IndexModel{
				Keys: bson.D{{Key: "gift_id", Value: 1}},
			},
		); err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
		dbGiftCodeCollection[appId] = collection
	}

	return collection, nil
}

func CreateGift(ctx context.Context, appId string, gift *Gift, codes []string) (err error) {
	giftCollection, err := getGiftCollection(ctx, appId)
	if err != nil {
		return
	}

	now := time.Now()

	// 已过期的话就不需要入库了
	if !gift.ExpireAt.IsZero() && gift.ExpireAt.Before(now) {
		return
	}

	gift.UpdateAt = now

	if _, err = giftCollection.InsertOne(ctx, gift); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err = newAlreadyExistsError("gift already exists")
		}
		return
	}

	if len(codes) > 0 {
		if err = AddCodesToGift(ctx, appId, gift.Id, codes); err != nil {
			return
		}
	}

	return
}

func RevokeGift(ctx context.Context, appId, giftId string) (err error) {
	giftCollection, err := getGiftCollection(ctx, appId)
	if err != nil {
		return
	}

	codeCollection, err := getGiftCodeCollection(ctx, appId)
	if err != nil {
		return
	}

	return doRevokeGift(ctx, giftId, giftCollection, codeCollection)
}

func doRevokeGift(
	ctx context.Context, giftId string,
	giftCollection, codeCollection *mongo.Collection) (err error) {

	if _, err = giftCollection.DeleteOne(ctx, &Gift{Id: giftId}); err != nil {
		if err != mongo.ErrNoDocuments {
			return
		}
		err = nil
	}

	if _, err = codeCollection.DeleteMany(ctx, &GiftCode{GiftId: giftId}); err != nil {
		if err != mongo.ErrNoDocuments {
			return
		}
		err = nil
	}

	return

}

func UpdateGift(ctx context.Context, appId string, gift *Gift) (err error) {
	giftCollection, err := getGiftCollection(ctx, appId)
	if err != nil {
		return
	}

	gift.UpdateAt = time.Now()

	if _, err = giftCollection.ReplaceOne(ctx, &Gift{Id: gift.Id}, gift); err != nil {
		if err == mongo.ErrNoDocuments {
			err = newNotFoundError("gift not exists")
		}
		return
	}

	return
}

func GetAllGifts(ctx context.Context, appId string) (gifts []*Gift, err error) {
	giftCollection, err := getGiftCollection(ctx, appId)
	if err != nil {
		return
	}

	cursor, err := giftCollection.Find(ctx, bson.D{})
	if err != nil {
		return
	}

	if err = cursor.All(ctx, &gifts); err != nil {
		return
	}

	return
}

func AddCodesToGift(ctx context.Context, appId, giftId string, giftCodes []string) (err error) {
	codeCollection, err := getGiftCodeCollection(ctx, appId)
	if err != nil {
		return
	}

	docs := make([]interface{}, 0, len(giftCodes))
	for _, code := range giftCodes {
		docs = append(docs, &GiftCode{
			Code:   code,
			GiftId: giftId,
		})
	}

	if _, err = codeCollection.InsertMany(
		ctx, docs, options.InsertMany().SetOrdered(false)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err = newAlreadyExistsError("some codes already exists")
		}
		return
	}

	return
}

func DelCodesFromGift(ctx context.Context, appId string, codes []string) (err error) {
	codeCollection, err := getGiftCodeCollection(ctx, appId)
	if err != nil {
		return
	}

	if _, err = codeCollection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": codes}}); err != nil {
		return
	}

	return
}

func GetGiftById(ctx context.Context, appId, giftId string) (_ *Gift, err error) {
	giftCollection, err := getGiftCollection(ctx, appId)
	if err != nil {
		return
	}
	var gift Gift
	if err = giftCollection.FindOne(ctx, &Gift{Id: giftId}).Decode(&gift); err != nil {
		return
	}
	return &gift, nil
}

func GetGiftByCode(ctx context.Context, appId, code string) (_ *Gift, err error) {
	codeCollection, err := getGiftCodeCollection(ctx, appId)
	if err != nil {
		return
	}
	var giftCode GiftCode
	if err = codeCollection.FindOne(ctx, &GiftCode{Code: code}).Decode(&giftCode); err != nil {
		return
	}
	return GetGiftById(ctx, appId, giftCode.GiftId)
}

func GetCodesByGiftId(ctx context.Context, appId, giftId string) (_ []*GiftCode, err error) {
	codeCollection, err := getGiftCodeCollection(ctx, appId)
	if err != nil {
		return
	}
	var giftCodes []*GiftCode
	cursor, err := codeCollection.Find(ctx, &GiftCode{GiftId: giftId})
	if err != nil {
		return
	}
	if err = cursor.All(ctx, &giftCodes); err != nil {
		return
	}
	return giftCodes, nil
}

func VerifyGiftCode(ctx context.Context, appId, code string) (_ *Gift, err error) {
	return doVerifyOrRedeemGiftCode(ctx, appId, code, false)
}

func RedeemGiftCode(ctx context.Context, appId, code string) (_ *Gift, err error) {
	return doVerifyOrRedeemGiftCode(ctx, appId, code, true)
}

func doVerifyOrRedeemGiftCode(ctx context.Context, appId, code string, redeem bool) (_ *Gift, err error) {
	giftCollection, err := getGiftCollection(ctx, appId)
	if err != nil {
		return
	}

	codeCollection, err := getGiftCodeCollection(ctx, appId)
	if err != nil {
		return
	}

	var res *mongo.SingleResult
	if redeem {
		res = codeCollection.FindOneAndUpdate(
			ctx,
			&GiftCode{Code: code},
			bson.M{"$set": &GiftCode{Redeemed: true}},
			options.FindOneAndUpdate().SetReturnDocument(options.Before),
		)
	} else {
		res = codeCollection.FindOne(
			ctx,
			&GiftCode{Code: code},
		)
	}

	var giftCode GiftCode
	if err = res.Decode(&giftCode); err != nil {
		if err == mongo.ErrNoDocuments {
			err = newNotFoundError("invalid gift code")
		}
		return
	}
	if giftCode.Redeemed {
		return nil, newCustomError(101, "redeemed gift code")
	}

	var gift Gift
	if err = giftCollection.FindOne(
		ctx, &Gift{Id: giftCode.GiftId}).Decode(&gift); err != nil {
		if err == mongo.ErrNoDocuments {
			err = newNotFoundError("invalid gift code")
		}
		return
	}
	if !gift.ExpireAt.IsZero() && gift.ExpireAt.Before(time.Now()) {
		return nil, newCustomError(102, "expired gift code")
	}

	return &gift, nil
}
