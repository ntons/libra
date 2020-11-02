package misc

import (
	"reflect"
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
