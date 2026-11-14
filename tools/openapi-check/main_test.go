package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func TestDecodeReportRejectsMalformedOrAmbiguousInput(t *testing.T) {
	data, err := json.Marshal(contractReport{Cases: []contractCase{validSample()}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeReport(data); err != nil {
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
