package scanner

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
	"configguard/internal/domain/rule"
	"configguard/internal/infrastructure/parser"
)

// Service — единственная точка входа для проверки конфига: файл, stdin или
// байты из HTTP/gRPC запроса приходят к одному и тому же движку правил.
type Service struct {
	rules   *rule.Registry
	parsers *parser.Registry
}

func NewService(rules *rule.Registry, parsers *parser.Registry) *Service {
	return &Service{rules: rules, parsers: parsers}
}

// ScanFile читает файл с диска и прогоняет его через все правила, включая
// FileAwareRule (нужны реальные права доступа файла).
func (s *Service) ScanFile(path string) (Report, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Report{}, fmt.Errorf("scanner: не удалось получить информацию о файле: %w", err)
	}
	if info.IsDir() {
		return Report{}, fmt.Errorf("scanner: %q — директория, используйте рекурсивное сканирование", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, fmt.Errorf("scanner: не удалось прочитать файл: %w", err)
	}

	cfg, err := s.parsers.Parse(data, parser.DetectByExt(path))
	if err != nil {
		return Report{}, err
	}

	return Report{Source: path, Issues: s.check(cfg, info, path)}, nil
}

// ScanReader проверяет конфиг, прочитанный из произвольного io.Reader
// (stdin, тело HTTP/gRPC запроса). FileAwareRule здесь не запускаются —
// у потока нет прав доступа на диске.
func (s *Service) ScanReader(r io.Reader, source string, format parser.Format) (Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Report{}, fmt.Errorf("scanner: не удалось прочитать данные: %w", err)
	}

	cfg, err := s.parsers.Parse(data, format)
	if err != nil {
		return Report{}, err
	}

	return Report{Source: source, Issues: s.check(cfg, nil, "")}, nil
}

func (s *Service) check(cfg *config.Config, info fs.FileInfo, path string) []issue.Issue {
	var found []issue.Issue
	for _, r := range s.rules.Rules() {
		found = append(found, r.Check(cfg)...)
		if info == nil {
			continue
		}
		if fa, ok := r.(rule.FileAwareRule); ok {
			found = append(found, fa.CheckFile(info, path)...)
		}
	}
	issue.SortBySeverity(found)
	return found
}
