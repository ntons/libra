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

	"github.com/ntons/libra/librad/db"
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
		r.BanAt = x.BanAt.Unix()
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

func (srv *userServer) Get(
	ctx context.Context, req *v1pb.UserGetRequest) (
	_ *v1pb.UserGetResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, errUnauthenticated
	}
	var (
		userIds = req.Ids
		roleIds []string
	)
	if req.Options != nil && req.Options.Fuzzy {
		userIds = make([]string, 0, len(req.Ids))
		roleIds = make([]string, 0, len(req.Ids))
		for _, id := range req.Ids {
			if _, tag, _ := db.DecId(id); tag == db.UserIdTag {
				userIds = append(userIds, id)
			} else if tag == db.RoleIdTag {
				roleIds = append(roleIds, id)
			}
		}
	}
	if len(roleIds) > 0 {
		roles, err := db.GetRoles(ctx, trusted.AppId, roleIds)
		if err != nil {
			log.Warnf("failed to get roles: %v", err)
			return nil, db.ErrDatabaseUnavailable
		}
		for _, role := range roles {
			userIds = append(userIds, role.UserId)
		}
	}
	var (
		users []*db.User
		roles []*db.Role
	)
	if users, err = db.GetUsers(ctx, trusted.AppId, userIds); err != nil {
		log.Warnf("failed to get users: %v", err)
		return nil, db.ErrDatabaseUnavailable
	}
	if req.Options != nil && req.Options.WithRoles {
		if roles, err = db.GetRolesByUserId(
			ctx, trusted.AppId, userIds); err != nil {
			log.Warnf("failed to get roles: %v", err)
			return nil, db.ErrDatabaseUnavailable
		}
	}
	return &v1pb.UserGetResponse{
		Users: fromDbUsers(users),
		Roles: fromDbRoles(roles),
	}, nil
}

func (srv *userServer) Query(
	ctx context.Context, req *v1pb.UserQueryRequest) (
	_ *v1pb.UserQueryResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, errUnauthenticated
	}

	var acctIds []string
	if req.Filters == nil {
		return nil, newInvalidArgumentError("require filters")
	} else if req.Filters.AcctDetails != nil {
		if acctIds, err = db.GetAcctIdByDetail(
			ctx,
			trusted.AppId,
			req.Filters.AcctDetails,
		); err != nil {
			return nil, newInternalError("failed to query")
		}
	} else {
		return nil, newInvalidArgumentError("require filters")
	}

	resp := &v1pb.UserQueryResponse{}
	if len(acctIds) > 0 {
		var (
			users []*db.User
			roles []*db.Role
		)
		if users, err = db.GetUsersByAcctId(
			ctx,
			trusted.AppId,
			acctIds...,
		); err != nil {
			return
		}

		if req.Options != nil && req.Options.WithRoles {
			var userIds []string
			for _, user := range users {
				userIds = append(userIds, user.Id)
			}
			if roles, err = db.GetRolesByUserId(
				ctx, trusted.AppId, userIds); err != nil {
				log.Warnf("failed to get roles: %v", err)
				return nil, db.ErrDatabaseUnavailable
			}
		}

		resp.Users = fromDbUsers(users)
		resp.Roles = fromDbRoles(roles)
	}
	return resp, nil
}

