package portal

import (
	"crypto/aes"
	"fmt"
	"sync/atomic"
	"unsafe"

	"math/rand"
	"testing"
)

func TestIdentity(t *testing.T) {
	for i := 0; i < 10000; i++ {
		aesKey := make([]byte, aes.BlockSize)
		rand.Read(aesKey)
		block, _ := aes.NewCipher(aesKey)
		app := &xApp{
			Id:    fmt.Sprintf("%x", aesKey[:8]),
			Key:   rand.Uint32(),
			block: block,
		}
		atomic.StorePointer(&apps, unsafe.Pointer(newAppMgr([]*xApp{app})))

		userId := newUserId(app.Key)
		if len(userId) != rawIdLen*8/5 {
			t.Errorf("unexpected user id length")
		}

		roleId := newRoleId(app.Key)
		if len(roleId) != rawIdLen*8/5 {
			t.Errorf("unexpected role id length")
		}

		token, err := genToken(app, userId)
		if err != nil {
			t.Errorf("failed to generate token: %v", err)
		}
		if len(token) != rawTokenLen*4/3 {
			t.Errorf("unexpected token length: %d", len(token))
		}

		ticket, err := genTicket(app, userId, roleId)
		if err != nil {
			t.Errorf("failed to generate ticket: %v", err)
		}
		if len(ticket) != rawTicketLen*4/3 {
			t.Errorf("unexpected ticket length: %d", len(ticket))
		}

		if _appId, _userId, err := decToken(token); err != nil {
			t.Errorf("failed to decode token: %s", err)
		} else if _appId != app.Id {
			t.Errorf("unexpected appId decoded from token: %s,%s", _appId, app.Id)
		} else if _userId != userId {
			t.Errorf("unexpected userId decoded from token: %s,%s", _userId, userId)
		}

		if _appId, _userId, _roleId, err := decTicket(ticket); err != nil {
			t.Errorf("failed to decode ticket: %s", err)
		} else if _appId != app.Id {
			t.Errorf("unexpected appId decoded from ticket: %s,%s", _appId, app.Id)
		} else if _userId != userId {
			t.Errorf("unexpected userId decoded from ticket: %s,%s", _userId, userId)
		} else if _roleId != roleId {
			t.Errorf("unexpected roleId decoded from ticket: %s,%s", _roleId, roleId)
		}
	}
}
