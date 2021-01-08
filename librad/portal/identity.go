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
	"io"
	"path"
	"sort"

	log "github.com/ntons/log-go"
	"github.com/sigurn/crc8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/ntons/libra/librad/comm"
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

// userId{10}: appKey{4}+random{5}+random{1}&0xF0|0x1
// roleId{10}: appKey{4}+random{5}+random{1}&0xF0|0x2
// token{18}:  ivSeed{1}+appKey{4}+aes(userId[4:]{6}+random{6}+CRC8)
// ticket{18}: ivSeed{1}+appKey{4}+aes(userId[4:]{6}+random{6}+CRC8)

const (
	xIdLen   = 10
	xCredLen = 18
)

type xId [xIdLen]byte
type xCred [xCredLen]byte

var (
	table = crc8.MakeTable(crc8.CRC8_DVB_S2)
	b32   = base32.StdEncoding
	b64   = base64.StdEncoding
)

// use base32 encoding id for readability
func encId(id xId) string {
	return b32.EncodeToString(id[:])
}
func decId(s string) (id xId, ok bool) {
	n, err := b32.Decode(id[:], comm.S2B(s))
	return id, err == nil && n == xIdLen
}

func encCred(cred xCred) string {
	return b64.EncodeToString(cred[:])
}
func decCred(s string) (cred xCred, ok bool) {
	n, err := b64.Decode(cred[:], comm.S2B(s))
	return cred, err == nil && n == xCredLen
}

// generate id
func newId(appKey uint32, tag uint8) (id xId) {
	binary.BigEndian.PutUint32(id[:], appKey)
	io.ReadFull(rand.Reader, id[4:])
	id[xIdLen-1] = id[xIdLen-1]&0xF0 | tag
	return
}
func newUserId(appKey uint32) string {
	return encId(newId(appKey, 0x1))
}
func newRoleId(appKey uint32) string {
	return encId(newId(appKey, 0x2))
}

// generate cred
func newCred(id xId, block cipher.Block) (cred xCred) {
	io.ReadFull(rand.Reader, cred[:])
	copy(cred[1:1+xIdLen], id[:])
	cred[xCredLen-1] = crc8.Checksum(cred[:xCredLen-1], table)
	iv := bytes.Repeat(cred[:1], aes.BlockSize)
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(cred[5:], cred[5:])
	return
}

// extract appKey from cred
func getAppKeyFromCred(cred xCred) (appKey uint32) {
	return binary.BigEndian.Uint32(cred[1:5])
}

// decode cred
func getIdFromCred(cred xCred, block cipher.Block) (id xId, ok bool) {
	iv := bytes.Repeat(cred[:1], aes.BlockSize)
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(cred[5:], cred[5:])
	if cred[xCredLen-1] != crc8.Checksum(cred[:xCredLen-1], table) {
		return id, false
	}
	copy(id[:], cred[1:1+xIdLen])
	return id, true
}

// token associated sess
type xSess struct{ appId, userId string }
type xSessKey struct{}

func (sess *xSess) getAppId() string {
	if sess == nil {
		return ""
	}
	return sess.appId
}
func (sess *xSess) getUserId() string {
	if sess == nil {
		return ""
	}
	return sess.userId
}

// login state required interceptor
type tokenRequired struct {
	exclusion []string
}

func newTokenRequired(exclusion ...string) *tokenRequired {
	sort.Strings(exclusion)
	return &tokenRequired{exclusion: exclusion}
}

func (lr *tokenRequired) InterceptUnary(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp interface{}, err error) {
	log.Debug(info.FullMethod)
	if i := sort.SearchStrings(
		lr.exclusion, path.Base(info.FullMethod)); i < len(lr.exclusion) {
		return handler(ctx, req)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get(xLibraToken)) == 0 {
		return nil, errInvalidToken
	}
	appId, userId, err := db.checkToken(ctx, md.Get(xLibraToken)[0])
	if err != nil {
		return
	}
	sess := &xSess{appId: appId, userId: userId}
	return handler(context.WithValue(ctx, xSessKey{}, sess), req)
}

// fetch sess from context
// sess must exist in login required methods
func getSessFromContext(ctx context.Context) *xSess {
	sess, _ := ctx.Value(xSessKey{}).(*xSess)
	return sess
}
func (lr *tokenRequired) getSessAppIdFromContext(ctx context.Context) string {
	return getSessFromContext(ctx).getAppId()
}
func (lr *tokenRequired) getUserIdFromContext(ctx context.Context) string {
	return getSessFromContext(ctx).getUserId()
}
