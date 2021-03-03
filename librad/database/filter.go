package database

import (
	"encoding/binary"
	"fmt"

	pb "google.golang.org/protobuf/proto"

	"github.com/ntons/libra/librad/comm/util"
)

// 数据过滤链
// 编码时候所有启用的过滤器都会执行，过滤器会给数据打自己的标记
// 解码时候要考虑旧版本数据的清清康，所有解码器都需要支持
// 已经上线的过滤器不可修改，只能禁用
// nop过滤器是编码起点和解码终点标记
// 这个声明是有顺序的，不要插入，不要删除！！！！
var chain = []enabler{
	/*0*/ enable(nopFilter{}), // nop filter must be enabled
	/*1*/ enable(lz4Filter{}),
}

type filter interface {
	// skip means input not suitable for this filter, skip it
	Encode(in []byte) (out []byte, skip bool, err error)
	Decode(in []byte) (out []byte, err error)
}

type enabler struct {
	filter
	enabled bool
}

func (x enabler) Enabled() bool { return x.enabled }

func enable(filter filter) enabler {
	return enabler{filter, true}
}
func disable(filter filter) enabler {
	return enabler{filter, false}
}

func encode(in []byte) ([]byte, error) {
	out := in
	for i, f := range chain {
		if !f.Enabled() {
			continue
		}
		t := make([]byte, binary.MaxVarintLen16)
		t = t[:binary.PutUvarint(t[:], uint64(i))]
		if b, skip, err := f.Encode(out); err != nil {
			return nil, err
		} else if skip {
			continue
		} else {
			out = append(t, b...)
		}
	}
	return out, nil
}
func decode(in []byte) ([]byte, error) {
	out := in
	for {
		i, n := binary.Uvarint(out)
		if i >= uint64(len(chain)) {
			return nil, fmt.Errorf("bad decoded tag %d", i)
		}
		var err error
		if out, err = chain[i].Decode(out[n:]); err != nil {
			return nil, err
		}
		if i == 0 {
			break
		}
	}
	return out, nil
}

func encodeMessage(m pb.Message) (string, error) {
	if b, err := pb.Marshal(m); err != nil {
		return "", err
	} else if b, err = encode(b); err != nil {
		return "", err
	} else {
		return util.BytesToString(b), nil
	}
}
func decodeMessage(s string, m pb.Message) error {
	if b, err := decode(util.StringToBytes(s)); err != nil {
		return err
	} else if pb.Unmarshal(b, m); err != nil {
		return err
	} else {
		return nil
	}
}
