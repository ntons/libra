package db

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type App struct {
	// 应用ID
	Id string `bson:"_id"`
	// 数值形式的应用ID
	Key uint32 `bson:"key"`
	// 应用签名密钥，授权访问
	Secret string `bson:"secret,omitempty"`
	// 应用指纹，特异化应用数据，增加安全性
	Fingerprint string `bson:"fingerprint,omitempty"`
	// 允许的服务
	Permissions []*Permission `bson:"permissions,omitempty"`
	// AES密钥，由Fingerprint生成
	block cipher.Block
}

func (x *App) parse() (err error) {
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
func (x *App) IsPermitted(path string) bool {
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
	a        []*App
	idIndex  map[string]*App
	keyIndex map[uint32]*App
}

func newAppIndex(apps []*App) *xAppIndex {
	var (
		idIndex  = make(map[string]*App)
		keyIndex = make(map[uint32]*App)
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

func FindAppById(id string) *App {
	a, _ := xApps.idIndex[id]
	return a
}
func findAppByKey(key uint32) *App {
	a, _ := xApps.keyIndex[key]
	return a
}
func ListApps() []*App {
	return xApps.a
}
func loadApps(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection, err := getAppCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get app collection: %w", err)
	}
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to query apps: %w", err)
	}
	var res []*App
	if err = cursor.All(ctx, &res); err != nil {
		return
	}
	for _, a := range res {
		if err = a.parse(); err != nil {
			return
		}
	}
	xApps = newAppIndex(res)
	appWatcher.trigger(res)
	return
}
