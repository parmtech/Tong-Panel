package auth

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPasswordVerifyBudgetLimits(t *testing.T) {
	for _, limit := range []int{-1, 0, 5, 100} {
		if b, err := NewPasswordVerifyBudget(limit); err == nil || b != nil {
			t.Fatalf("accepted invalid concurrency limit %d", limit)
		}
	}
	for _, limit := range []int{1, 2, 4} {
		b, err := NewPasswordVerifyBudget(limit)
		if err != nil || cap(b.slots) != limit {
			t.Fatalf("limit %d: %+v err=%v", limit, b, err)
		}
	}
	var nilBudget *PasswordVerifyBudget
	if accepted, err := nilBudget.Verify(nil, ""); accepted || !errors.Is(err, ErrPasswordVerifierBusy) {
		t.Fatalf("nil verifier accepted: %v %v", accepted, err)
	}
}

func TestPasswordVerifyBudgetRejectsOverloadAndReleasesSlot(t *testing.T) {
	b, err := NewPasswordVerifyBudget(1)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	b.verify = func(password []byte, encoded string) bool {
		if string(password) != "long-password" || encoded != "opaque-hash" {
			t.Errorf("verification mutated inputs")
		}
		close(entered)
		<-release
		return true
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var verified bool
	var verifyErr error
	go func() {
		defer wg.Done()
		verified, verifyErr = b.Verify([]byte("long-password"), "opaque-hash")
	}()
	<-entered
	if ok, err := b.Verify([]byte("wrong"), "wrong-hash"); ok || !errors.Is(err, ErrPasswordVerifierBusy) {
		t.Fatalf("overloaded password verification not rejected: %v %v", ok, err)
	}
	close(release)
	wg.Wait()
	if !verified || verifyErr != nil {
		t.Fatalf("admitted verification failed: %v %v", verified, verifyErr)
	}
	if ok, err := b.Verify([]byte("long-password"), "opaque-hash"); !ok || err != nil {
		t.Fatalf("slot not released after verification: %v %v", ok, err)
	}
}

func TestPasswordVerifyBudgetAdmitsAtMostConfiguredCapacity(t *testing.T) {
	const slots = 2
	b, _ := NewPasswordVerifyBudget(slots)
	block := make(chan struct{})
	started := make(chan struct{}, slots)
	var active int32
	var peak int32
	b.verify = func([]byte, string) bool {
		n := atomic.AddInt32(&active, 1)
		for {
			old := atomic.LoadInt32(&peak)
			if n <= old || atomic.CompareAndSwapInt32(&peak, old, n) {
				break
			}
		}
		started <- struct{}{}
		<-block
		atomic.AddInt32(&active, -1)
		return true
	}
	var wg sync.WaitGroup
	for i := 0; i < slots; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, err := b.Verify(nil, ""); !ok || err != nil {
				t.Errorf("verification failed: %v %v", ok, err)
			}
		}()
	}
	<-started
	<-started
	// More callers are rejected without scheduling more expensive work.
	for i := 0; i < 50; i++ {
		if ok, err := b.Verify(nil, ""); ok || !errors.Is(err, ErrPasswordVerifierBusy) {
			t.Fatalf("unexpected verification during capacity exhaustion: %v %v", ok, err)
		}
	}
	close(block)
	wg.Wait()
	if atomic.LoadInt32(&peak) != slots || atomic.LoadInt32(&active) != 0 {
		t.Fatalf("capacity constraint broken: peak=%d active=%d", peak, active)
	}
}

func TestPasswordVerifyBudgetReturnsRejectedCredentialsWithoutError(t *testing.T) {
	b, _ := NewPasswordVerifyBudget(1)
	b.verify = func([]byte, string) bool { return false }
	if accepted, err := b.Verify([]byte("incorrect"), "malformed"); accepted || err != nil {
		t.Fatalf("rejected password should not be confused with overload: %v %v", accepted, err)
	}
}
