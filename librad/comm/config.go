package comm

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/ntons/libra/librad/comm/util"
)

// main configuration instance
var Config = &struct {
	// serving address
	Bind string
	// unix domain socket used to forward gateway request to
	UnixDomainSock string
	// only grpc enabled
	GrpcOnly bool
	// development | production
	Env string

	// modularized service configuration
	Services map[string]json.RawMessage
	// log configuration
	Log *zap.Config
}{
	UnixDomainSock: fmt.Sprintf("/tmp/%s.sock", util.RandomString(10)),
}

func IsDevEnv() bool {
	return strings.HasPrefix(strings.ToLower(Config.Env), "dev")
}

func LoadConfig(filePath string) (err error) {
	return util.LoadFromFile(filePath, Config)
}
