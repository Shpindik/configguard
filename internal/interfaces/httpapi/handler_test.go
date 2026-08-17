package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"configguard/internal/application/scanner"
	"configguard/internal/domain/rule"
	"configguard/internal/domain/rule/builtin"
	"configguard/internal/infrastructure/parser"
	"configguard/internal/interfaces/httpapi"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	rules := rule.NewRegistry(builtin.All()...)
	parsers := parser.NewRegistry(parser.JSONParser{}, parser.YAMLParser{})
	svc := scanner.NewService(rules, parsers)

	handler := httpapi.NewServer(svc, ":0").Handler()
	return httptest.NewServer(handler)
}

func TestHandleScan_FindsIssues(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body := `{"version": "0.1", "log":{"output":"stdout", "level": "debug"}}`
	resp, err := http.Post(srv.URL+"/api/v1/scan", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var parsed struct {
		Issues    []map[string]any `json:"issues"`
		HasIssues bool             `json:"has_issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatal(err)
	}
	if !parsed.HasIssues || len(parsed.Issues) != 1 {
		t.Fatalf("unexpected response: %+v", parsed)
	}
	if parsed.Issues[0]["level"] != "LOW" {
		t.Fatalf("level = %v, want LOW", parsed.Issues[0]["level"])
	}
}

func TestHandleScan_InvalidBody(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/scan", "application/json", strings.NewReader("{not valid"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHandleHealth(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
