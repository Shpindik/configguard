// Package parser реализует разбор конфигов разных форматов в доменное
// дерево config.Config. Новый формат подключается новым Parser в Registry.
package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"configguard/internal/domain/config"
)

type Format string

const (
	JSON    Format = "json"
	YAML    Format = "yaml"
	Unknown Format = ""
)

// Parser разбирает сырые байты конфига в доменное дерево.
type Parser interface {
	Format() Format
	Parse(data []byte) (*config.Config, error)
}

// Registry подбирает нужный Parser по формату/расширению/содержимому.
type Registry struct {
	parsers map[Format]Parser
}

func NewRegistry(parsers ...Parser) *Registry {
	r := &Registry{parsers: make(map[Format]Parser, len(parsers))}
	for _, p := range parsers {
		r.parsers[p.Format()] = p
	}
	return r
}

func (r *Registry) Get(f Format) (Parser, bool) {
	p, ok := r.parsers[f]
	return p, ok
}

// DetectByExt определяет формат по расширению файла.
func DetectByExt(path string) Format {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return JSON
	case ".yaml", ".yml":
		return YAML
	default:
		return Unknown
	}
}

// Sniff определяет формат по содержимому — используется для stdin, где
// расширения нет. JSON — более строгий синтаксис, поэтому его проверяем
// первым: валидный JSON почти никогда не является случайным совпадением.
func Sniff(data []byte) Format {
	trimmed := bytes.TrimSpace(data)
	if json.Valid(trimmed) {
		return JSON
	}
	return YAML
}

// Parse разбирает данные, выбирая парсер по формату. Unknown → пробуем Sniff.
func (r *Registry) Parse(data []byte, format Format) (*config.Config, error) {
	if format == Unknown {
		format = Sniff(data)
	}
	p, ok := r.Get(format)
	if !ok {
		return nil, fmt.Errorf("parser: неподдерживаемый формат %q", format)
	}
	return p.Parse(data)
}
