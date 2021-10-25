package registry

import (
	"context"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	log "github.com/ntons/log-go"

	"github.com/ntons/libra/librad/db"
)

func fromDbRole(x *db.Role) *v1pb.RoleData {
	return &v1pb.RoleData{
		Id:       x.Id,
		Index:    x.Index,
		UserId:   x.UserId,
		Metadata: x.Metadata,
	}
}
func fromDbRoles(a []*db.Role) []*v1pb.RoleData {
	r := make([]*v1pb.RoleData, 0, len(a))
	for _, x := range a {
		r = append(r, fromDbRole(x))
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
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}
	roles, err := db.ListRoles(ctx, trusted.AppId, trusted.UserId)
	if err != nil {
		log.Warnf("failed to list roles: %v", err)
		return
	}
	return &v1pb.RoleListResponse{Roles: fromDbRoles(roles)}, nil
}
func (srv *roleServer) Create(
	ctx context.Context, req *v1pb.RoleCreateRequest) (
	resp *v1pb.RoleCreateResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}
	role, err := db.CreateRole(ctx, trusted.AppId, trusted.UserId, req.Index)
	if err != nil {
		return
	}
	return &v1pb.RoleCreateResponse{Role: fromDbRole(role)}, nil
}
func (srv *roleServer) SignIn(
	ctx context.Context, req *v1pb.RoleSignInRequest) (
	resp *v1pb.RoleSignInResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}
	if err = db.SignInRole(ctx, trusted.AppId, trusted.UserId, req.RoleId); err != nil {
		return
	}
	return &v1pb.RoleSignInResponse{}, nil
}
func (srv *roleServer) SetMetadata(
	ctx context.Context, req *v1pb.RoleSetMetadataRequest) (
	resp *v1pb.RoleSetMetadataResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}
	for k, v := range req.Metadata {
		if len(k)+len(v) > 1024 {
			return nil, errMetadataTooLarge
		}
	}
	if err = db.SetRoleMetadata(
		ctx, trusted.AppId, trusted.UserId, req.RoleId, req.Metadata); err != nil {
		return
	}
	return &v1pb.RoleSetMetadataResponse{}, nil
}

func (srv *roleServer) GetMetadata(
	ctx context.Context, req *v1pb.RoleGetMetadataRequest) (
	resp *v1pb.RoleGetMetadataResponse, err error) {
	trusted := L.RequireAuthByToken(ctx)
	if trusted == nil {
		return nil, errLoginRequired
	}

	role, err := db.GetRole(ctx, trusted.AppId, trusted.UserId, req.RoleId)
	if err != nil {
		return
	}

	if len(req.Keys) == 0 {
		resp = &v1pb.RoleGetMetadataResponse{
			Metadata: role.Metadata,
		}
	} else {
		resp = &v1pb.RoleGetMetadataResponse{
			Metadata: make(map[string]string),
		}
		for _, key := range req.Keys {
			if value, ok := role.Metadata[key]; ok {
				resp.Metadata[key] = value
			}
		}
	}
	return
}
