// Package scanner — прикладной сервис проверки конфигурации: связывает
// парсеры (infrastructure) и правила (domain), не зная про CLI/HTTP/gRPC.
package scanner

import "configguard/internal/domain/issue"

// Report — результат проверки одного конфига.
type Report struct {
	Source string        `json:"source"`
	Issues []issue.Issue `json:"issues"`
}

func (r Report) HasIssues() bool { return len(r.Issues) > 0 }

func (r Report) CountByLevel() map[issue.Level]int {
	counts := map[issue.Level]int{issue.Low: 0, issue.Medium: 0, issue.High: 0}
	for _, i := range r.Issues {
		counts[i.Level]++
	}
	return counts
}
