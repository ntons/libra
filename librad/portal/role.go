package portal

import (
	"context"

	"github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/ntons/libra/librad/comm"
)

func fromRole(x *xRole) *v1.RoleData {
	return &v1.RoleData{
		Id:       x.Id,
		Index:    x.Index,
		Metadata: x.Metadata,
	}
}
func fromRoleList(a []*xRole) []*v1.RoleData {
	r := make([]*v1.RoleData, 0, len(a))
	for _, x := range a {
		r = append(r, fromRole(x))
	}
	return r
}

type roleServer struct {
	v1.UnimplementedRoleServer
	comm.GrpcUnaryInterceptor
}

func newRoleServer() *roleServer {
	return &roleServer{
		GrpcUnaryInterceptor: newTokenRequired(),
	}
}

func (srv *roleServer) List(
	ctx context.Context, req *v1.RoleListRequest) (
	resp *v1.RoleListResponse, err error) {
	appId, userId := getSessionFromContext(ctx)
	roles, err := listRoles(ctx, appId, userId)
	if err != nil {
		log.Warnf("failed to list roles: %v", err)
		return
	}
	return &v1.RoleListResponse{Roles: fromRoleList(roles)}, nil
}
func (srv *roleServer) Create(
	ctx context.Context, req *v1.RoleCreateRequest) (
	resp *v1.RoleCreateResponse, err error) {
	appId, userId := getSessionFromContext(ctx)
	role, err := createRole(ctx, appId, userId, req.Index)
	if err != nil {
		return
	}
	return &v1.RoleCreateResponse{Role: fromRole(role)}, nil
}
func (srv *roleServer) SignIn(
	ctx context.Context, req *v1.RoleSignInRequest) (
	resp *v1.RoleSignInResponse, err error) {
	appId, userId := getSessionFromContext(ctx)
	role, err := signInRole(ctx, appId, userId, req.RoleId)
	if err != nil {
		return
	}
	ticket, err := newTicket(ctx, appId, role)
	if err != nil {
		return
	}
	md := metadata.Pairs(xLibraTicket, ticket, xLibraCookieTicket, ticket)
	grpc.SetHeader(ctx, md)
	return &v1.RoleSignInResponse{}, nil
}
func (srv *roleServer) SetMetadata(
	ctx context.Context, req *v1.RoleSetMetadataRequest) (
	resp *v1.RoleSetMetadataResponse, err error) {
	appId, userId := getSessionFromContext(ctx)
	if err = setRoleMetadata(
		ctx, appId, userId, req.RoleId, req.Metadata); err != nil {
		return
	}
	return &v1.RoleSetMetadataResponse{}, nil
}
