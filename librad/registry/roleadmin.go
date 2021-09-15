package registry

import (
	"context"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	log "github.com/ntons/log-go"
)

type roleAdminServer struct {
	v1pb.UnimplementedRoleAdminServer
}

func newRoleAdminServer() *roleAdminServer {
	return &roleAdminServer{}
}

func (srv *roleAdminServer) Get(
	ctx context.Context, req *v1pb.RoleAdminGetRequest) (
	res *v1pb.RoleAdminGetResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, errUnauthenticated
	}
	roles, err := getRoles(ctx, trusted.AppId, req.Ids)
	if err != nil {
		log.Warnf("failed to get roles: %v", err)
		return nil, errDatabaseUnavailable
	}
	return &v1pb.RoleAdminGetResponse{
		Roles: fromDbRoles(roles),
	}, nil

}
