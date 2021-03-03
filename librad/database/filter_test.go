package database

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func RandomBytes() []byte {
	b := make([]byte, rand.Intn(1000)+1)
	if n, err := rand.Read(b); err != nil || n != len(b) {
		panic("failed to rand")
	}
	return bytes.Repeat(b, rand.Intn(lz4MaxLen/len(b)-1)+1)
}
func TestChain(t *testing.T) {
	for i := 0; i < 10000; i++ {
		b1 := RandomBytes()
		b2, err := encode(b1)
		if err != nil {
			t.Fatal("failed to encode: ", err)
		}
		b3, err := decode(b2)
		if err != nil {
			t.Fatal("failed to decode: ", err)
		}
		if !bytes.Equal(b1, b3) {
			t.Fatal("not equal: ", i, len(b1), len(b2), len(b3))
		}
	}
}

func TestLz4(t *testing.T) {
	filter := lz4Filter{}
	n, N := 0, 10000
	for i := 0; i < N; i++ {
		b1 := RandomBytes()
		b2, skip, err := filter.Encode(b1)
		if err != nil {

			t.Fatal("failed to encode: ", err)
		}
		if skip {
			n++
			continue
		}
		b3, err := filter.Decode(b2)
		if err != nil {
			t.Fatal("failed to decode: ", err)
		}
		if !bytes.Equal(b1, b3) {
			t.Fatal("not equal: ", i, len(b1), len(b2), len(b3))
		}
	}
	fmt.Printf("skip: %d/%d\n", n, N)
}
