package db

import (
	"context"
	"sync"
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/libra/librad/common/rpc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	dbOrderCollectionMu sync.Mutex
	dbOrderCollection   = make(map[string]*mongo.Collection)
)

func getOrderCollection(ctx context.Context, appId string) (_ *mongo.Collection, err error) {
	dbOrderCollectionMu.Lock()
	defer dbOrderCollectionMu.Unlock()

	if collection, ok := dbOrderCollection[appId]; ok {
		return collection, nil
	}

	const tblName = "libra.orders"
	dbName := getAppDBName(appId)

	collection := mdb.Database(dbName).Collection(tblName)
	if _, err = collection.Indexes().CreateMany(
		ctx,
		[]mongo.IndexModel{
			{
				Keys: bson.D{
					{Key: "order_id", Value: 1},
				},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{
					{Key: "channel_id", Value: 1},
					{Key: "transaction_id", Value: 1},
				},
				Options: options.Index().SetUnique(true).SetSparse(true),
			},
		},
	); err != nil {
		return nil, rpc.NewUnavailableError("failed to create index for order collection")
	}

	dbOrderCollection[appId] = collection
	return collection, nil
}

// 查询订单
func GetOrder(ctx context.Context, appId, orderId string) (_ *v1pb.PurchaseData, err error) {
	collection, err := getOrderCollection(ctx, appId)
	if err != nil {
		return
	}

	filter, _ := msgToDoc(&v1pb.PurchaseData{OrderId: orderId})

	var doc map[string]interface{}
	if err = collection.FindOne(ctx, filter).Decode(&doc); err != nil {
		return
	}

	var data v1pb.PurchaseData
	if err = docToMsg(doc, &data); err != nil {
		return
	}

	return &data, nil
}

// 创建订单
func CreateOrder(ctx context.Context, appId string, data *v1pb.PurchaseData) (err error) {
	collection, err := getOrderCollection(ctx, appId)
	if err != nil {
		return
	}

	var (
		doc interface{}
	)

	data.CreateAt = time.Now().UnixNano() / 1e6
	data.State = v1pb.PurchaseState_PurchaseStatePending
	data.AppId = appId

	if doc, err = msgToDoc(data); err != nil {
		return
	}
	_, err = collection.InsertOne(ctx, doc)
	if err != nil {
		if !mongo.IsDuplicateKeyError(err) {
			return // 重复创建不算错误
		}
		err = nil
	}
	return
}

// 设置订单已取消
func CancelOrder(ctx context.Context, appId, orderId, reason string) (err error) {
	collection, err := getOrderCollection(ctx, appId)
	if err != nil {
		return
	}

	var filter, update map[string]interface{}

	if filter, err = msgToDoc(&v1pb.PurchaseData{
		OrderId: orderId,
	}); err != nil {
		return
	}
	filter["state"] = bson.M{"$in": bson.A{
		int32(v1pb.PurchaseState_PurchaseStatePending),
		int32(v1pb.PurchaseState_PurchaseStateCanceled),
	}}

	if update, err = msgToDoc(&v1pb.PurchaseData{
		State:        v1pb.PurchaseState_PurchaseStateCanceled,
		CancelAt:     time.Now().UnixNano() / 1e6,
		CancelReason: reason,
	}); err != nil {
		return
	}

	if _, err = collection.UpdateOne(ctx, filter, update); err != nil {
		if err == mongo.ErrNoDocuments {
			return rpc.NewNotFoundError("Order not found or in incorrect state")
		}
		return
	}

	return
}

// 设置订单完成，必须提供原始凭证ID以备查验
func PayOrder(ctx context.Context, appId, orderId, transactionId, receipt string) (err error) {
	collection, err := getOrderCollection(ctx, appId)
	if err != nil {
		return
	}

	var filter, update map[string]interface{}

	if filter, err = msgToDoc(&v1pb.PurchaseData{
		OrderId: orderId,
	}); err != nil {
		return
	}
	filter["state"] = bson.M{"$in": bson.A{
		int32(v1pb.PurchaseState_PurchaseStatePending),
		int32(v1pb.PurchaseState_PurchaseStatePaid),
	}}

	if update, err = msgToDoc(&v1pb.PurchaseData{
		State:         v1pb.PurchaseState_PurchaseStatePaid,
		PayAt:         time.Now().UnixNano() / 1e6,
		TransactionId: transactionId,
		Receipt:       receipt,
	}); err != nil {
		return
	}

	if _, err = collection.UpdateOne(ctx, filter, update); err != nil {
		if err == mongo.ErrNoDocuments {
			return rpc.NewNotFoundError("Order not found or in incorrect state")
		}
		return
	}

	return
}

func FulfillOrder(ctx context.Context, appId, orderId string) (err error) {
	collection, err := getOrderCollection(ctx, appId)
	if err != nil {
		return
	}

	var filter, update map[string]interface{}

	if filter, err = msgToDoc(&v1pb.PurchaseData{
		OrderId: orderId,
	}); err != nil {
		return
	}
	filter["state"] = bson.M{"$in": bson.A{
		int32(v1pb.PurchaseState_PurchaseStatePaid),
		int32(v1pb.PurchaseState_PurchaseStateFulfilled),
	}}

	if update, err = msgToDoc(&v1pb.PurchaseData{
		State:     v1pb.PurchaseState_PurchaseStateFulfilled,
		FulfillAt: time.Now().UnixNano() / 1e6,
	}); err != nil {
		return
	}

	if _, err = collection.UpdateOne(ctx, filter, update); err != nil {
		if err == mongo.ErrNoDocuments {
			return rpc.NewNotFoundError("Order not found or in incorrect state")
		}
		return rpc.NewUnavailableError("Database is unavailable")
	}

	return
}
