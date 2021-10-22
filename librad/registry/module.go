package registry

import (
	"encoding/json"

	"github.com/onemoreteam/httpframework/modularity"
	sm "github.com/onemoreteam/httpframework/modularity/server"

	admv1pb "github.com/ntons/libra-go/api/libra/admin/v1"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
	authpb "github.com/ntons/libra/librad/registry/envoy_service_auth_v3"
)

func init() {
	modularity.Register(&module{})
}

type module struct {
	modularity.Skeleton
	srv *server
}

func (module) Name() string { return "database" }

func (m *module) Initialize(jb json.RawMessage) (err error) {
	if m.srv, err = createServer(jb); err != nil {
		return
	}
	admv1pb.RegisterAppAdminServer(sm.Default, m.srv.appAdmin)
	v1pb.RegisterUserServer(sm.Default, m.srv.user)
	v1pb.RegisterRoleServer(sm.Default, m.srv.role)
	authpb.RegisterAuthorizationServer(sm.Default, m.srv.auth)
	v1pb.RegisterUserAdminServer(sm.Default, m.srv.userAdmin)
	v1pb.RegisterRoleAdminServer(sm.Default, m.srv.roleAdmin)
	return
}
