package auth

import (
	"encoding/json"

	"github.com/onemoreteam/httpframework/modularity"
	sm "github.com/onemoreteam/httpframework/modularity/server"

	authpb "github.com/ntons/libra/librad/common/envoy_service_auth_v3"
)

func init() {
	modularity.Register(&module{})
}

type module struct {
	modularity.Skeleton
}

func (module) Name() string { return "auth" }

func (m *module) Initialize(jb json.RawMessage) (err error) {
	sm.RegisterService(&authpb.Authorization_ServiceDesc, newAuthServer())
	return
}
