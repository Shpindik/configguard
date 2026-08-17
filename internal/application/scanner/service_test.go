package scanner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestScanReader_ExamplesFromSpec(t *testing.T) {
	svc := newService()

	debugJSON := `{"version": "0.1", "log":{"output":"stdout", "level": "debug"}}`
	report, err := svc.ScanReader(strings.NewReader(debugJSON), "stdin", parser.Unknown)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 1 || report.Issues[0].String() != "LOW: логирование в debug-режиме. Поменяйте режим на более избирательный (info+)." {
		t.Fatalf("unexpected report: %+v", report.Issues)
	}

	weakAlgoYAML := "version: 2.2\nstorage:\n  digest-algorithm: MD5\n"
	report, err = svc.ScanReader(strings.NewReader(weakAlgoYAML), "stdin", parser.Unknown)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 1 || report.Issues[0].String() != "HIGH: слишком слабый алгоритм - MD5. Замените его на более безопасный." {
		t.Fatalf("unexpected report: %+v", report.Issues)
	}
}

func TestScanFile_UsesFileAwareRules(t *testing.T) {
	svc := newService()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"server":{"host":"127.0.0.1"}}`), 0o666); err != nil {
		t.Fatal(err)
	}

	report, err := svc.ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !report.HasIssues() {
		t.Fatalf("expected file-permission issue for world-writable file, got none")
	}
	found := false
	for _, iss := range report.Issues {
		if iss.RuleID == "file-permission" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected file-permission issue, got %+v", report.Issues)
	}
}

func TestScanFile_NoIssues(t *testing.T) {
	svc := newService()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"server":{"host":"127.0.0.1"},"tls":{"enabled":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := svc.ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.HasIssues() {
		t.Fatalf("expected no issues, got %+v", report.Issues)
	}
}

func TestScanFile_MissingFile(t *testing.T) {
	svc := newService()
	if _, err := svc.ScanFile("/no/such/file.json"); err == nil {
		t.Fatal("expected error for missing file")
	}
}
