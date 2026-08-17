package builtin

import (
	"strings"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// TLSDisabledRule находит отключённый TLS и отключённую проверку сертификатов.
type TLSDisabledRule struct{}

const tlsDisabledID = "tls-disabled"

var tlsSectionKeys = map[string]struct{}{"tls": {}, "ssl": {}}

var insecureSkipVerifyKeys = map[string]struct{}{
	"insecure_skip_verify": {}, "insecureskipverify": {}, "insecure-skip-verify": {},
	"skip_verify": {}, "skip-verify": {}, "tls_verify": {}, "verify_tls": {},
}

func (TLSDisabledRule) ID() string { return tlsDisabledID }

func (TLSDisabledRule) Check(cfg *config.Config) []issue.Issue {
	var found []issue.Issue
	cfg.Walk(func(n config.Node) {
		key := strings.ToLower(n.Key)

		// tls: false / ssl: false напрямую.
		if _, isTLSSection := tlsSectionKeys[key]; isTLSSection {
			if b, ok := config.AsBool(n.Value); ok && !b {
				found = append(found, newIssue(tlsDisabledID, issue.High, n.DotPath(),
					"TLS отключён.",
					"Включите TLS для защиты трафика от перехвата."))
			}
			return
		}

		// enabled: false внутри секции tls/ssl.
		if key == "enabled" && hasParent(n.Path, tlsSectionKeys) {
			if b, ok := config.AsBool(n.Value); ok && !b {
				found = append(found, newIssue(tlsDisabledID, issue.High, n.DotPath(),
					"TLS отключён.",
					"Включите TLS для защиты трафика от перехвата."))
			}
			return
		}

		// insecure_skip_verify: true и аналоги — проверка сертификата отключена.
		if _, skip := insecureSkipVerifyKeys[key]; skip {
			if b, ok := config.AsBool(n.Value); ok && b {
				found = append(found, newIssue(tlsDisabledID, issue.High, n.DotPath(),
					"проверка TLS-сертификата отключена.",
					"Включите проверку сертификата, используйте доверенный CA."))
			}
		}
	})
	return found
}

// hasParent проверяет, есть ли среди родительских сегментов пути ключ из set.
// Path включает сам узел последним элементом, поэтому родители — всё, кроме
// последнего.
func hasParent(path []string, set map[string]struct{}) bool {
	if len(path) < 2 {
		return false
	}
	parent := strings.ToLower(path[len(path)-2])
	_, ok := set[parent]
	return ok
}
