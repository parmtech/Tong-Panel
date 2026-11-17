package auth

import (
	"errors"
	"testing"
)

func TestPasswordHashSharesVerificationAdmission(t *testing.T) {
	budget, _ := NewPasswordVerifyBudget(1)
	budget.hash = func(password []byte) (string, error) {
		if string(password) != "exact password bytes" {
			t.Fatal("hash input changed")
		}
		if _, err := budget.Verify(nil, ""); !errors.Is(err, ErrPasswordVerifierBusy) {
			t.Fatal("hash did not occupy verification slot")
		}
		return "fixture", nil
	}
	budget.verify = func([]byte, string) bool {
		if _, err := budget.Hash(nil); !errors.Is(err, ErrPasswordVerifierBusy) {
			t.Fatal("verification did not occupy hash slot")
		}
		return true
	}
	if hash, err := budget.Hash([]byte("exact password bytes")); err != nil || hash != "fixture" {
		t.Fatal("admitted hash failed")
	}
	if ok, err := budget.Verify(nil, ""); err != nil || !ok {
		t.Fatal("hash slot was not released")
	}
}

func TestPasswordHashBudgetReleasesOnErrorAndPanic(t *testing.T) {
	budget, _ := NewPasswordVerifyBudget(1)
	failure := errors.New("fixture failure")
	budget.hash = func([]byte) (string, error) { return "", failure }
	if _, err := budget.Hash(nil); !errors.Is(err, failure) {
		t.Fatal("hash error lost")
	}
	budget.hash = func([]byte) (string, error) { panic("fixture panic") }
	func() {
		defer func() {
			if recover() == nil {
				t.Error("fixture panic missing")
			}
		}()
		_, _ = budget.Hash(nil)
	}()
	if len(budget.slots) != 0 {
		t.Fatal("hash leaked admission slot")
	}
	var missing *PasswordVerifyBudget
	if _, err := missing.Hash(nil); !errors.Is(err, ErrPasswordVerifierBusy) {
		t.Fatal("nil hash budget accepted")
	}
}
