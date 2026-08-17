package builtin

import (
	"io/fs"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// FilePermissionRule проверяет права доступа самого файла конфига через
// os.Stat (файл может содержать секреты, поэтому его видимость важна).
// Реализует rule.FileAwareRule — не применяется при чтении из stdin.
type FilePermissionRule struct{}

const filePermissionID = "file-permission"

func (FilePermissionRule) ID() string { return filePermissionID }

// Check — часть контракта rule.Rule. Без доступа к fs.FileInfo делать нечего.
func (FilePermissionRule) Check(*config.Config) []issue.Issue { return nil }

func (FilePermissionRule) CheckFile(info fs.FileInfo, path string) []issue.Issue {
	mode := info.Mode().Perm()

	switch {
	case mode&0o002 != 0:
		return []issue.Issue{newIssue(filePermissionID, issue.High, path,
			"файл конфига доступен на запись всем пользователям (world-writable).",
			"Установите права не шире 0640: chmod 0640 "+path+".")}
	case mode&0o004 != 0:
		return []issue.Issue{newIssue(filePermissionID, issue.Medium, path,
			"файл конфига доступен на чтение всем пользователям, а может содержать секреты.",
			"Ограничьте права доступа: chmod 0640 "+path+".")}
	}
	return nil
}
