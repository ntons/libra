package registry

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
)

type xApp struct {
	// 应用ID
	Id string `bson:"_id"`
	// 数值形式的应用ID
	Key uint32 `bson:"key"`
	// 应用签名密钥，授权访问
	Secret string `bson:"secret,omitempty"`
	// 应用指纹指纹，特异化应用数据，增加安全性
	Fingerprint string `bson:"fingerprint,omitempty"`
	// 允许的服务
	Permissions []*xPermission `bson:"permissions,omitempty"`
	// AES密钥，由Fingerprint生成
	block cipher.Block
}

func (x *xApp) parse() (err error) {
	// check permission expression
	for _, p := range x.Permissions {
		if err = p.parse(); err != nil {
			return
		}
	}
	// hash fingerprint to 32 bytes byte array, NewCipher must success
	hash := sha256.Sum256([]byte(x.Fingerprint))
	x.block, _ = aes.NewCipher(hash[:])
	return
}
func (x *xApp) isPermitted(path string) bool {
	for _, p := range cfg.CommonPermissions {
		if p.isPermitted(path) {
			return true
		}
	}
	for _, p := range x.Permissions {
		if p.isPermitted(path) {
			return true
		}
	}
	return false
}

// App collection with index
type xAppIndex struct {
	a        []*xApp
	idIndex  map[string]*xApp
	keyIndex map[uint32]*xApp
}

func newAppIndex(apps []*xApp) *xAppIndex {
	var (
		idIndex  = make(map[string]*xApp)
		keyIndex = make(map[uint32]*xApp)
	)
	for _, a := range apps {
		idIndex[a.Id] = a
		keyIndex[a.Key] = a
	}
	return &xAppIndex{
		a:        apps,
		idIndex:  idIndex,
		keyIndex: keyIndex,
	}
}

func findAppById(id string) *xApp {
	a, _ := xApps.idIndex[id]
	return a
}
func findAppByKey(key uint32) *xApp {
	a, _ := xApps.keyIndex[key]
	return a
}
func listApps() []*xApp {
	return xApps.a
}
