package api

import (
	"encoding/json"
	"testing"
)

func TestAuditSessionIDPreservesClientUint64(t *testing.T) {
	// RustDesk 1.4.9 sends LoginRequest.session_id as a JSON uint64.
	// Adjacent IDs above 2^53 must not collapse through float64 rounding.
	for _, id := range []string{"0", "9007199254740992", "9007199254740993", "18446744073709551615"} {
		var form AuditConnForm
		if err := json.Unmarshal([]byte(`{"session_id":`+id+`}`), &form); err != nil {
			t.Fatal(err)
		}
		if got := form.ToAuditConn().SessionId; got != id {
			t.Errorf("session_id %s was stored as %s", id, got)
		}
	}
	for _, id := range []string{"-1", "1.5", "18446744073709551616"} {
		var form AuditConnForm
		if err := json.Unmarshal([]byte(`{"session_id":`+id+`}`), &form); err == nil {
			t.Errorf("accepted invalid client uint64 %s", id)
		}
	}
}
