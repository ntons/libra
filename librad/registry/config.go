package registry

import (
	"time"
)

var cfg = &config{}

type config struct {
	Auth struct { // 登录态验证
		Redis []string
	}
	Nonce struct { // nonce检查
		Redis   []string
		Timeout string
		// parsed to
		timeout time.Duration
	}
	Mongo string // 配置/注册DB
}

func (cfg *config) Parse() (err error) {
	if s := cfg.Nonce.Timeout; s != "" {
		if cfg.Nonce.timeout, err = time.ParseDuration(s); err != nil {
			return
		}
	} else {
		cfg.Nonce.timeout = time.Hour
	}
	return
}
