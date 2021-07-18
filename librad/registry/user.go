package registry

import (
	"context"
	"strings"
	"time"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"github.com/ntons/tongo/httputil"
	"github.com/ntons/tongo/sign"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/internal/comm"
)

func fromDbUser(x *dbUser) *v1pb.UserData {
	return &v1pb.UserData{
		Id:       x.Id,
		AcctIds:  x.AcctIds,
		Metadata: x.Metadata,
	}
}

type userServer struct {
	v1pb.UnimplementedUserServer
}

func newUserServer() *userServer {
	return &userServer{}
}

func (srv *userServer) CheckUniformLoginState(
	ctx context.Context, app *xApp, state *v1pb.UniformLoginState) (err error) {
	if err = checkNonce(ctx, app.Id, state.Nonce); err != nil {
		return
	}

	// ts-30 签名有效期只有30秒钟
	// ts+5  是为了容忍一定的系统时间误差
	ts := time.Now().Unix()
	if state.Timestamp < ts-30 || state.Timestamp > ts+5 {
		return errInvalidTimestamp
	}

	signature := state.Signature
	state.Signature = ""
	expected := sign.ProtoHMACWithSHA1(state, app.Secret)
	if !strings.EqualFold(signature, expected) {
		log.Warnf("signature mismatch: %s, %s, %s, %s",
			signature, expected, app.Secret, state)
		return errInvalidSignature
	}

	return
}

func (srv *userServer) Login(
	ctx context.Context, req *v1pb.UserLoginRequest) (
	_ *v1pb.UserLoginResponse, err error) {
	app := findAppById(req.AppId)
	if app == nil {
		log.Warnf("invalid app id: %v", req.AppId)
		return nil, errInvalidAppId
	}
	userIp := httputil.GetRemoteIpFromContext(ctx)

	var acctIds []string
	if err = func() (err error) {
		state, err := anypb.UnmarshalNew(req.State, proto.UnmarshalOptions{})
		if err != nil {
			log.Warnf("failed to unmarshal state: %v", err)
			return errInvalidState
		}
		switch state := state.(type) {
		case *v1pb.DevLoginState:
			if !comm.IsDevEnv() {
				return errInvalidState
			}
			acctIds = append(acctIds, "dev$"+state.Username)
		case *v1pb.UniformLoginState:
			if err = srv.CheckUniformLoginState(ctx, app, state); err != nil {
				return
			}
			if state.UserIp != "" {
				userIp = state.UserIp
			}
			acctIds = state.AcctIds
		default:
			log.Warnf("unhandled state type: %T", state)
			return errInvalidState
		}
		return
	}(); err != nil {
		log.Warnf("failed to check state: %v", err)
		return
	}

	user, err := loginUser(ctx, app, userIp, acctIds, req.Options)
	if err != nil {
		log.Warnf("failed to login user: %v", err)
		return
	}
	if user.BanTime.After(time.Now()) {
		return nil, newPermissionDeniedError(&struct {
			BanTime   int64  `json:"ban_time"`
			BanReason string `json:"ban_reason"`
		}{
			BanTime:   user.BanTime.Unix(),
			BanReason: user.BanReason,
		})
	}

	sess, err := newSess(ctx, app, user.Id)
	if err != nil {
		log.Warnf("failed to new session: %v", err)
		return
	}

	log.Infow("user login", "user_id", user.Id, "acct_ids", acctIds)

	grpc.SetHeader(ctx, metadata.Pairs(L.XLibraToken, sess.Token))

	return &v1pb.UserLoginResponse{User: fromDbUser(user)}, nil
}

func (srv *userServer) Bind(
	ctx context.Context, req *v1pb.UserBindRequest) (
	_ *v1pb.UserBindResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}

	app := findAppById(trusted.AppId)
	if app == nil {
		log.Warnf("invalid app id: %v", trusted.AppId)
		return nil, errInvalidAppId
	}

	state, err := anypb.UnmarshalNew(req.State, proto.UnmarshalOptions{})
	if err != nil {
		return nil, errInvalidState
	}

	var acctIds []string
	switch state := state.(type) {
	case *v1pb.UniformLoginState:
		if err = srv.CheckUniformLoginState(ctx, app, state); err != nil {
			return
		}
		acctIds = state.AcctIds
	default:
		log.Warnf("unhandled state type: %T", state)
		return nil, errInvalidState
	}

	resp := &v1pb.UserBindResponse{}
	if resp.AcctIds, err = bindAcctIdToUser(
		ctx, trusted.AppId, trusted.UserId, acctIds, req.Options); err != nil {
		log.Warnf("failed to bind acct to user: %v", err)
		return
	}
	for _, acctId := range acctIds {
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
	if resp.AcctIds, err = unbindAcctIdFromUser(
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
	if err = setUserMetadata(
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

	user, err := getUser(ctx, trusted.AppId, trusted.UserId)
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
