package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

const minimalContract = `{
 "openapi":"3.0.3","info":{"title":"synthetic test","version":"1"},
 "paths":{"/sample":{"post":{"operationId":"createSample",
 "requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["name"],"additionalProperties":false,"properties":{"name":{"type":"string"}}}}}},
 "responses":{
 "201":{"description":"Created","content":{"application/json":{"schema":{"type":"object","required":["ok"],"additionalProperties":false,"properties":{"ok":{"type":"boolean"}}}}}},
 "204":{"description":"No content"},
 "400":{"description":"Bad request","x-error-codes":["invalid_sample"],"content":{"application/json":{"schema":{"type":"object","required":["error"],"properties":{"error":{"type":"object","required":["code","message"],"properties":{"code":{"type":"string"},"message":{"type":"string"}}}}}}}}
 }}}}}
`

func validSample() contractCase {
	return contractCase{Name: "synthetic", OperationID: "createSample", Method: "POST", Path: "/sample", RequestHeaders: http.Header{"Content-Type": {"application/json"}}, RequestBody: []byte(`{"name":"fixture"}`), Status: 201, ResponseHeaders: http.Header{"Content-Type": {"application/json"}}, ResponseBody: []byte(`{"ok":true}`)}
}

func TestValidateCasesAcceptsSuccessAndRejectedRequest(t *testing.T) {
	doc, err := loadContract(context.Background(), []byte(minimalContract))
	if err != nil {
		t.Fatal(err)
	}
	negative := validSample()
	negative.RequestBody = []byte(`not JSON`)
	negative.Status = 400
	negative.ResponseBody = []byte(`{"error":{"code":"invalid_sample","message":"Bad Request"}}`)
	if err := validateCases(context.Background(), doc, contractReport{Cases: []contractCase{validSample(), negative}}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCasesRejectsContractDriftAndCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*contractCase)
		want   string
	}{
		{"request shape", func(c *contractCase) { c.RequestBody = []byte(`{"other":true}`) }, "successful request differs"},
		{"response shape", func(c *contractCase) { c.ResponseBody = []byte(`{"ok":"wrong"}`) }, "response differs"},
		{"operation", func(c *contractCase) { c.OperationID = "wrong" }, "operation does not match"},
		{"route", func(c *contractCase) { c.Path = "/unknown" }, "operation does not match"},
		{"status", func(c *contractCase) { c.Status = 202 }, "undocumented response status"},
		{"authorization", func(c *contractCase) { c.RequestHeaders["aUtHoRiZaTiOn"] = []string{"Bearer synthetic-private-value"} }, "non-allowlisted header"},
		{"cookie", func(c *contractCase) { c.RequestHeaders.Set("Cookie", "synthetic-private-value") }, "non-allowlisted header"},
		{"csrf", func(c *contractCase) { c.RequestHeaders.Set("X-CSRF-Token", "synthetic-private-value") }, "non-allowlisted header"},
		{"set cookie", func(c *contractCase) { c.ResponseHeaders.Set("Set-Cookie", "synthetic-private-value") }, "non-allowlisted header"},
		{"absolute URL", func(c *contractCase) { c.Path = "https://example.test/sample" }, "invalid request path"},
		{"authority URL", func(c *contractCase) { c.Path = "//example.test/sample" }, "invalid request path"},
		{"no content body", func(c *contractCase) { c.Status = 204 }, "no-content response has a body"},
		{"error code", func(c *contractCase) {
			c.Status = 400
			c.ResponseBody = []byte(`{"error":{"code":"unknown","message":"Bad Request"}}`)
		}, "error code is undocumented"},
		{"error disclosure", func(c *contractCase) {
			c.Status = 400
			c.ResponseBody = []byte(`{"error":{"code":"invalid_sample","message":"synthetic-private-value"}}`)
		}, "error envelope is not generic"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := loadContract(context.Background(), []byte(minimalContract))
			if err != nil {
				t.Fatal(err)
			}
			c := validSample()
			tc.mutate(&c)
			err = validateCases(context.Background(), doc, contractReport{Cases: []contractCase{validSample(), c}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q failure, got %v", tc.want, err)
			}
			if strings.Contains(err.Error(), "synthetic-private-value") {
				t.Fatal("checker disclosed private input")
			}
		})
	}
}

