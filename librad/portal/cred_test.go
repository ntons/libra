package portal

import (
	"math/rand"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/ntons/libra/librad/internal/util"
)

func randTestApp() *xApp {
	return &xApp{
		Id:          util.RandomString(16, util.Letters+util.Digits),
		Key:         rand.Uint32(),
		Secret:      util.RandomString(32, util.Letters+util.Digits),
		Fingerprint: util.RandomString(32, util.Letters+util.Digits),
	}
}

func TestCredV1(t *testing.T) {
	a := randTestApp()
	atomic.StorePointer(&apps, unsafe.Pointer(newAppMgr([]*xApp{a})))

	id := newUserId(a.Key)
	token, err := genCredV1(a, id)
	if err != nil {
		t.Fatalf("failed to generate cred: %v", err)
	}

	if _a, _id, err := decCredV1(token); err != nil {
		t.Fatalf("failed to decrypt token: %v", err)
	} else if _a != a {
		t.Fatalf("bad decrypted app: %s", _a.Id)
	} else if _id != id {
		t.Fatalf("bad decrypted user id: %s", id)
	}
}

func BenchmarkGenCredV1(b *testing.B) {
	a := randTestApp()
	atomic.StorePointer(&apps, unsafe.Pointer(newAppMgr([]*xApp{a})))

	id := newUserId(a.Key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := genCredV1(a, id); err != nil {
			b.Fatalf("failed to generate cred: %v", err)
		}
	}
}
func BenchmarkDecCredV1(b *testing.B) {
	a := randTestApp()
	atomic.StorePointer(&apps, unsafe.Pointer(newAppMgr([]*xApp{a})))

	id := newUserId(a.Key)

	tokens := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		if token, err := genCredV1(a, id); err != nil {
			b.Fatalf("failed to generate cred: %v", err)
		} else {
			tokens[i] = token
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _a, _id, err := decCredV1(tokens[i]); err != nil {
			b.Fatalf("failed to decrypt cred: %v", err)
		} else if _a != a || _id != id {
			b.Fatalf("bad decrypted data: %s, %s", _a.Id, _id)
		}
	}
}
