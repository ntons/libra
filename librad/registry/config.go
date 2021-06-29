package registry

import (
	"regexp"
	"strings"
	"time"
)

type xPermission struct {
	Path   string `json:"path,omitempty" bson:"path,omitempty"`
	Prefix string `json:"prefix,omitempty" bson:"prefix,omitempty"`
	Regexp string `json:"regexp,omitempty" bson:"regexp,omitempty"`
	// pre-compiled regexp
	re *regexp.Regexp
}

func (x *xPermission) parse() (err error) {
	if x.Regexp != "" {
		if x.re, err = regexp.Compile(x.Regexp); err != nil {
			return
		}
	}
	return
}
func (x *xPermission) isPermitted(path string) bool {
	if x.Path != "" && path != x.Path {
		return false
	}
	if x.Prefix != "" && !strings.HasPrefix(path, x.Prefix) {
		return false
	}
	if x.re != nil && !x.re.MatchString(path) {
		return false
	}
	return true
}

var cfg = &xConfig{
	ConfigDBName: "librad",
}

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
	// 配置DB名字
	ConfigDBName string
	// AppDB前缀
	AppDBNamePrefix string
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

func getAppDBName(appId string) string {
	return cfg.AppDBNamePrefix + appId
}
