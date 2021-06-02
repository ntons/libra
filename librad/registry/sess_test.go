package registry

import (
	"fmt"
	"testing"

	"github.com/vmihailenco/msgpack/v4"
)

func TestSess(t *testing.T) {
	s := &xSess{
		Token: "xxx",
		Data: xSessData{
			RoleId: "123",
		},
	}
	b, _ := msgpack.Marshal(s)

	var v interface{}

	msgpack.Unmarshal(b, &v)
	fmt.Println(v)

}
