package portal

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/sigurn/crc8"
	"google.golang.org/grpc/metadata"
)

// 对象身份(Id)和身份凭证(Cred)
// 需要保证
// * 身份凭证无法被伪造
// * 一个身份凭证只能唯一对应一个身份
// * 一个对象同时只能有一个身份凭证
// 需要注意
// * Cred需要加密，提前进行一次校验，避免遭暴力破解时累及缓存。
// * 最终输出的运算结果需经过base32编码的，为了空间效率，长度为5的倍数。
// * 最终输出的运算结果需经过base64编码的，为了空间效率，长度为3的倍数。

// userId{10}: appKey{4}+rand{5}+rand{1}&0xF0|0x1
// roleId{10}: appKey{4}+rand{5}+rand{1}&0xF0|0x2
// token{18}:  ivSeed{1}+appKey{4}+aes(userId[4:]{6}+rand{6}+CRC8)
// ticket{18}: ivSeed{1}+appKey{4}+aes(roleId[4:]{6}+rand{6}+CRC8)

const (
	rawIdLen   = 10
	rawCredLen = 18
)

var (
	table = crc8.MakeTable(crc8.CRC8_DVB_S2)
)

// generate id
func newId(appKey uint32, tag uint8) string {
	id := make([]byte, rawIdLen)
	binary.BigEndian.PutUint32(id, appKey)
	io.ReadFull(rand.Reader, id[4:])
	id[rawIdLen-1] = id[rawIdLen-1]&0xF0 | tag
	return base32.StdEncoding.EncodeToString(id)
}
func newUserId(appKey uint32) string { return newId(appKey, 0x1) }
func newRoleId(appKey uint32) string { return newId(appKey, 0x2) }

func genCred(app *xApp, id string) (string, error) {
	rawId, err := base32.StdEncoding.DecodeString(id)
	if err != nil {
		return "", fmt.Errorf("failed to decode id: %w", err)
	}
	if len(rawId) != rawIdLen {
		return "", fmt.Errorf("bad raw role id length: %d", len(rawId))
	}
	if app.Key != binary.BigEndian.Uint32(rawId[:4]) {
		return "", fmt.Errorf("mismatched app key and id")
	}
	b := make([]byte, rawCredLen)
	io.ReadFull(rand.Reader, b[:1])
	copy(b[1:], rawId[:])
	io.ReadFull(rand.Reader, b[1+rawIdLen:])
	b[rawCredLen-1] = crc8.Checksum(b[:rawCredLen-1], table)
	stream := cipher.NewCTR(app.block, bytes.Repeat(b[:1], aes.BlockSize))
	stream.XORKeyStream(b[5:], b[5:])
	return base64.StdEncoding.EncodeToString(b), nil
}
func decCred(t string) (app *xApp, id string, err error) {
	b, err := base64.StdEncoding.DecodeString(t)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode cred: %w", err)
	}
	if len(b) != rawCredLen {
		return nil, "", fmt.Errorf("bad raw cred length: %d", len(b))
	}
	if app = getAppByKey(binary.BigEndian.Uint32(b[1:5])); app == nil {
		return nil, "", fmt.Errorf("bad app key")
	}
	stream := cipher.NewCTR(app.block, bytes.Repeat(b[:1], aes.BlockSize))
	stream.XORKeyStream(b[5:], b[5:])
	if b[rawCredLen-1] != crc8.Checksum(b[:rawCredLen-1], table) {
		return nil, "", fmt.Errorf("bad checksum")
	}
	rawId := make([]byte, rawIdLen)
	copy(rawId, b[1:1+rawIdLen])
	return app, base32.StdEncoding.EncodeToString(rawId), nil
}

// fetch session from context, session must exist at login required methods
func getSessionFromContext(ctx context.Context) (appId, userId string, ok bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
	if v := md.Get(xLibraTrustedAppId); len(v) != 1 {
		return "", "", false
	} else {
		appId = v[0]
	}
	if v := md.Get(xLibraTrustedUserId); len(v) != 1 {
		return "", "", false
	} else {
		userId = v[0]
	}
	return
}
