// Package rule определяет контракт правил проверки конфигурации и реестр,
// через который новые правила подключаются без изменения движка сканера.
package rule

import (
	"io/fs"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// Rule проверяет разобранный конфиг и возвращает найденные проблемы.
type Rule interface {
	ID() string
	Check(cfg *config.Config) []issue.Issue
}

// FileAwareRule — опциональное расширение Rule для правил, которым нужны
// метаданные самого файла на диске (права доступа и т.п.). Работает только
// при сканировании реального файла, при чтении из stdin пропускается.
type FileAwareRule interface {
	Rule
	CheckFile(info fs.FileInfo, path string) []issue.Issue
}

// Registry — набор активных правил. Позволяет расширять утилиту новыми
// правилами без изменения кода сканера (регистрация в одном месте).
type Registry struct {
	rules []Rule
}

func NewRegistry(rules ...Rule) *Registry {
	return &Registry{rules: rules}
}

func (r *Registry) Register(rules ...Rule) {
	r.rules = append(r.rules, rules...)
}

func (r *Registry) Rules() []Rule {
	return r.rules
}
