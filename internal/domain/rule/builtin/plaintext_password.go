package builtin

import (
	"strings"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// PlaintextPasswordRule находит пароли/секреты, записанные в конфиге
// открытым текстом вместо ссылки на переменную окружения/секрет-хранилище.
type PlaintextPasswordRule struct{}

const plaintextPasswordID = "plaintext-password"

var secretKeyContains = []string{
	"password", "passwd", "pwd", "secret", "token", "api_key", "apikey", "api-key",
}

// secretRefPrefixes — значения, похожие на ссылку на секрет, а не на сам секрет.
var secretRefPrefixes = []string{"${", "env:", "vault:", "$"}

var placeholderValues = map[string]struct{}{
	"": {}, "changeme": {}, "redacted": {}, "xxx": {}, "***": {}, "<redacted>": {},
}

func (PlaintextPasswordRule) ID() string { return plaintextPasswordID }

func (PlaintextPasswordRule) Check(cfg *config.Config) []issue.Issue {
	var found []issue.Issue
	cfg.Walk(func(n config.Node) {
		key := strings.ToLower(n.Key)
		if !containsAny(key, secretKeyContains) {
			return
		}

		s, ok := config.AsString(n.Value)
		if !ok {
			return
		}
		if _, placeholder := placeholderValues[strings.ToLower(strings.TrimSpace(s))]; placeholder {
			return
		}
		if hasAnyPrefix(s, secretRefPrefixes) {
			return
		}

		found = append(found, newIssue(plaintextPasswordID, issue.High, n.DotPath(),
			"секрет хранится в открытом виде в конфиге.",
			"Вынесите значение в переменную окружения или секрет-хранилище (Vault, KMS и т.п.)."))
	})
	return found
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
