package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/parmtech/tong-panel/internal/jobs"
)

func TestJobCreateEnforcesASCIIIdempotencyKeyByteBounds(t *testing.T) {
	server := setup(t)
	h := server.Handler()
	cases := []struct {
		name string
		key  string
		want bool
	}{
		{name: "below minimum", key: strings.Repeat("a", 7)},
		{name: "minimum", key: strings.Repeat("a", 8), want: true},
		{name: "maximum", key: strings.Repeat("b", 128), want: true},
		{name: "above maximum", key: strings.Repeat("c", 129)},
		{name: "non-ASCII at minimum byte length", key: strings.Repeat("é", 4)},
	}

	wantJobs := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(h, http.MethodPost, "/api/v1/jobs", secret, tc.key, `{"kind":"validation.noop"}`)
			if tc.want {
				if rec.Code != http.StatusAccepted {
					t.Fatalf("create status=%d body=%s; want 202", rec.Code, rec.Body.String())
				}
				var created jobs.Job
				if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.IdempotencyKey != tc.key {
					t.Fatalf("created key=%q err=%v; want exact key of %d bytes", created.IdempotencyKey, err, len(tc.key))
				}
				wantJobs++
			} else {
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("invalid key status=%d body=%s; want 400", rec.Code, rec.Body.String())
				}
				var failure struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &failure); err != nil || failure.Error.Code != "invalid_job" {
					t.Fatalf("invalid key response=%s error=%q err=%v; want invalid_job", rec.Body.String(), failure.Error.Code, err)
				}
			}
			if got := len(server.Jobs.List("tenant-A")); got != wantJobs {
				t.Fatalf("job count=%d after key length=%d; want %d", got, len(tc.key), wantJobs)
			}
		})
	}
}
