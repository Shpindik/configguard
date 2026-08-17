package builtin

import (
	"strings"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// WeakAlgorithmRule находит использование устаревших/небезопасных алгоритмов.
type WeakAlgorithmRule struct{}

const weakAlgorithmID = "weak-algorithm"

var algorithmKeyHints = []string{"algorithm", "cipher", "digest", "hash"}

var weakAlgorithms = map[string]struct{}{
	"md5": {}, "sha1": {}, "sha-1": {}, "des": {}, "3des": {}, "rc4": {},
}

func (WeakAlgorithmRule) ID() string { return weakAlgorithmID }

func (WeakAlgorithmRule) Check(cfg *config.Config) []issue.Issue {
	var found []issue.Issue
	cfg.Walk(func(n config.Node) {
		key := strings.ToLower(n.Key)
		if !containsAny(key, algorithmKeyHints) {
			return
		}
		s, ok := config.AsString(n.Value)
		if !ok {
			return
		}
		norm := strings.ToLower(strings.TrimSpace(s))
		if _, weak := weakAlgorithms[norm]; !weak {
			return
		}
		found = append(found, newIssue(weakAlgorithmID, issue.High, n.DotPath(),
			"слишком слабый алгоритм - "+s+".",
			"Замените его на более безопасный."))
	})
	return found
}
