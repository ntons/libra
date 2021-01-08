package portal

import (
	"context"

	"github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/ntons/libra/librad/comm"
)

func toRoleData(x *xRole) *v1.RoleData {
	return &v1.RoleData{
		Id:       x.Id,
		Metadata: x.Metadata,
	}
}
func toRoleDataList(a []*xRole) []*v1.RoleData {
	r := make([]*v1.RoleData, 0, len(a))
	for _, x := range a {
		r = append(r, toRoleData(x))
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
	sess := getSessFromContext(ctx)
	roles, err := db.listRoles(ctx, sess.appId, sess.userId)
	if err != nil {
		log.Warnf("failed to list roles: %v", err)
		return
	}
	return &v1.RoleListResponse{Roles: toRoleDataList(roles)}, nil
}
func (srv *roleServer) Create(
	ctx context.Context, req *v1.RoleCreateRequest) (
	resp *v1.RoleCreateResponse, err error) {
	sess := getSessFromContext(ctx)
	role, err := db.createRole(ctx, sess.appId, sess.userId, req.Index)
	if err != nil {
		return
	}
	return &v1.RoleCreateResponse{Role: toRoleData(role)}, nil
}
func (srv *roleServer) SignIn(
	ctx context.Context, req *v1.RoleSignInRequest) (
	resp *v1.RoleSignInResponse, err error) {
	sess := getSessFromContext(ctx)
	role, err := db.getRole(ctx, sess.appId, req.RoleId)
	if err != nil {
		return
	}
	if sess.userId != role.UserId {
		return nil, errRoleNotFound
	}
	ticket, err := db.newTicket(ctx, sess.appId, role.Id)
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
	sess := getSessFromContext(ctx)
	if err = db.setRoleMetadata(
		ctx, sess.appId, sess.userId, req.RoleId, req.Metadata); err != nil {
		return
	}
	return &v1.RoleSetMetadataResponse{}, nil
}
