package portal

import (
	"context"
	"strings"

	"github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"github.com/ntons/tongo/sign"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func toUser(x *dbUser) *v1.User {
	return &v1.User{
		Id:       x.Id,
		AcctId:   x.AcctId,
		Metadata: x.Metadata,
	}
}
func toRole(x *dbRole) *v1.Role {
	return &v1.Role{
		Id:       x.Id,
		UserId:   x.UserId,
		Metadata: x.Metadata,
	}
}
func toUsers(a []*dbUser) []*v1.User {
	r := make([]*v1.User, 0, len(a))
	for _, x := range a {
		r = append(r, toUser(x))
	}
	return r
}
func toRoles(a []*dbRole) []*v1.Role {
	r := make([]*v1.Role, 0, len(a))
	for _, x := range a {
		r = append(r, toRole(x))
	}
	return r
}

type account struct {
	v1.UnimplementedAccountServer
	db *database
}

func newAccount(db *database) *account {
	return &account{db: db}
}

// implement v1.AcctServer
func (acct *account) Login(
	ctx context.Context, req *v1.AccountLoginRequest) (
	resp *v1.AccountLoginResponse, err error) {
	app, err := acct.db.getApp(req.AppId)
	if err != nil {
		log.Warnf("invalid app id: %v", req.AppId)
		return nil, errInvalidAppId
	}
	acctId, err := acct.checkLoginState(app, req.State)
	if err != nil {
		return
	}
	user, err := acct.db.loginUser(ctx, app, acctId)
	if err != nil {
		log.Warnf("failed to login user: %v", err)
		return
	}
	token, err := acct.db.newToken(ctx, app, user.Id)
	if err != nil {
		log.Warnf("failed to new token: %v", err)
		return
	}
	grpc.SendHeader(ctx, metadata.Pairs(xCookie+"-"+xTokenKey, token))
	resp = &v1.AccountLoginResponse{Token: token, User: toUser(user)}
	return
}
func (acct *account) checkLoginState(
	app *dbApp, any *anypb.Any) (acctId []string, err error) {
	state, err := anypb.UnmarshalNew(any, proto.UnmarshalOptions{})
	if err != nil {
		log.Warnf("failed to unmarshal state: %v", err)
		return nil, errInvalidState
	}
	switch state := state.(type) {
	case *v1.DevLoginState:
		acctId = []string{"dev$" + state.Username}
	case *v1.UniformLoginState:
		signature := state.Signature
		state.Signature = ""
		if !strings.EqualFold(
			signature, sign.ProtoHMACWithSHA1(state, app.Secret)) {
			return nil, errInvalidSignature
		}
		acctId = state.AcctId
	default:
		log.Warnf("unhandled state type: %T", state)
		return nil, errInvalidState
	}
	return
}

func (acct *account) Bind(
	ctx context.Context, req *v1.AccountBindRequest) (
	resp *v1.AccountBindResponse, err error) {
	app, userId, err := acct.checkToken(ctx, req)
	if err != nil {
		return
	}
	if err = acct.db.bindAcctIdToUser(
		ctx, app.Id, userId, req.AcctId); err != nil {
		return
	}
	return
}

func (acct *account) ListRoles(
	ctx context.Context, req *v1.AccountListRolesRequest) (
	resp *v1.AccountListRolesResponse, err error) {
	app, userId, err := acct.checkToken(ctx, req)
	if err != nil {
		return
	}
	roles, err := acct.db.listRoles(ctx, app.Id, userId)
	resp = &v1.AccountListRolesResponse{Roles: toRoles(roles)}
	return
}

func (acct *account) CreateRole(
	ctx context.Context, req *v1.AccountCreateRoleRequest) (
	resp *v1.AccountCreateRoleResponse, err error) {
	app, userId, err := acct.checkToken(ctx, req)
	if err != nil {
		return
	}
	role, err := acct.db.createRole(ctx, app, userId, req.Index)
	if err != nil {
		return
	}
	resp = &v1.AccountCreateRoleResponse{Role: toRole(role)}
	return
}

func (acct *account) SignIn(
	ctx context.Context, req *v1.AccountSignInRequest) (
	resp *v1.AccountSignInResponse, err error) {
	app, userId, err := acct.checkToken(ctx, req)
	if err != nil {
		return
	}
	role, err := acct.db.getRole(ctx, app.Id, req.RoleId)
	if err != nil {
		return
	}
	if userId != role.UserId {
		return nil, errRoleNotFound
	}
	ticket, err := acct.db.newTicket(ctx, app, role.Id)
	if err != nil {
		return
	}
	resp = &v1.AccountSignInResponse{Ticket: ticket, Role: toRole(role)}
	return
}

func (acct *account) SetUserMetadata(
	ctx context.Context, req *v1.AccountSetUserMetadataRequest) (
	resp *v1.AccountSetUserMetadataResponse, err error) {
	app, userId, err := acct.checkToken(ctx, req)
	if err != nil {
		return
	}
	if err = acct.db.setUserMetadata(
		ctx, app.Id, userId, req.Metadata); err != nil {
		return
	}
	return
}
func (acct *account) SetRoleMetadata(
	ctx context.Context, req *v1.AccountSetRoleMetadataRequest) (
	resp *v1.AccountSetRoleMetadataResponse, err error) {
	app, userId, err := acct.checkToken(ctx, req)
	if err != nil {
		return
	}
	if err = acct.db.setRoleMetadata(
		ctx, app.Id, userId, req.RoleId, req.Metadata); err != nil {
		return
	}
	return
}

type apiReq interface {
	GetAppId() string
	GetToken() string
}

func (acct *account) checkToken(
	ctx context.Context, req apiReq) (app *dbApp, userId string, err error) {
	if app, err = acct.db.getApp(req.GetAppId()); err != nil {
		return
	}
	token := req.GetToken()
	if md, ok := metadata.FromIncomingContext(
		ctx); ok && len(md.Get(xTokenKey)) > 0 {
		token = md.Get(xTokenKey)[0]
	}
	if token == "" {
		return nil, "", errInvalidToken
	}
	if userId, err = acct.db.checkToken(ctx, app, token); err != nil {
		return
	}
	return
}
