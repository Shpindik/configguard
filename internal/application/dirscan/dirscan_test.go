package dirscan_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"configguard/internal/application/dirscan"
	"configguard/internal/application/scanner"
	"configguard/internal/domain/rule"
	"configguard/internal/domain/rule/builtin"
	"configguard/internal/infrastructure/parser"
)

func newService() *scanner.Service {
	rules := rule.NewRegistry(builtin.All()...)
	parsers := parser.NewRegistry(parser.JSONParser{}, parser.YAMLParser{})
	return scanner.NewService(rules, parsers)
}

func TestScan_FindsAllFilesConcurrently(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(dir, "a.json"):      `{"log":{"level":"debug"}}`,
		filepath.Join(dir, "b.yaml"):      "server:\n  host: 0.0.0.0\n",
		filepath.Join(sub, "c.yml"):       "tls:\n  enabled: true\n",
		filepath.Join(dir, "ignored.txt"): "not a config",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	svc := newService()
	results, err := dirscan.Scan(context.Background(), svc, dir, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 3 {
		t.Fatalf("got %d results, want 3 (ignored.txt must be skipped): %+v", len(results), results)
	}

	byPath := make(map[string]dirscan.Result, len(results))
	for _, r := range results {
		byPath[r.Path] = r
	}

	if !byPath[filepath.Join(dir, "a.json")].Report.HasIssues() {
		t.Error("a.json: expected debug-logging issue")
	}
	if !byPath[filepath.Join(dir, "b.yaml")].Report.HasIssues() {
		t.Error("b.yaml: expected unrestricted-bind issue")
	}
	if byPath[filepath.Join(sub, "c.yml")].Report.HasIssues() {
		t.Error("c.yml: expected no issues")
	}
}

func TestScan_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	svc := newService()
	if _, err := dirscan.Scan(context.Background(), svc, dir, 2); err == nil {
		t.Fatal("expected error for directory without config files")
	}
}
