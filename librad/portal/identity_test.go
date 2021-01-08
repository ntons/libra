package portal

import (
	"bytes"
	"crypto/aes"

	"math/rand"
	"testing"
)

func TestIdentity(t *testing.T) {
	for i := 0; i < 1000; i++ {
		appKey := rand.Uint32()
		idTag := uint8(rand.Int()%2 + 1)
		id := newId(appKey, idTag)
		if xIdLen*8/5 != len(encId(id)) {
			t.Errorf("unexpected id length")
		}
		aesKey := make([]byte, aes.BlockSize)
		rand.Read(aesKey)
		block, _ := aes.NewCipher(aesKey)
		cred := newCred(id, block)
		if xCredLen*4/3 != len(encCred(cred)) {
			t.Errorf("unexpected cred length")
		}
		if v := getAppKeyFromCred(cred); v != appKey {
			t.Errorf("unexpected app key mismatch")
		}
		if v, ok := getIdFromCred(cred, block); !ok {
			t.Errorf("unexpected cred checksum")
		} else if !bytes.Equal(id[:], v[:]) {
			t.Errorf("unexpected cred mismatch")
		}
	}
}
