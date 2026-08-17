// Package issue содержит доменные типы для описания найденных проблем.
package issue

import "sort"

// Level — уровень серьёзности проблемы.
type Level string

const (
	Low    Level = "LOW"
	Medium Level = "MEDIUM"
	High   Level = "HIGH"
)

// weight нужен для сортировки issues по убыванию серьёзности.
func (l Level) weight() int {
	switch l {
	case High:
		return 3
	case Medium:
		return 2
	case Low:
		return 1
	default:
		return 0
	}
}

// Issue — одна найденная проблема конфигурации.
type Issue struct {
	RuleID         string `json:"rule_id"`
	Level          Level  `json:"level"`
	Field          string `json:"field,omitempty"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation"`
}

// String форматирует issue в человекочитаемую строку из ТЗ:
// "HIGH: слишком слабый алгоритм - MD5. Замените его на более безопасный."
func (i Issue) String() string {
	return string(i.Level) + ": " + i.Message + " " + i.Recommendation
}

// SortBySeverity сортирует issues от самых серьёзных к наименее серьёзным (in-place).
func SortBySeverity(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		return issues[i].Level.weight() > issues[j].Level.weight()
	})
}
