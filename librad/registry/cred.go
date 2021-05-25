package registry

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

	"github.com/sigurn/crc16"
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

const (
	rawIdLen   = 10
	rawCredLen = 18
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
	raw, err := genCredV1(app, id)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
func decCred(t string) (app *xApp, id string, err error) {
	raw, err := base64.StdEncoding.DecodeString(t)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode cred: %w", err)
	}
	switch raw[0] {
	case 0x1:
		return decCredV1(raw)
	default:
		return nil, "", fmt.Errorf("bad cred version")
	}
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

// CredV1{24}: 0x1+ivSeed{3}+appKey{4}+aes(rawId[4:]{6}+rand{8}+crc16)
var t16 = crc16.MakeTable(crc16.CRC16_XMODEM)

func genCredV1(app *xApp, id string) (rawCred []byte, err error) {
	rawId, err := base32.StdEncoding.DecodeString(id)
	if err != nil {
		return nil, fmt.Errorf("failed to decode id: %w", err)
	}
	if len(rawId) != rawIdLen {
		return nil, fmt.Errorf("bad raw role id length: %d", len(rawId))
	}
	if app.Key != binary.BigEndian.Uint32(rawId[:4]) {
		return nil, fmt.Errorf("mismatched app key and id")
	}
	rawCred = make([]byte, 24)
	rawCred[0] = 0x1
	io.ReadFull(rand.Reader, rawCred[1:4])
	copy(rawCred[4:], rawId[:])
	io.ReadFull(rand.Reader, rawCred[14:22])
	binary.BigEndian.PutUint16(rawCred[22:], crc16.Checksum(rawCred[:22], t16))
	iv := bytes.Repeat(rawCred[:4], aes.BlockSize/4)
	cipher.NewCBCEncrypter(app.block, iv).CryptBlocks(rawCred[8:], rawCred[8:])
	return rawCred, nil
}
func decCredV1(rawCred []byte) (app *xApp, id string, err error) {
	if len(rawCred) != 24 {
		return nil, "", fmt.Errorf("bad rawCred cred length: %d", len(rawCred))
	}
	if app = getAppByKey(binary.BigEndian.Uint32(rawCred[4:8])); app == nil {
		return nil, "", fmt.Errorf("bad app key")
	}
	iv := bytes.Repeat(rawCred[:4], aes.BlockSize/4)
	cipher.NewCBCDecrypter(app.block, iv).CryptBlocks(rawCred[8:], rawCred[8:])
	if binary.BigEndian.Uint16(rawCred[22:]) != crc16.Checksum(rawCred[:22], t16) {
		return nil, "", fmt.Errorf("bad checksum")
	}
	rawId := make([]byte, rawIdLen)
	copy(rawId, rawCred[4:4+rawIdLen])
	return app, base32.StdEncoding.EncodeToString(rawId), nil
}
