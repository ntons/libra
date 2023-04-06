package gift

import (
	"context"
	"encoding/json"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/libra/librad/db"
	"github.com/ntons/log-go"
	"github.com/onemoreteam/httpframework/modularity"
	"github.com/onemoreteam/httpframework/modularity/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() { modularity.Register(&giftServer{}) }

type giftServer struct {
	modularity.Skeleton
	v1pb.UnimplementedGiftServer
}

func (giftServer) Name() string { return "gift" }

func (srv *giftServer) Initialize(jb json.RawMessage) (err error) {
	if jb == nil {
		return
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

func (srv *giftServer) Search(
	ctx context.Context, req *v1pb.GiftSearchRequest) (
	_ *v1pb.GiftSearchResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	var (
		appId     = trusted.AppId
		gifts     []*db.Gift
		withCodes = false
	)
	if len(req.Id) > 0 {
		var gift *db.Gift
		if gift, err = db.GetGiftById(ctx, appId, req.Id); err != nil {
			return
		}
		gifts, withCodes = append(gifts, gift), true
	} else if len(req.Code) > 0 {
		var gift *db.Gift
		if gift, err = db.GetGiftByCode(ctx, appId, req.Code); err != nil {
			return
		}
		gifts = append(gifts, gift)
	} else {
		if gifts, err = db.GetAllGifts(ctx, appId); err != nil {
			return
		}
	}

	resp := &v1pb.GiftSearchResponse{}
	for _, gift := range gifts {
		e := &v1pb.GiftSearchResponse_Entry{}
		resp.List = append(resp.List, e)
		if e.Data, err = giftToData(gift); err != nil {
			return
		}
		if !withCodes {
			continue
		}
		var giftCodes []*db.GiftCode
		if giftCodes, err = db.GetCodesByGiftId(ctx, appId, gift.Id); err != nil {
			return
		}
		for _, giftCode := range giftCodes {
			e.Codes = append(e.Codes, &v1pb.GiftCodeData{
				Code:     giftCode.Code,
				Redeemed: giftCode.Redeemed,
			})
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

	log.Infow("redeem gift", "code", req.Code, "gift", gift.Id)

	return &v1pb.GiftRedeemResponse{Data: data}, nil
}
