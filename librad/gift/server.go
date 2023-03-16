package gift

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/libra/librad/db"
	"github.com/ntons/log-go"
	"github.com/ntons/redis"
	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() { modularity.Register(&giftServer{}) }

type giftServer struct {
	modularity.Skeleton
	v1pb.UnimplementedGiftServer
	cli redis.Client
}

func (giftServer) Name() string { return "gift" }

func (srv *giftServer) Initialize(jb json.RawMessage) (err error) {
	if jb == nil {
		return
	} else if err = json.Unmarshal(jb, &cfg); err != nil {
		return
	} else if cfg.Redis == "" {
		return fmt.Errorf("require redis configuration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if srv.cli, err = redis.Dial(ctx, cfg.Redis); err != nil {
		log.Warnf("failed to connect to redis: %v", err)
		return fmt.Errorf("failed to connect to redis")
	}

	server.RegisterService(&v1pb.Gift_ServiceDesc, srv)
	return
}

func (srv *giftServer) Create(
	ctx context.Context, req *v1pb.GiftCreateRequest) (
	_ *v1pb.GiftCreateResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	gift, err := giftFromData(req.Data)
	if err != nil {
		return
	}

	if err = db.CreateGift(ctx, trusted.AppId, gift, req.Codes); err != nil {
		return
	}

	return &v1pb.GiftCreateResponse{}, nil
}

func (srv *giftServer) Revoke(
	ctx context.Context, req *v1pb.GiftRevokeRequest) (
	_ *v1pb.GiftRevokeResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	if err = db.RevokeGift(ctx, trusted.AppId, req.Id); err != nil {
		return
	}

	return &v1pb.GiftRevokeResponse{}, nil
}

func (srv *giftServer) Update(
	ctx context.Context, req *v1pb.GiftUpdateRequest) (
	_ *v1pb.GiftUpdateResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	if req.Data != nil {
		req.Data.Id = req.Id
		var gift *db.Gift
		if gift, err = giftFromData(req.Data); err != nil {
			return
		}
		if err = db.UpdateGift(ctx, trusted.AppId, gift); err != nil {
			return
		}
	}

	if len(req.CodesToDel) > 0 {
		if err = db.DelCodesFromGift(
			ctx, trusted.AppId, req.CodesToDel); err != nil {
			return
		}
	}

	if len(req.CodesToAdd) > 0 {
		if err = db.AddCodesToGift(
			ctx, trusted.AppId, req.Id, req.CodesToAdd); err != nil {
			return
		}
	}

	return &v1pb.GiftUpdateResponse{}, nil
}

func (srv *giftServer) List(
	ctx context.Context, req *v1pb.GiftListRequest) (
	_ *v1pb.GiftListResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	gifts, err := db.ListGifts(ctx, trusted.AppId)
	if err != nil {
		return
	}

	resp := &v1pb.GiftListResponse{}
	for _, gift := range gifts {
		if data, err := giftToData(gift); err == nil {
			resp.Data = append(resp.Data, data)
		}
	}

	return resp, nil
}

func (srv *giftServer) Verify(
	ctx context.Context, req *v1pb.GiftVerifyRequest) (
	_ *v1pb.GiftVerifyResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	gift, err := db.VerifyGiftCode(ctx, trusted.AppId, req.Code)
	if err != nil {
		return
	}

	data, err := giftToData(gift)
	if err != nil {
		return
	}

	return &v1pb.GiftVerifyResponse{Data: data}, nil
}

func (srv *giftServer) Redeem(
	ctx context.Context, req *v1pb.GiftRedeemRequest) (
	_ *v1pb.GiftRedeemResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	gift, err := db.RedeemGiftCode(ctx, trusted.AppId, req.Code)
	if err != nil {
		return
	}

	data, err := giftToData(gift)
	if err != nil {
		return
	}

	return &v1pb.GiftRedeemResponse{Data: data}, nil
}
