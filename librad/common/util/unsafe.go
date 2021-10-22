package util

import (
	"reflect"
	"unsafe"
)

// If you know for sure that the byte slice won't be mutated,
// you won't get bounds (or GC) issues with the above conversions.
func BytesToString(buf []byte) (str string) {
	return *(*string)(unsafe.Pointer(&buf))
}
func StringToBytes(str string) (buf []byte) {
	*(*string)(unsafe.Pointer(&buf)) = str
	(*reflect.SliceHeader)(unsafe.Pointer(&buf)).Cap = len(str)
	return
}
