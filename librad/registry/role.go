package registry

import (
	"context"

	v1pb "github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func fromRole(x *xRole) *v1pb.RoleData {
	return &v1pb.RoleData{
		Id:       x.Id,
		Index:    x.Index,
		Metadata: x.Metadata,
	}
}
func fromRoleList(a []*xRole) []*v1pb.RoleData {
	r := make([]*v1pb.RoleData, 0, len(a))
	for _, x := range a {
		r = append(r, fromRole(x))
	}
	return r
}

type roleServer struct {
	v1pb.UnimplementedRoleServer
}

func newRoleServer() *roleServer {
	return &roleServer{}
}

func (srv *roleServer) List(
	ctx context.Context, req *v1pb.RoleListRequest) (
	resp *v1pb.RoleListResponse, err error) {
	appId, userId, ok := getSessionFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	roles, err := listRoles(ctx, appId, userId)
	if err != nil {
		log.Warnf("failed to list roles: %v", err)
		return
	}
	return &v1pb.RoleListResponse{Roles: fromRoleList(roles)}, nil
}
func (srv *roleServer) Create(
	ctx context.Context, req *v1pb.RoleCreateRequest) (
	resp *v1pb.RoleCreateResponse, err error) {
	appId, userId, ok := getSessionFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	role, err := createRole(ctx, appId, userId, req.Index)
	if err != nil {
		return
	}
	return &v1pb.RoleCreateResponse{Role: fromRole(role)}, nil
}
func (srv *roleServer) SignIn(
	ctx context.Context, req *v1pb.RoleSignInRequest) (
	resp *v1pb.RoleSignInResponse, err error) {
	appId, userId, ok := getSessionFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
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
	return &v1pb.RoleSignInResponse{}, nil
}
func (srv *roleServer) SetMetadata(
	ctx context.Context, req *v1pb.RoleSetMetadataRequest) (
	resp *v1pb.RoleSetMetadataResponse, err error) {
	appId, userId, ok := getSessionFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	if err = setRoleMetadata(
		ctx, appId, userId, req.RoleId, req.Metadata); err != nil {
		return
	}
	return &v1pb.RoleSetMetadataResponse{}, nil
}
