package database

import (
	"fmt"

	"github.com/pierrec/lz4/v4"
)

// 允许的最大存档
const lz4MaxLen = 160 * 1024

type lz4Filter struct{}

func (f lz4Filter) Encode(in []byte) ([]byte, bool, error) {
	if len(in) > lz4MaxLen {
		return nil, false, fmt.Errorf("too large")
	}
	buf := make([]byte, len(in))
	var c lz4.Compressor
	n, err := c.CompressBlock(in, buf)
	if err != nil {
		return nil, false, err
	}
	if n == 0 {
		return nil, true, nil
	}
	return buf[:n], false, nil
}
func (f lz4Filter) Decode(in []byte) ([]byte, error) {
	var buf [lz4MaxLen]byte
	n, err := lz4.UncompressBlock(in, buf[:])
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