func (srv *userServer) Ban(
	ctx context.Context, req *v1pb.UserBanRequest) (
	_ *v1pb.UserBanResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil || !db.IdBelongToAppId(trusted.AppId, req.UserIds...) {
		return nil, errUnauthenticated
	}

	resp := &v1pb.UserBanResponse{}

	if len(req.UserIds) > 0 {
		var (
			appId   = trusted.AppId
			userIds = req.UserIds
			now     = time.Now()
		)

		if req.Seconds > 0 {
			var (
				banTo  = now.Add(time.Duration(req.Seconds) * time.Second)
				banFor = req.Reason
			)
			if err = db.BanUsers(ctx, appId, userIds, banTo, banFor); err != nil {
				log.Warnf("failed to ban users: %v", err)
				return nil, db.ErrDatabaseUnavailable
			}
			if err = db.LogoutUser(ctx, userIds...); err != nil {
				log.Warnf("failed to logout users: %v", err)
				return nil, db.ErrDatabaseUnavailable
			}
		} else if req.Seconds < 0 {
			if err = db.UnbanUsers(ctx, appId, userIds); err != nil {
				log.Warnf("failed to unban users: %v", err)
				return nil, db.ErrDatabaseUnavailable
			}
		}

		fields := []string{"ban_to", "ban_for"}
		users, err := db.GetUsersWithFields(ctx, appId, userIds, fields)
		if err != nil {
			log.Warnf("failed to get users: %v", err)
			return nil, db.ErrDatabaseUnavailable
		}

		for _, user := range users {
			state := &v1pb.UserBanState{Id: user.Id}
			if user.BanTo.After(now) {
				state.BanAt = user.BanAt.Unix()
				state.BanTo = user.BanTo.Unix()
				state.BanFor = user.BanFor
			}
			resp.States = append(resp.States, state)
		}
	}
	return resp, nil
}

func (srv *userServer) Block(
	ctx context.Context, req *v1pb.UserBlockRequest) (
	_ *v1pb.UserBlockResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, errUnauthenticated
	}

	var keys []string
	keys = append(keys, req.Keys...)
	keys = append(keys, req.AcctIds...)
	keys = append(keys, req.DeviceIds...)
	keys = append(keys, req.ClientIps...)

	if len(keys) > 0 {
		var (
			appId = trusted.AppId
			now   = time.Now()
		)
		if req.Seconds > 0 {
			var (
				banTo  = now.Add(time.Duration(req.Seconds) * time.Second)
				banFor = req.Reason
			)
			if err = db.Block(ctx, appId, keys, banTo, banFor); err != nil {
				log.Warnf("failed to ban keys: %v", err)
				return nil, db.ErrDatabaseUnavailable
			}
		} else if req.Seconds < 0 {
			if err = db.Allow(ctx, appId, keys); err != nil {
				log.Warnf("failed to unban keys: %v", err)
				return nil, db.ErrDatabaseUnavailable
			}
		}
	}

	return &v1pb.UserBlockResponse{}, nil
}

func (srv *userServer) BindAcctId(
	ctx context.Context, req *v1pb.UserBindAcctIdRequest) (
	_ *v1pb.UserBindAcctIdResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil || !db.IdBelongToAppId(trusted.AppId, req.UserId) {
		return nil, errUnauthenticated
	}
	if _, err = db.BindAcctIdToUser(
		ctx, trusted.AppId, req.UserId, req.AcctIds,
		req.TakeOverAcctIdIfDuplicated,
	); err != nil {
		log.Warnf("failed to transfer acct id: %v", err)
		return nil, db.ErrDatabaseUnavailable
	}
	return &v1pb.UserBindAcctIdResponse{}, nil
}

func (srv *userServer) CheckLoginState(
	ctx context.Context, appId string, anyState *anypb.Any) (
	app *db.App, state *v1pb.UniformLoginState, err error) {
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
	_ *v1pb.UniformLoginState, err error) {
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
	return state, nil
}

func (srv *userServer) Login(
	ctx context.Context, req *v1pb.UserLoginRequest) (
	_ *v1pb.UserLoginResponse, err error) {
	appId := req.AppId

	app, state, err := srv.CheckLoginState(ctx, appId, req.State)
	if err != nil {
		return
	}

	clientIp := req.GetClient().GetIp()
	if clientIp == "" {
		clientIp = state.UserIp
	}
	if clientIp == "" {
		clientIp = httputil.GetRemoteIpFromContext(ctx)
	}

	deviceId := req.GetDevice().GetId()

	user, sess, err := db.LoginUser(
		ctx, app, clientIp, deviceId, state.AcctIds,
		req.CreateIfNotFound)
	if err != nil {
		log.Warnf("failed to login user: %v", err)
		return
	}

	log.Infow("user login", "user_id", user.Id, "acct_ids", state.AcctIds)

	for id, detail := range state.AcctDetails {
		if detail == "" {
			continue
		}
		if err := db.UpdateAcctDetail(ctx, appId, id, detail); err != nil {
			log.Warnw(
				"failed to update acct detail",
				"app_id", appId,
				"acct_id", id,
				"detail", detail,
				"error", err,
			)
		}
	}

	//grpc.SetHeader(ctx, metadata.Pairs(L.XLibraToken, sess.Token))
	grpc.SetTrailer(ctx, metadata.Pairs(L.XLibraToken, sess.Token))

	return &v1pb.UserLoginResponse{User: fromDbUser(user)}, nil
}

