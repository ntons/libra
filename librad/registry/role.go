package registry

import (
	"context"

	v1pb "github.com/ntons/libra-go/api/v1"
	log "github.com/ntons/log-go"
)

func fromDbRole(x *dbRole) *v1pb.RoleData {
	return &v1pb.RoleData{
		Id:       x.Id,
		Index:    x.Index,
		Metadata: x.Metadata,
	}
}
func fromDbRoleList(a []*dbRole) []*v1pb.RoleData {
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
	appId, userId, ok := getTrustedFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	roles, err := listRoles(ctx, appId, userId)
	if err != nil {
		log.Warnf("failed to list roles: %v", err)
		return
	}
	return &v1pb.RoleListResponse{Roles: fromDbRoleList(roles)}, nil
}
func (srv *roleServer) Create(
	ctx context.Context, req *v1pb.RoleCreateRequest) (
	resp *v1pb.RoleCreateResponse, err error) {
	appId, userId, ok := getTrustedFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	role, err := createRole(ctx, appId, userId, req.Index)
	if err != nil {
		return
	}
	return &v1pb.RoleCreateResponse{Role: fromDbRole(role)}, nil
}
func (srv *roleServer) SignIn(
	ctx context.Context, req *v1pb.RoleSignInRequest) (
	resp *v1pb.RoleSignInResponse, err error) {
	appId, userId, ok := getTrustedFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	if err = signInRole(ctx, appId, userId, req.RoleId); err != nil {
		return
	}
	return &v1pb.RoleSignInResponse{}, nil
}
func (srv *roleServer) SetMetadata(
	ctx context.Context, req *v1pb.RoleSetMetadataRequest) (
	resp *v1pb.RoleSetMetadataResponse, err error) {
	appId, userId, ok := getTrustedFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}
	if err = setRoleMetadata(
		ctx, appId, userId, req.RoleId, req.Metadata); err != nil {
		return
	}
	return &v1pb.RoleSetMetadataResponse{}, nil
}

func (srv *roleServer) GetMetadata(
	ctx context.Context, req *v1pb.RoleGetMetadataRequest) (
	resp *v1pb.RoleGetMetadataResponse, err error) {
	appId, userId, ok := getTrustedFromContext(ctx)
	if !ok {
		return nil, errLoginRequired
	}

	role, err := getRole(ctx, appId, userId, req.RoleId)
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