func TestValidateCasesRequiresEveryOperationSuccess(t *testing.T) {
	doc, err := loadContract(context.Background(), []byte(minimalContract))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCases(context.Background(), doc, contractReport{}); err == nil || !strings.Contains(err.Error(), "lacks a successful") {
		t.Fatalf("expected coverage failure, got %v", err)
	}
}

func TestLoadContractRejectsExternalReferencesAndInvalidDocuments(t *testing.T) {
	for _, input := range []string{"null", `{}`, `not JSON`, strings.Replace(minimalContract, `"3.0.3"`, `"3.1.0"`, 1), strings.Replace(minimalContract, `"type":"boolean"`, `"$ref":"https://example.test/never-fetch.json"`, 1), strings.Replace(minimalContract, `"type":"boolean"`, `"$ref":"file:///never-read.json"`, 1), strings.Replace(minimalContract, `"createSample"`, `"unsafe operation ID"`, 1)} {
		if _, err := loadContract(context.Background(), []byte(input)); err == nil {
			t.Fatal("accepted invalid or externally referenced contract")
		}
	}
}

func TestLoadContractRejectsRootAndNestedDuplicateMembers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{
			name:  "root member",
			input: strings.Replace(minimalContract, `"openapi":"3.0.3"`, `"openapi":"3.0.3","openapi":"3.0.3"`, 1),
		},
		{
			name:  "nested member",
			input: strings.Replace(minimalContract, `"title":"synthetic test"`, `"title":"synthetic test","title":"duplicate"`, 1),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.input == minimalContract {
				t.Fatal("duplicate-member fixture did not match the contract")
			}
			if _, err := loadContract(context.Background(), []byte(tc.input)); err == nil {
				t.Fatal("accepted duplicate OpenAPI object member")
			}
		})
	}
}

func TestRejectDuplicateContractMembersUsesExactNames(t *testing.T) {
	if err := rejectDuplicateContractMembers([]byte(`{"Name":1,"name":2}`)); err != nil {
		t.Fatalf("case-distinct contract member names were folded together: %v", err)
	}
	if err := rejectDuplicateContractMembers([]byte(`{"name":1,"name":2}`)); err == nil {
		t.Fatal("accepted repeated exact contract member name")
	}
}

func TestRejectDuplicateContractMembersDecodesEscapesAndScopesKeysPerObject(t *testing.T) {
	for _, tc := range []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "escaped equivalent member name",
			input:     `{"a":1,"\u0061":2}`,
			wantError: true,
		},
		{
			name:      "duplicate in object nested in array",
			input:     `{"items":[{"a":1,"a":2}]}`,
			wantError: true,
		},
		{
			name:  "same key in sibling objects",
			input: `{"left":{"a":1},"right":{"a":2}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := rejectDuplicateContractMembers([]byte(tc.input))
			if tc.wantError && err == nil {
				t.Fatal("accepted duplicate decoded member name")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("rejected same key in a separate object: %v", err)
			}
		})
	}
}

func TestRejectDuplicateContractMembersRejectsMalformedAndTrailingInput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "unclosed top-level object", input: `{"a":1`},
		{name: "malformed top-level value", input: `{"a":}`},
		{name: "second top-level JSON value", input: `{"a":1} {}`},
		{name: "trailing non-JSON bytes", input: `{"a":1} trailing`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := rejectDuplicateContractMembers([]byte(tc.input)); err == nil {
				t.Fatal("accepted malformed or trailing JSON input")
			}
		})
	}
}

func TestLoadContractRejectsDuplicateMembersBeforeExternalReferenceResolution(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"boolean"}`))
	}))
	defer server.Close()

	input := strings.Replace(minimalContract,
		`"title":"synthetic test"`, `"title":"synthetic test","title":"duplicate"`, 1)
	input = strings.Replace(input, `"type":"boolean"`, `"$ref":"`+server.URL+`/schema.json"`, 1)
	if input == minimalContract {
		t.Fatal("duplicate/reference fixture did not match the contract")
	}
	if _, err := loadContract(context.Background(), []byte(input)); err == nil {
		t.Fatal("accepted duplicate member containing an external reference")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("resolved an external reference %d times for invalid input", got)
	}
}

