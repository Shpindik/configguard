package builtin

import (
	"strconv"
	"strings"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// PermissiveFileModeRule находит слишком широкие права доступа, заданные
// прямо в конфиге (например, режим для загружаемых файлов/директорий).
type PermissiveFileModeRule struct{}

const permissiveFileModeID = "permissive-file-mode"

var fileModeKeys = map[string]struct{}{
	"mode": {}, "permissions": {}, "file_mode": {}, "filemode": {}, "chmod": {}, "dir_mode": {},
}

func (PermissiveFileModeRule) ID() string { return permissiveFileModeID }

func (PermissiveFileModeRule) Check(cfg *config.Config) []issue.Issue {
	var found []issue.Issue
	cfg.Walk(func(n config.Node) {
		key := strings.ToLower(n.Key)
		if _, ok := fileModeKeys[key]; !ok {
			return
		}
		mode, ok := parseFileMode(n.Value)
		if !ok {
			return
		}

		switch {
		case mode&0o002 != 0:
			found = append(found, newIssue(permissiveFileModeID, issue.High, n.DotPath(),
				"права доступа разрешают запись всем (world-writable).",
				"Ограничьте права до 0640/0750 — запись должна быть только у владельца/группы."))
		case mode&0o020 != 0:
			found = append(found, newIssue(permissiveFileModeID, issue.Medium, n.DotPath(),
				"права доступа разрешают запись группе.",
				"Сузьте права, если группа не должна изменять эти файлы."))
		}
	})
	return found
}

// parseFileMode приводит значение поля к числовым битам прав доступа.
// Строки трактуются как восьмеричное число ("0777"/"777"/"0o777"), числа —
// как уже готовое значение битовой маски.
func parseFileMode(v any) (uint32, bool) {
	switch val := v.(type) {
	case string:
		s := strings.TrimPrefix(strings.TrimSpace(val), "0o")
		if s == "" {
			return 0, false
		}
		n, err := strconv.ParseUint(s, 8, 32)
		if err != nil {
			return 0, false
		}
		return uint32(n), true
	case float64:
		return uint32(val), true
	case int:
		return uint32(val), true
	default:
		return 0, false
	}
}
