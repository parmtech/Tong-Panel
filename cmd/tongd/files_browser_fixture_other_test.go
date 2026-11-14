//go:build !linux

package main

import (
	"github.com/parmtech/tong-panel/internal/auth"
	"testing"
)

func setupFilesBrowserFixture(t *testing.T, sessions *auth.SQLiteSessionStore) string { return "" }
