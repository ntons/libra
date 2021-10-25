package registry

import (
	"context"
	"strings"
	"time"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	log "github.com/ntons/log-go"
	"github.com/ntons/tongo/httputil"
	"github.com/ntons/tongo/sign"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/registry/db"
)

func fromDbUser(x *db.User) *v1pb.UserData {
	r := &v1pb.UserData{
		Id:       x.Id,
		AcctIds:  x.AcctIds,
		CreateAt: x.CreateAt.Unix(),
		LoginAt:  x.LoginAt.Unix(),
		LoginIp:  x.LoginIp,
		Metadata: x.Metadata,
	}
	if x.BanTo.After(time.Now()) {
		r.BanTo = x.BanTo.Unix()
		r.BanFor = x.BanFor
	}
	return r
}
func fromDbUsers(x []*db.User) []*v1pb.UserData {
	r := make([]*v1pb.UserData, 0, len(x))
	for _, e := range x {
		r = append(r, fromDbUser(e))
	}
	return r
}

type userServer struct {
	v1pb.UnimplementedUserServer
}

func newUserServer() *userServer {
	return &userServer{}
}

type xGenericLoginState struct {
	AcctIds    []string
	Properties *v1pb.LoginStateProperties
	Features   []*v1pb.SessionFeature
}

func (srv *userServer) CheckLoginState(
	ctx context.Context, appId string, anyState *anypb.Any) (
	app *db.App, state *xGenericLoginState, err error) {
	if app = db.FindAppById(appId); app == nil {
		log.Warnf("invalid app id: %v", appId)
		return nil, nil, db.ErrInvalidAppId
	}

	x, err := anypb.UnmarshalNew(anyState, proto.UnmarshalOptions{})
	if err != nil {
		return nil, nil, errInvalidState
	}

	switch x := x.(type) {
	case *v1pb.UniformLoginState:
		if state, err = srv.CheckUniformLoginState(ctx, app, x); err != nil {
			return
		}
	default:
		return nil, nil, errInvalidState
	}
	return
}

func (srv *userServer) CheckUniformLoginState(
	ctx context.Context, app *db.App, state *v1pb.UniformLoginState) (
	_ *xGenericLoginState, err error) {
	if state == nil {
		return nil, errInvalidState
	}
	if err = db.CheckNonce(ctx, app.Id, state.Nonce); err != nil {
		return
	}
	// ts-30 签名有效期只有30秒钟
	// ts+5  是为了容忍一定的系统时间误差
	ts := time.Now().Unix()
	if state.Timestamp < ts-30 || state.Timestamp > ts+5 {
		return nil, errInvalidTimestamp
	}
	signature := state.Signature
	state.Signature = ""
	expected := sign.ProtoHMACWithSHA1(state, app.Secret)
	if !strings.EqualFold(signature, expected) {
		log.Warnf("signature mismatch: %s, %s, %s, %s",
			signature, expected, app.Secret, state)
		return nil, errInvalidSignature
	}
	return &xGenericLoginState{
		AcctIds:    state.AcctIds,
		Properties: state.Properties,
		Features:   state.Features,
	}, nil
}

func (srv *userServer) Login(
	ctx context.Context, req *v1pb.UserLoginRequest) (
	_ *v1pb.UserLoginResponse, err error) {
	app, state, err := srv.CheckLoginState(ctx, req.AppId, req.State)
	if err != nil {
		return
	}

	user, sess, err := db.LoginUser(
		ctx, app,
		httputil.GetRemoteIpFromContext(ctx),
		state.AcctIds,
		req.CreateIfNotFound)
	if err != nil {
		log.Warnf("failed to login user: %v", err)
		return
	}

	log.Infow("user login", "user_id", user.Id, "acct_ids", state.AcctIds)

	//grpc.SetHeader(ctx, metadata.Pairs(L.XLibraToken, sess.Token))
	grpc.SetTrailer(ctx, metadata.Pairs(L.XLibraToken, sess.Token))

	return &v1pb.UserLoginResponse{User: fromDbUser(user)}, nil
}

func (srv *userServer) Bind(
	ctx context.Context, req *v1pb.UserBindRequest) (
	_ *v1pb.UserBindResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}
	_, state, err := srv.CheckLoginState(ctx, trusted.AppId, req.State)
	if err != nil {
		return
	}

	resp := &v1pb.UserBindResponse{}
	if resp.AcctIds, err = db.BindAcctIdToUser(
		ctx, trusted.AppId, trusted.UserId,
		state.AcctIds,
		state.Properties.GetTakeOverAcctIdIfDuplicated()); err != nil {
		log.Warnf("failed to bind acct to user: %v", err)
		return
	}
	for _, acctId := range state.AcctIds {
		log.Infow("user bind acct", "user_id", trusted.UserId, "acct_id", acctId)
	}
	return resp, nil
}

func (srv *userServer) Unbind(
	ctx context.Context, req *v1pb.UserUnbindRequest) (
	_ *v1pb.UserUnbindResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}

	resp := &v1pb.UserUnbindResponse{}
	if resp.AcctIds, err = db.UnbindAcctIdFromUser(
		ctx, trusted.AppId, trusted.UserId, req.AcctIds); err != nil {
		log.Warnf("failed to unbind acct from user: %v", err)
		return
	}
	for _, acctId := range req.AcctIds {
		log.Infow("user unbind acct", "user_id", trusted.UserId, "acct_id", acctId)
	}
	return resp, nil
}

func (srv *userServer) SetMetadata(
	ctx context.Context, req *v1pb.UserSetMetadataRequest) (
	_ *v1pb.UserSetMetadataResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}

	for k, v := range req.Metadata {
		if len(k)+len(v) > 1024 {
			return nil, errMetadataTooLarge
		}
	}
	if err = db.SetUserMetadata(
		ctx, trusted.AppId, trusted.UserId, req.Metadata); err != nil {
		log.Warnf("failed to set user metadata: %v", err)
		return
	}
	return &v1pb.UserSetMetadataResponse{}, nil
}

func (srv *userServer) GetMetadata(
	ctx context.Context, req *v1pb.UserGetMetadataRequest) (
	resp *v1pb.UserGetMetadataResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}

	user, err := db.GetUser(ctx, trusted.AppId, trusted.UserId)
	if err != nil {
		return
	}

	if len(req.Keys) == 0 {
		resp = &v1pb.UserGetMetadataResponse{
			Metadata: user.Metadata,
		}
	} else {
		resp = &v1pb.UserGetMetadataResponse{
			Metadata: make(map[string]string),
		}
		for _, key := range req.Keys {
			if value, ok := user.Metadata[key]; ok {
				resp.Metadata[key] = value
			}
		}
	}
	return
}
