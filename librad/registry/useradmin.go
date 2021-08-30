package registry

import (
	"context"

	L "github.com/ntons/libra-go"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	log "github.com/ntons/log-go"
)

type userAdminServer struct {
	v1pb.UnimplementedUserAdminServer
}

func newUserAdminServer() *userAdminServer {
	return &userAdminServer{}
}

func (srv *userAdminServer) SetMetadata(
	ctx context.Context, req *v1pb.UserAdminSetMetadataRequest) (
	_ *v1pb.UserAdminSetMetadataResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, errUnauthenticated
	}
	if err = setUserMetadata(
		ctx, trusted.AppId, req.UserId, req.Metadata); err != nil {
		log.Warnf("failed to set user metadata: %v", err)
		return
	}
	return &v1pb.UserAdminSetMetadataResponse{}, nil
}

func (srv *userAdminServer) GetMetadata(
	ctx context.Context, req *v1pb.UserAdminGetMetadataRequest) (
	_ *v1pb.UserAdminGetMetadataResponse, err error) {
	trusted := L.RequireAuthBySecret(ctx)
	if trusted == nil {
		return nil, errUnauthenticated
	}
	user, err := getUser(ctx, trusted.AppId, req.UserId)
	if err != nil {
		log.Warnf("failed to get user: %v", err)
		return
	}
	return &v1pb.UserAdminGetMetadataResponse{
		Metadata: user.Metadata,
	}, nil
}
