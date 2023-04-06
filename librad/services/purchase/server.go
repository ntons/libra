package purchase

import (
	"context"
	"strings"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/libra/librad/common/rpc"
	"github.com/ntons/libra/librad/db"
)

type orderServer struct {
	v1pb.UnimplementedOrderServer
}

func newOrderServer() *orderServer {
	return &orderServer{}
}

func (*orderServer) Get(
	ctx context.Context, req *v1pb.OrderGetRequest) (
	_ *v1pb.OrderGetResponse, err error) {
	return
}

func (srv *orderServer) Create(
	ctx context.Context, req *v1pb.OrderCreateRequest) (
	_ *v1pb.OrderCreateResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, rpc.InvalidAppSecretError
	}

	if err = srv.checkStringArguments(
		"order_id", req.OrderId,
		"price", req.Price,
		"channel_id", req.ChannelId,
		"product_id", req.ProductId,
		"product_quantity", req.ProductQuantity,
		"user_id", req.UserId,
		"role_id", req.RoleId,
	); err != nil {
		return
	}

	if err = db.CreateOrder(
		ctx, trusted.AppId, &v1pb.PurchaseData{
			OrderId:         req.OrderId,
			Currency:        req.Currency,
			Price:           req.Price,
			ChannelId:       req.ChannelId,
			ProductId:       req.ProductId,
			ProductName:     req.ProductName,
			ProductQuantity: req.ProductQuantity,
			UserId:          req.UserId,
			RoleId:          req.RoleId,
		},
	); err != nil {
		return
	}
	return
}

func (srv *orderServer) Cancel(
	ctx context.Context, req *v1pb.OrderCancelRequest) (
	_ *v1pb.OrderCancelResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, rpc.InvalidAppSecretError
	}

	if err = srv.checkStringArguments(
		"order_id", req.OrderId,
	); err != nil {
		return
	}

	if err = db.CancelOrder(
		ctx, trusted.AppId, req.OrderId, req.Reason,
	); err != nil {
		return
	}

	return
}

func (srv *orderServer) Pay(
	ctx context.Context, req *v1pb.OrderPayRequest) (
	_ *v1pb.OrderPayResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, rpc.InvalidAppSecretError
	}

	if err = srv.checkStringArguments(
		"order_id", req.OrderId,
		"transaction_id", req.TransactionId,
		"receipt", req.Receipt,
	); err != nil {
		return
	}

	if err = db.PayOrder(
		ctx, trusted.AppId, req.OrderId, req.TransactionId, req.Receipt,
	); err != nil {
		return
	}

	return
}

func (srv *orderServer) Fulfill(
	ctx context.Context, req *v1pb.OrderFulfillRequest) (
	_ *v1pb.OrderFulfillResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, rpc.InvalidAppSecretError
	}

	if err = srv.checkStringArguments(
		"order_id", req.OrderId,
	); err != nil {
		return
	}

	if err = db.FulfillOrder(
		ctx, trusted.AppId, req.OrderId,
	); err != nil {
		return
	}

	return
}

func (orderServer) checkStringArguments(keyVals ...string) error {
	for i := 1; i < len(keyVals); i += 2 {
		if strings.TrimSpace(keyVals[i]) == "" {
			return rpc.NewInvalidArgumentError("invalid argument %s", keyVals[i-1])
		}
	}
	return nil
}
