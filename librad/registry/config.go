package registry

import (
	"time"
)

var cfg = &xConfig{}

type xConfig struct {
	// 登录态验证
	Auth struct {
		Redis []string
	}
	// 随机数验证
	Nonce struct {
		Redis   []string
		Timeout string
		// parsed to
		timeout time.Duration
	}
	// 配置/注册DB
	Mongo string
	// 每个App都有的通用权限
	CommonPermissions []*xPermission
}

func (cfg *xConfig) parse() (err error) {
	if s := cfg.Nonce.Timeout; s != "" {
		if cfg.Nonce.timeout, err = time.ParseDuration(s); err != nil {
			return
		}
	} else {
		cfg.Nonce.timeout = time.Hour
	}
	for _, p := range cfg.CommonPermissions {
		if err = p.parse(); err != nil {
			return
		}
	}
	return
}
func (cfg *xConfig) isPermitted(path string) bool {
	for _, p := range cfg.CommonPermissions {
		if p.match(path) {
			return true
		}
	}
	return false
}
