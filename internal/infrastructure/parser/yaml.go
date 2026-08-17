package parser

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"configguard/internal/domain/config"
)

type YAMLParser struct{}

func (YAMLParser) Format() Format { return YAML }

// Parse yaml.v3 декодирует map-узлы сразу в map[string]interface{}, поэтому дерево
// совместимо с тем, что отдаёт encoding/json, без дополнительной нормализации.
// Числа могут прийти как int (в отличие от float64 у JSON) — это учтено в
// правилах, которые работают с числовыми значениями (parseFileMode).
func (YAMLParser) Parse(data []byte) (*config.Config, error) {
	var root any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parser: некорректный YAML: %w", err)
	}
	return config.New(root, data), nil
}
