package registry

import (
	"context"

	v1pb "github.com/ntons/libra-go/api/v1"
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
	appId, ok := getTrustedAppId(ctx)
	if !ok {
		return nil, errUnauthenticated
	}
	if err = setUserMetadata(ctx, appId, req.UserId, req.Metadata); err != nil {
		log.Warnf("failed to set user metadata: %v", err)
		return
	}
	return &v1pb.UserAdminSetMetadataResponse{}, nil
}
