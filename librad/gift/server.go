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

func (srv *giftServer) List(
	ctx context.Context, req *v1pb.GiftListRequest) (
	_ *v1pb.GiftListResponse, err error) {

	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	var giftAndCodes = make(map[*db.Gift][]string)

	if req.Id == "" {
		var gifts []*db.Gift
		if gifts, err = db.GetAllGifts(ctx, trusted.AppId); err != nil {
			return
		}
		for _, gift := range gifts {
			giftAndCodes[gift] = nil
		}
	} else {
		var gift *db.Gift
		var codes []string
		if gift, codes, err = db.GetGiftAndCodes(
			ctx, trusted.AppId, req.Id); err != nil {
			return
		}
		giftAndCodes[gift] = codes
	}

	resp := &v1pb.GiftListResponse{}
	for gift, codes := range giftAndCodes {
		if data, err := giftToData(gift); err == nil {
			resp.List = append(resp.List, &v1pb.GiftListResponse_Entry{Data: data, Codes: codes})
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
