package comm

import (
	"encoding/json"
	"strings"

	"github.com/ntons/libra/librad/internal/util"
	logcfg "github.com/ntons/log-go/config"
)

// main configuration instance
var Config = &struct {
	// serving address
	Bind string
	// development | production
	Env string
	// modularized service configuration
	Services map[string]json.RawMessage
	// log configuration
	Log *logcfg.Config
}{}

func IsDevEnv() bool {
	return strings.HasPrefix(strings.ToLower(Config.Env), "dev")
}

func LoadConfig(filePath string) (err error) {
	return util.LoadFromFile(filePath, Config)
}
