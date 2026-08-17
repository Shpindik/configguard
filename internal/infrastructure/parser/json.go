package parser

import (
	"encoding/json"
	"fmt"

	"configguard/internal/domain/config"
)

type JSONParser struct{}

func (JSONParser) Format() Format { return JSON }

func (JSONParser) Parse(data []byte) (*config.Config, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parser: некорректный JSON: %w", err)
	}
	return config.New(root, data), nil
}
