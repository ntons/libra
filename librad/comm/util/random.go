package util

import (
	"math/rand"
	"strings"
)

const (
	LowerCase = "abcdefghijklmnopqrstuvwxyz"
	UpperCase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Letters   = LowerCase + UpperCase
	Digits    = "0123456789"
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
