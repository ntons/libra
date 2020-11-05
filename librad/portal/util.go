package portal

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// 以下均为二进制长度，最终将会由base32(NoPadding)进行编码
	// appKey[4]+random[6]
	idLen = 10
	//
	tagLen = 2
	// aes(userId[10]+tag[2]+random[4])
	tokenLen = aes.BlockSize
	// aes(roleId[10]+tag[2]+random[4])
	ticketLen = aes.BlockSize
)

var (
	// token/ticket tag
	tag = []byte{'O', 'K'}
	// base32 encoding
	enc = base32.StdEncoding.WithPadding(base32.NoPadding)
	// aes iv
	iv = bytes.Repeat([]byte{0x0}, aes.BlockSize)
)

func newUserId(appKey uint32) string {
	b := make([]byte, idLen)
	binary.LittleEndian.PutUint32(b, appKey)
	if _, err := io.ReadFull(rand.Reader, b[4:]); err != nil {
		panic(err)
	}
	return enc.EncodeToString(b)
}
func newRoleId(appKey uint32) string {
	b := make([]byte, idLen)
	binary.LittleEndian.PutUint32(b, appKey)
	if _, err := io.ReadFull(rand.Reader, b[4:]); err != nil {
		panic(err)
	}
	return enc.EncodeToString(b)
}

func tokenKey(userId string) string {
	return fmt.Sprintf("token:{%s}", userId)
}
func newToken(app *dbApp, userId string) string {
	b := make([]byte, tokenLen)
	if _, err := enc.Decode(b, []byte(userId)); err != nil {
		panic(errMalformedUserId)
	}
	copy(b[idLen:], tag)
	io.ReadFull(rand.Reader, b[idLen+tagLen:])
	cipher.NewCBCEncrypter(app.block, iv).CryptBlocks(b, b)
	return enc.EncodeToString(b)
}
func decToken(app *dbApp, token string) (userId string, err error) {
	b, err := enc.DecodeString(token)
	if err != nil || len(b) != tokenLen {
		return "", errInvalidToken
	}
	cipher.NewCBCDecrypter(app.block, iv).CryptBlocks(b, b)
	if !bytes.Equal(b[idLen:idLen+tagLen], tag) {
		return "", errInvalidToken
	}
	return enc.EncodeToString(b[:idLen]), nil
}

func ticketKey(roleId string) string {
	return fmt.Sprintf("ticket:{%s}", roleId)
}
func newTicket(app *dbApp, roleId string) (ticket string) {
	b := make([]byte, ticketLen)
	if _, err := enc.Decode(b, []byte(roleId)); err != nil {
		panic(errMalformedRoleId)
	}
	copy(b[idLen:], tag)
	io.ReadFull(rand.Reader, b[idLen+tagLen:])
	cipher.NewCBCEncrypter(app.block, iv).CryptBlocks(b, b)
	return enc.EncodeToString(b)
}
func decTicket(app *dbApp, ticket string) (roleId string, err error) {
	b, err := enc.DecodeString(ticket)
	if err != nil || len(b) != ticketLen {
		return "", errInvalidTicket
	}
	cipher.NewCBCDecrypter(app.block, iv).CryptBlocks(b, b)
	if !bytes.Equal(b[idLen:idLen+tagLen], tag) {
		return "", errInvalidTicket
	}
	return enc.EncodeToString(b[:idLen]), nil
}
