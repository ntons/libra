package comm

import (
	"math/rand"
	"reflect"
	"strings"
	"unsafe"
)

func B2S(b []byte) (s string) {
	return *(*string)(unsafe.Pointer(&b))
}

func S2B(s string) (b []byte) {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	return
}

const (
	Digits    = "0123456789"
	LowerCase = "abcdefghijklmnopqrstuvwxyz"
	UpperCase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// random n byte string, not used for security!!!
func RandomString(n int, categories ...string) string {
	S := strings.Join(categories, "")
	if len(S) == 0 {
		S = Digits + LowerCase
	}
	N := len(S)

	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(S[rand.Intn(N)])
	}
	return sb.String()
}
