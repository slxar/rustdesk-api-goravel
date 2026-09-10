package service

import (
	"crypto/sha256"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebAccessSingleUseAndExpiry(t *testing.T) {
	store := &WebAccessStore{}
	token, err := store.Issue("123456789", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var success atomic.Int32
	var wg sync.WaitGroup
	var cookie string
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, a, err := store.Redeem(token)
			if err == nil {
				success.Add(1)
				cookie = c
				if a.PeerID != "123456789" {
					t.Error("target changed")
				}
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("%d redemptions", success.Load())
	}
	if store.Session(cookie) == nil || store.Session(token) != nil {
		t.Fatal("link/session separation failed")
	}
	a := store.Session(cookie)
	if !a.AllowRelay("relay-uuid-123456789") || !a.ClaimRelay("relay-uuid-123456789") || a.ClaimRelay("relay-uuid-123456789") || a.ClaimRelay("another-uuid-12345") {
		t.Fatal("relay is not single-use")
	}
	store.Revoke(cookie)
	if store.Session(cookie) != nil {
		t.Fatal("revocation failed")
	}
	expired, _ := store.Issue("123456789", 1, time.Minute)
	store.links[sha256.Sum256([]byte(expired))].Expires = time.Now().Add(-time.Second)
	if _, _, err := store.Redeem(expired); err == nil {
		t.Fatal("expired link redeemed")
	}
	for _, id := range []string{"", "12345", "123456@other-server", "../123456"} {
		if _, err := store.Issue(id, 1, time.Minute); err == nil {
			t.Errorf("accepted invalid id %q", id)
		}
	}
}