func (srv *userServer) Bind(
	ctx context.Context, req *v1pb.UserBindRequest) (
	_ *v1pb.UserBindResponse, err error) {
	var appId, userId string
	if trusted := L.RequireAuthByToken(ctx); trusted != nil {
		appId, userId = trusted.AppId, trusted.UserId
	} else if trusted := L.RequireAuthBySecret(ctx); trusted != nil {
		appId, userId = trusted.AppId, req.UserId
	} else {
		return nil, errLoginRequired
	}

	_, state, err := srv.CheckLoginState(ctx, appId, req.State)
	if err != nil {
		return
	}

	resp := &v1pb.UserBindResponse{}
	if resp.AcctIds, err = db.BindAcctIdToUser(
		ctx, appId, userId, state.AcctIds,
		state.GetProperties().GetTakeOverAcctIdIfDuplicated()); err != nil {
		log.Warnf("failed to bind acct to user: %v", err)
		return
	}
	for _, acctId := range state.AcctIds {
		log.Infow("user bind acct", "user_id", userId, "acct_id", acctId)
	}
	return resp, nil
}

func (srv *userServer) Unbind(
	ctx context.Context, req *v1pb.UserUnbindRequest) (
	_ *v1pb.UserUnbindResponse, err error) {
	var appId, userId string
	if trusted := L.RequireAuthByToken(ctx); trusted != nil {
		appId, userId = trusted.AppId, trusted.UserId
	} else if trusted := L.RequireAuthBySecret(ctx); trusted != nil {
		appId, userId = trusted.AppId, req.UserId
	} else {
		return nil, errLoginRequired
	}

	resp := &v1pb.UserUnbindResponse{}
	if resp.AcctIds, err = db.UnbindAcctIdFromUser(
		ctx, appId, userId, req.AcctIds); err != nil {
		log.Warnf("failed to unbind acct from user: %v", err)
		return
	}
	for _, acctId := range req.AcctIds {
		log.Infow("user unbind acct", "user_id", userId, "acct_id", acctId)
	}
	return resp, nil
}

func (srv *userServer) SetMetadata(
	ctx context.Context, req *v1pb.UserSetMetadataRequest) (
	_ *v1pb.UserSetMetadataResponse, err error) {
	var appId, userId string
	if trusted := L.RequireAuthByToken(ctx); trusted != nil {
		appId, userId = trusted.AppId, trusted.UserId
	} else if trusted := L.RequireAuthBySecret(ctx); trusted != nil {
		appId, userId = trusted.AppId, req.UserId
	} else {
		return nil, errLoginRequired
	}

	for k, v := range req.Metadata {
		if len(k)+len(v) > 1024 {
			return nil, errMetadataTooLarge
		}
	}
	if err = db.SetUserMetadata(
		ctx, appId, userId, req.Metadata); err != nil {
		log.Warnf("failed to set user metadata: %v", err)
		return
	}
	return &v1pb.UserSetMetadataResponse{}, nil
}

func (srv *userServer) GetMetadata(
	ctx context.Context, req *v1pb.UserGetMetadataRequest) (
	resp *v1pb.UserGetMetadataResponse, err error) {
	var appId, userId string
	if trusted := L.RequireAuthByToken(ctx); trusted != nil {
		appId, userId = trusted.AppId, trusted.UserId
	} else if trusted := L.RequireAuthBySecret(ctx); trusted != nil {
		appId, userId = trusted.AppId, req.UserId
	} else {
		return nil, errLoginRequired
	}

	user, err := db.GetUser(ctx, appId, userId)
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
