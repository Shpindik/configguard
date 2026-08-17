// Package builtin содержит встроенные правила проверки конфигурации.
// Новое правило подключается добавлением файла сюда и регистрацией в All().
package builtin

import (
	"configguard/internal/domain/issue"
)

func newIssue(ruleID string, lvl issue.Level, field, message, recommendation string) issue.Issue {
	return issue.Issue{
		RuleID:         ruleID,
		Level:          lvl,
		Field:          field,
		Message:        message,
		Recommendation: recommendation,
	}
}