func TestDecodeReportAcceptsValidMarshaledReport(t *testing.T) {
	data, err := json.Marshal(contractReport{Cases: []contractCase{validSample()}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := decodeReport(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cases) != 1 || report.Cases[0].OperationID != "createSample" {
		t.Fatalf("decoded unexpected report: %#v", report)
	}
}

func TestDecodeReportRejectsDuplicateJSONMembersInReportObjects(t *testing.T) {
	data, err := json.Marshal(contractReport{Cases: []contractCase{validSample()}})
	if err != nil {
		t.Fatal(err)
	}
	valid := string(data)
	for _, tc := range []struct {
		name  string
		input string
	}{
		{
			name:  "duplicate top-level report field",
			input: strings.Replace(valid, `{"cases":`, `{"cases":[],"cases":`, 1),
		},
		{
			name:  "duplicate nested case field",
			input: strings.Replace(valid, `"status":201`, `"status":201,"status":400`, 1),
		},
		{
			name:  "case-variant matching the same nested struct field",
			input: strings.Replace(valid, `"status":201`, `"Status":201,"status":201`, 1),
		},
		{
			name:  "duplicate nested header map field",
			input: strings.Replace(valid, `"Content-Type":["application/json"]`, `"Content-Type":["application/json"],"Content-Type":["text/plain"]`, 1),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.input == valid {
				t.Fatal("duplicate-member fixture did not match the marshaled report")
			}
			if _, err := decodeReport([]byte(tc.input)); err == nil {
				t.Fatal("accepted duplicate JSON object member")
			} else if err.Error() != "handler report is invalid" {
				t.Fatalf("duplicate-member error was not redacted: %v", err)
			}
		})
	}
}

func TestDecodeReportRejectsObjectWithTooManyMembers(t *testing.T) {
	var input strings.Builder
	input.WriteString(`{"cases":[{`)
	for i := 0; i < 17; i++ {
		if i > 0 {
			input.WriteByte(',')
		}
		input.WriteString(`"` + string(rune('a'+i)) + `":null`)
	}
	input.WriteString(`}]}`)
	if _, err := decodeReport([]byte(input.String())); err == nil || err.Error() != "handler report is invalid" {
		t.Fatalf("expected redacted object member limit failure, got %v", err)
	}
}

func TestDecodeReportRejectsMalformedOrAmbiguousInput(t *testing.T) {
	data, err := json.Marshal(contractReport{Cases: []contractCase{validSample()}})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{`null`, `{"cases":[]}`, `{"cases":[],"unknown":true}`, string(data) + ` {}`, `{"cases":[{"unknown":true}]}`} {
		if _, err := decodeReport([]byte(input)); err == nil {
			t.Fatal("accepted malformed report")
		}
	}
}

func TestRunReadsBoundedLocalInputs(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.json")
	report := filepath.Join(dir, "report.json")
	if err := os.WriteFile(spec, []byte(minimalContract), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(contractReport{Cases: []contractCase{validSample()}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(report, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{spec, report}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{spec}); err != nil {
		t.Fatal(err)
	}
	if _, err := boundedRead(report, 1); err == nil {
		t.Fatal("accepted oversized input")
	}
	if err := run(nil); err == nil {
		t.Fatal("accepted missing arguments")
	}
}
