// openapi-check validates the development contract and private handler reports.
// It never starts an API, fetches external references, or prints response bodies.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

type contractCase struct {
	Name            string      `json:"name"`
	OperationID     string      `json:"operation_id"`
	Method          string      `json:"method"`
	Path            string      `json:"path"`
	RequestHeaders  http.Header `json:"request_headers"`
	RequestBody     []byte      `json:"request_body"`
	Status          int         `json:"status"`
	ResponseHeaders http.Header `json:"response_headers"`
	ResponseBody    []byte      `json:"response_body"`
}

type contractReport struct {
	Cases []contractCase `json:"cases"`
}

var operationID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,95}$`)

func boundedRead(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("input unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("input unreadable or exceeds bound")
	}
	return data, nil
}

func internalReferences(value any) bool {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "$ref" {
				ref, ok := child.(string)
				if !ok || !strings.HasPrefix(ref, "#/") {
					return false
				}
			}
			if !internalReferences(child) {
				return false
			}
		}
	case []any:
		for _, child := range value {
			if !internalReferences(child) {
				return false
			}
		}
	}
	return true
}

func loadContract(ctx context.Context, data []byte) (*openapi3.T, error) {
	if rejectDuplicateContractMembers(data) != nil {
		return nil, errors.New("contract must be JSON with internal references only")
	}
	var raw any
	if json.Unmarshal(data, &raw) != nil || !internalReferences(raw) {
		return nil, errors.New("contract must be JSON with internal references only")
	}
	loader := openapi3.NewLoader()
	loader.Context = ctx
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromData(data)
	if err != nil || document == nil || document.OpenAPI != "3.0.3" {
		return nil, errors.New("OpenAPI contract could not be loaded")
	}
	if err := document.Validate(ctx); err != nil {
		return nil, errors.New("OpenAPI contract is invalid")
	}
	ids := map[string]bool{}
	for _, path := range document.Paths.Map() {
		for _, operation := range path.Operations() {
			if !operationID.MatchString(operation.OperationID) || ids[operation.OperationID] {
				return nil, errors.New("operation IDs must be unique and log-safe")
			}
			ids[operation.OperationID] = true
		}
	}
	return document, nil
}

// rejectDuplicateContractMembers rejects repeated decoded member names exactly.
// OpenAPI input has no handler-report member-count or case-folding policy.
func rejectDuplicateContractMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanContractJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return errors.New("contract has trailing JSON values")
	}
	return nil
}

func scanContractJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("contract object member is invalid")
			}
			if _, exists := seen[key]; exists {
				return errors.New("contract has duplicate object members")
			}
			seen[key] = struct{}{}
			if err := scanContractJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if end, ok := closing.(json.Delim); !ok || end != '}' {
			return errors.New("contract object is invalid")
		}
	case '[':
		for decoder.More() {
			if err := scanContractJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if end, ok := closing.(json.Delim); !ok || end != ']' {
			return errors.New("contract array is invalid")
		}
	default:
		return errors.New("contract value is invalid")
	}
	return nil
}

func decodeReport(data []byte) (contractReport, error) {
	var report contractReport
	if err := rejectDuplicateJSONMembers(data); err != nil {
		return report, errors.New("handler report is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&report) != nil || len(report.Cases) == 0 || len(report.Cases) > 512 {
		return report, errors.New("handler report is invalid")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return report, errors.New("handler report has trailing data")
	}
	return report, nil
}

const maxJSONMembers = 16

// rejectDuplicateJSONMembers scans the complete first JSON value before typed
// decoding. Case folding is deliberately as strict as encoding/json's
// case-insensitive struct-field matching; it also rejects ambiguous header
// names, which are case-insensitive at the HTTP layer.
func rejectDuplicateJSONMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	return scanJSONValue(decoder, 0)
}

func scanJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("handler report nesting exceeds limit")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{}, maxJSONMembers)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("handler report object member is invalid")
			}
			folded := foldJSONMemberName(key)
			if _, exists := seen[folded]; exists {
				return errors.New("handler report has duplicate object members")
			}
			if len(seen) >= maxJSONMembers {
				return errors.New("handler report object has too many members")
			}
			seen[folded] = struct{}{}
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if end, ok := closing.(json.Delim); !ok || end != '}' {
			return errors.New("handler report object is invalid")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if end, ok := closing.(json.Delim); !ok || end != ']' {
			return errors.New("handler report array is invalid")
		}
	default:
		return errors.New("handler report value is invalid")
	}
	return nil
}

// foldJSONMemberName mirrors encoding/json's case-insensitive name folding so
// names that target the same struct field cannot be supplied more than once.
func foldJSONMemberName(name string) string {
	var folded strings.Builder
	folded.Grow(len(name))
	for _, char := range name {
		if char >= 'a' && char <= 'z' {
			char -= 'a' - 'A'
		} else if char > 0x7f {
			char = foldJSONRune(char)
		}
		folded.WriteRune(char)
	}
	return folded.String()
}

func foldJSONRune(char rune) rune {
	folded := char
	for candidate := unicode.SimpleFold(char); candidate != char; candidate = unicode.SimpleFold(candidate) {
		if candidate < folded {
			folded = candidate
		}
	}
	return folded
}

func allowedHeaders(headers http.Header, response bool) bool {
	allowed := map[string]bool{"content-type": true, "accept": true, "origin": true, "idempotency-key": true}
	if response {
		allowed = map[string]bool{"content-type": true, "cache-control": true, "x-content-type-options": true, "x-request-id": true}
	}
	for key := range headers {
		if !allowed[strings.ToLower(key)] {
			return false
		}
	}
	return true
}

func validateCases(ctx context.Context, document *openapi3.T, report contractReport) error {
	router, err := gorillamux.NewRouter(document)
	if err != nil {
		return errors.New("contract routes are invalid")
	}
	success := map[string]bool{}
	for index, sample := range report.Cases {
		failure := func(phase string) error { return fmt.Errorf("case %d: %s", index+1, phase) }
		if !allowedHeaders(sample.RequestHeaders, false) || !allowedHeaders(sample.ResponseHeaders, true) {
			return failure("report contains a non-allowlisted header")
		}
		if !strings.HasPrefix(sample.Path, "/") || strings.HasPrefix(sample.Path, "//") || strings.ContainsAny(sample.Path, "\r\n") {
			return failure("invalid request path")
		}
		request, err := http.NewRequestWithContext(ctx, sample.Method, "http://127.0.0.1:8090"+sample.Path, bytes.NewReader(sample.RequestBody))
		if err != nil {
			return failure("invalid request context")
		}
		request.Header = sample.RequestHeaders.Clone()
		route, parameters, err := router.FindRoute(request)
		if err != nil || route.Operation.OperationID != sample.OperationID {
			return failure("operation does not match the documented route")
		}
		response := route.Operation.Responses.Value(strconv.Itoa(sample.Status))
		if response == nil || response.Value == nil {
			return failure("undocumented response status")
		}
		// Reports omit credentials. Actual Go handler tests assert authentication,
		// TLS/CSRF and ownership; this tool checks wire shapes, not credential trust.
		options := &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			IncludeResponseStatus: true, SkipSettingDefaults: true}
		input := &openapi3filter.RequestValidationInput{Request: request, PathParams: parameters, Route: route, Options: options}
		if sample.Status >= 200 && sample.Status < 300 {
			if openapi3filter.ValidateRequest(ctx, input) != nil {
				return failure("successful request differs from contract")
			}
			success[sample.OperationID] = true
		}
		if sample.Status == 204 && len(sample.ResponseBody) != 0 {
			return failure("no-content response has a body")
		}
		output := &openapi3filter.ResponseValidationInput{RequestValidationInput: input,
			Status: sample.Status, Header: sample.ResponseHeaders.Clone(), Options: options}
		output.SetBodyBytes(sample.ResponseBody)
		if openapi3filter.ValidateResponse(ctx, output) != nil {
			return failure("response differs from contract")
		}
		if sample.Status >= 400 {
			var body struct {
				Error struct{ Code, Message string } `json:"error"`
			}
			if json.Unmarshal(sample.ResponseBody, &body) != nil || body.Error.Message != http.StatusText(sample.Status) {
				return failure("error envelope is not generic")
			}
			encoded, _ := json.Marshal(response.Value.Extensions["x-error-codes"])
			var codes []string
			if json.Unmarshal(encoded, &codes) != nil || !contains(codes, body.Error.Code) {
				return failure("error code is undocumented")
			}
		}
	}
	for _, path := range document.Paths.Map() {
		for _, operation := range path.Operations() {
			if !success[operation.OperationID] {
				return errors.New("documented operation lacks a successful handler sample")
			}
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func run(arguments []string) error {
	if len(arguments) < 1 || len(arguments) > 2 {
		return errors.New("usage: openapi-check SPEC_JSON [PRIVATE_REPORT_JSON]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	data, err := boundedRead(arguments[0], 2<<20)
	if err != nil {
		return err
	}
	document, err := loadContract(ctx, data)
	if err != nil {
		return err
	}
	if len(arguments) == 2 {
		data, err := boundedRead(arguments[1], 4<<20)
		if err != nil {
			return err
		}
		report, err := decodeReport(data)
		if err != nil {
			return err
		}
		if err := validateCases(ctx, document, report); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "contract check:", err)
		os.Exit(1)
	}
	fmt.Println("OpenAPI contract checks passed; authentication is asserted separately by handler tests")
}
