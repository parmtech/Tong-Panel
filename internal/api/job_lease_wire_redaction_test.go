package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/jobs"
)

func TestSQLiteJobAPIDoesNotExposeActiveWorkerLease(t *testing.T) {
	server, queue, path := setupSQLAPI(t)
	h := server.Handler()
	created := call(h, "POST", "/api/v1/jobs", secret, "lease-redaction-key-0001", `{"kind":"validation.noop"}`)
	job := assertCreated(t, created.Code, created.Body.Bytes())

	claim, err := queue.ClaimNext("lease-redaction-worker", time.Minute)
	if err != nil {
		t.Fatalf("claim synthetic job: %v", err)
	}
	if claim.Job.ID != job.ID {
		t.Fatalf("claim returned job %q, want %q", claim.Job.ID, job.ID)
	}
	if _, err := queue.Start(claim); err != nil {
		t.Fatalf("start synthetic job: %v", err)
	}

	// Confirm the local fixture is actually running with the expected worker
	// and persisted token digest before checking the public API responses.
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	var state, owner string
	var storedHash []byte
	if err := db.QueryRow(`SELECT state,lease_owner,lease_token_sha256 FROM jobs WHERE id=?`, job.ID).Scan(&state, &owner, &storedHash); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	tokenBytes, err := hex.DecodeString(claim.Token)
	if err != nil {
		t.Fatalf("decode synthetic lease token: %v", err)
	}
	wantHash := sha256.Sum256(tokenBytes)
	if state != jobs.Running || owner != claim.WorkerID || !bytes.Equal(storedHash, wantHash[:]) {
		t.Fatalf("synthetic lease fixture mismatch: running=%t ownerMatches=%t hashMatches=%t",
			state == jobs.Running, owner == claim.WorkerID, bytes.Equal(storedHash, wantHash[:]))
	}

	for _, endpoint := range []struct {
		name, path string
	}{
		{name: "detail", path: "/api/v1/jobs/" + job.ID},
		{name: "legacy list", path: "/api/v1/jobs"},
		{name: "paged list", path: "/api/v1/jobs?limit=10"},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			response := call(h, "GET", endpoint.path, secret, "", "")
			if response.Code != 200 {
				t.Fatalf("GET %s returned %d", endpoint.path, response.Code)
			}
			body := response.Body.String()
			if !strings.Contains(body, `"state":"running"`) {
				t.Fatal("active job state is missing from API response")
			}
			for _, privateValue := range []struct{ name, value string }{
				{name: "worker lease token", value: claim.Token},
				{name: "worker lease token digest", value: hex.EncodeToString(wantHash[:])},
				{name: "worker ID", value: claim.WorkerID},
				{name: "lease_owner field", value: "lease_owner"},
				{name: "lease_until field", value: "lease_until"},
				{name: "lease_token_sha256 field", value: "lease_token_sha256"},
				{name: "leaseToken field", value: "leaseToken"},
			} {
				if strings.Contains(body, privateValue.value) {
					t.Fatalf("job API exposed %s", privateValue.name)
				}
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			jobFields := []map[string]json.RawMessage{envelope}
			if endpoint.name != "detail" {
				var page struct {
					Jobs []map[string]json.RawMessage `json:"jobs"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || len(page.Jobs) != 1 {
					t.Fatalf("decode one-job page: jobs=%d err=%v", len(page.Jobs), err)
				}
				jobFields = page.Jobs
			}
			allowed := map[string]bool{
				"id": true, "tenant_id": true, "requested_by": true, "kind": true,
				"state": true, "idempotency_key": true, "created_at": true, "updated_at": true,
			}
			for _, fields := range jobFields {
				for key := range fields {
					if !allowed[key] {
						t.Errorf("%s response added undeclared Job property %q", endpoint.name, key)
					}
				}
			}
		})
	}
}
