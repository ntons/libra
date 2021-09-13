package registry

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/sigurn/crc16"
)

// 对象身份(Id)和身份凭证(Token)
// 需要保证
// * 身份凭证无法被伪造
// * 一个身份凭证只能唯一对应一个身份
// * 一个对象同时只能有一个身份凭证
// 需要注意
// * Token需要加密，提前进行一次校验，避免遭暴力破解时累及缓存。
// * 最终输出的运算结果需经过base32编码的，为了空间效率，长度为5的倍数。
// * 最终输出的运算结果需经过base64编码的，为了空间效率，长度为3的倍数。

// userId{10}: appKey{4}+rand{5}+rand{1}&0xF0|0x1
// roleId{10}: appKey{4}+rand{5}+rand{1}&0xF0|0x2

const (
	rawIdLen    = 10
	rawTokenLen = 18
)

// generate id
func newId(appKey uint32, tag uint8) string {
	b := make([]byte, rawIdLen)
	binary.BigEndian.PutUint32(b, appKey)
	io.ReadFull(rand.Reader, b[4:])
	b[rawIdLen-1] = b[rawIdLen-1]&0xF0 | tag
	return base32.StdEncoding.EncodeToString(b)
}
func decId(id string) (appKey uint32, tag uint8, err error) {
	b, err := base32.StdEncoding.DecodeString(id)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid id")
	}
	appKey = binary.BigEndian.Uint32(b)
	tag = b[rawIdLen-1]
	return
}

func newUserId(appKey uint32) string { return newId(appKey, 0x1) }
func newRoleId(appKey uint32) string { return newId(appKey, 0x2) }

func newToken(app *xApp, id string) (string, error) {
	raw, err := newTokenV1(app, id)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func decToken(t string) (app *xApp, id string, err error) {
	raw, err := base64.StdEncoding.DecodeString(t)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode cred: %w", err)
	}
	switch raw[0] {
	case 0x1:
		return decTokenV1(raw)
	default:
		return nil, "", fmt.Errorf("bad cred version")
	}
}

// TokenV1{24}: 0x1+ivSeed{3}+appKey{4}+aes(rawId[4:]{6}+rand{8}+crc16)
var t16 = crc16.MakeTable(crc16.CRC16_XMODEM)

func newTokenV1(app *xApp, id string) (rawToken []byte, err error) {
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
	rawToken = make([]byte, 24)
	rawToken[0] = 0x1
	io.ReadFull(rand.Reader, rawToken[1:4])
	copy(rawToken[4:], rawId[:])
	io.ReadFull(rand.Reader, rawToken[14:22])
	binary.BigEndian.PutUint16(rawToken[22:], crc16.Checksum(rawToken[:22], t16))
	iv := bytes.Repeat(rawToken[:4], aes.BlockSize/4)
	cipher.NewCBCEncrypter(app.block, iv).CryptBlocks(rawToken[8:], rawToken[8:])
	return rawToken, nil
}

func decTokenV1(rawToken []byte) (app *xApp, id string, err error) {
	if len(rawToken) != 24 {
		err = fmt.Errorf("bad rawToken cred length: %d", len(rawToken))
		return
	}
	if app = findAppByKey(
		binary.BigEndian.Uint32(rawToken[4:8])); app == nil {
		err = fmt.Errorf("bad app key")
		return
	}
	iv := bytes.Repeat(rawToken[:4], aes.BlockSize/4)
	cipher.NewCBCDecrypter(app.block, iv).CryptBlocks(rawToken[8:], rawToken[8:])
	if binary.BigEndian.Uint16(rawToken[22:]) !=
		crc16.Checksum(rawToken[:22], t16) {
		err = fmt.Errorf("bad checksum")
		return
	}
	rawId := make([]byte, rawIdLen)
	copy(rawId, rawToken[4:4+rawIdLen])
	return app, base32.StdEncoding.EncodeToString(rawId), nil
}
