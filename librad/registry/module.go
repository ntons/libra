package registry

import (
	"encoding/json"

	"github.com/onemoreteam/httpframework/modularity"
	sm "github.com/onemoreteam/httpframework/modularity/server"

	admv1pb "github.com/ntons/libra-go/api/libra/admin/v1"
	v1pb "github.com/ntons/libra-go/api/libra/v1"
)

func init() {
	modularity.Register(&module{})
}

type module struct {
	modularity.Skeleton
}

func (module) Name() string { return "registry" }

func (m *module) Initialize(jb json.RawMessage) (err error) {
	var (
		appAdmin = newAppServer()
		user     = newUserServer()
		role     = newRoleServer()
	)
	sm.RegisterService(&admv1pb.AppAdmin_ServiceDesc, appAdmin)
	sm.RegisterService(&v1pb.User_ServiceDesc, user)
	sm.RegisterService(&v1pb.Role_ServiceDesc, role)
	return
}
