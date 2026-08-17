package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
	"configguard/internal/domain/rule"
)

type ruleCase struct {
	name       string
	cfg        map[string]any
	wantIssues int
	wantLevel  issue.Level
}

func runCases(t *testing.T, r rule.Rule, cases []ruleCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := r.Check(config.New(c.cfg, nil))
			if len(got) != c.wantIssues {
				t.Fatalf("issues = %d, want %d (%+v)", len(got), c.wantIssues, got)
			}
			if c.wantIssues > 0 && got[0].Level != c.wantLevel {
				t.Fatalf("level = %s, want %s", got[0].Level, c.wantLevel)
			}
		})
	}
}

func TestDebugLoggingRule(t *testing.T) {
	runCases(t, DebugLoggingRule{}, []ruleCase{
		{"debug-level", map[string]any{"log": map[string]any{"level": "debug"}}, 1, issue.Low},
		{"info-level", map[string]any{"log": map[string]any{"level": "info"}}, 0, ""},
		{"debug-flag-true", map[string]any{"debug": true}, 1, issue.Low},
		{"debug-flag-false", map[string]any{"debug": false}, 0, ""},
	})
}

func TestPlaintextPasswordRule(t *testing.T) {
	runCases(t, PlaintextPasswordRule{}, []ruleCase{
		{"plain", map[string]any{"db": map[string]any{"password": "abc123"}}, 1, issue.High},
		{"env-ref", map[string]any{"db": map[string]any{"password": "${DB_PASSWORD}"}}, 0, ""},
		{"empty", map[string]any{"db": map[string]any{"password": ""}}, 0, ""},
		{"placeholder", map[string]any{"db": map[string]any{"password": "CHANGEME"}}, 0, ""},
	})
}

func TestUnrestrictedBindRule(t *testing.T) {
	runCases(t, UnrestrictedBindRule{}, []ruleCase{
		{"all-interfaces", map[string]any{"server": map[string]any{"host": "0.0.0.0"}}, 1, issue.Medium},
		{"with-port", map[string]any{"server": map[string]any{"host": "0.0.0.0:8080"}}, 1, issue.Medium},
		{"localhost", map[string]any{"server": map[string]any{"host": "127.0.0.1"}}, 0, ""},
	})
}

func TestTLSDisabledRule(t *testing.T) {
	runCases(t, TLSDisabledRule{}, []ruleCase{
		{"section-false", map[string]any{"tls": false}, 1, issue.High},
		{"nested-enabled-false", map[string]any{"tls": map[string]any{"enabled": false}}, 1, issue.High},
		{"nested-enabled-true", map[string]any{"tls": map[string]any{"enabled": true}}, 0, ""},
		{"insecure-skip-verify", map[string]any{"insecure_skip_verify": true}, 1, issue.High},
		{"skip-verify-false", map[string]any{"insecure_skip_verify": false}, 0, ""},
	})
}

func TestWeakAlgorithmRule(t *testing.T) {
	runCases(t, WeakAlgorithmRule{}, []ruleCase{
		{"md5", map[string]any{"storage": map[string]any{"digest-algorithm": "MD5"}}, 1, issue.High},
		{"sha256", map[string]any{"storage": map[string]any{"digest-algorithm": "SHA256"}}, 0, ""},
	})

	got := WeakAlgorithmRule{}.Check(config.New(map[string]any{
		"storage": map[string]any{"digest-algorithm": "MD5"},
	}, nil))
	want := "HIGH: слишком слабый алгоритм - MD5. Замените его на более безопасный."
	if got[0].String() != want {
		t.Fatalf("String() = %q, want %q", got[0].String(), want)
	}
}

func TestPermissiveFileModeRule(t *testing.T) {
	runCases(t, PermissiveFileModeRule{}, []ruleCase{
		{"world-writable-string", map[string]any{"upload": map[string]any{"mode": "0777"}}, 1, issue.High},
		{"world-writable-number", map[string]any{"upload": map[string]any{"mode": float64(511)}}, 1, issue.High},
		{"group-writable", map[string]any{"upload": map[string]any{"mode": "0770"}}, 1, issue.Medium},
		{"safe", map[string]any{"upload": map[string]any{"mode": "0640"}}, 0, ""},
	})
}

func TestFilePermissionRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("a: 1"), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		mode os.FileMode
		want issue.Level
		none bool
	}{
		{"safe", 0o600, "", true},
		{"world-readable", 0o644, issue.Medium, false},
		{"world-writable", 0o666, issue.High, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := os.Chmod(path, c.mode); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			got := FilePermissionRule{}.CheckFile(info, path)
			if c.none {
				if len(got) != 0 {
					t.Fatalf("expected no issues, got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Level != c.want {
				t.Fatalf("got %+v, want single issue with level %s", got, c.want)
			}
		})
	}
}
