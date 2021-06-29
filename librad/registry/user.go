package registry

import (
	"context"
	"strings"
	"time"

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
	if ok, err := checkNonce(ctx, app.Id, state.Nonce); err != nil {
		return errDatabaseUnavailable
	} else if !ok {
		return errInvalidNonce
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
	resp *v1pb.UserLoginResponse, err error) {
	app := xApps.findById(req.AppId)
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

	user, sess, err := loginUser(ctx, app, userIp, acctIds)
	if err != nil {
		log.Warnf("failed to login user: %v", err)
		return
	}

	log.Infow("user login", "user_id", user.Id, "acct_ids", acctIds)

	grpc.SetHeader(ctx, metadata.Pairs(xLibraToken, sess.Token))

	return &v1pb.UserLoginResponse{User: fromDbUser(user)}, nil
}

func (srv *userServer) Bind(
	ctx context.Context, req *v1pb.UserBindRequest) (
	resp *v1pb.UserBindResponse, err error) {
	appId, userId, ok := getTrustedFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}

	app := xApps.findById(appId)
	if app == nil {
		log.Warnf("invalid app id: %v", appId)
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

	if err = bindAcctToUser(ctx, appId, userId, acctIds); err != nil {
		log.Warnf("failed to bind acct to user: %v", err)
		return
	}
	for _, acctId := range acctIds {
		log.Infof("user bind acct", "user_id", userId, "acct_id", acctId)
	}
	return &v1pb.UserBindResponse{}, nil
}

func (srv *userServer) SetMetadata(
	ctx context.Context, req *v1pb.UserSetMetadataRequest) (
	resp *v1pb.UserSetMetadataResponse, err error) {
	appId, userId, ok := getTrustedFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	if err = setUserMetadata(ctx, appId, userId, req.Metadata); err != nil {
		log.Warnf("failed to set user metadata: %v", err)
		return
	}
	return &v1pb.UserSetMetadataResponse{}, nil
}
