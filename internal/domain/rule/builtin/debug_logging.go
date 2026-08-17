package builtin

import (
	"strings"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// DebugLoggingRule находит слишком подробное логирование (debug-режим).
type DebugLoggingRule struct{}

const debugLoggingID = "debug-logging"

var debugLevelKeys = map[string]struct{}{
	"level": {}, "log_level": {}, "log-level": {}, "loglevel": {},
}

func (DebugLoggingRule) ID() string { return debugLoggingID }

func (DebugLoggingRule) Check(cfg *config.Config) []issue.Issue {
	var found []issue.Issue
	cfg.Walk(func(n config.Node) {
		key := strings.ToLower(n.Key)

		if _, ok := debugLevelKeys[key]; ok {
			if s, ok := config.AsString(n.Value); ok && strings.EqualFold(s, "debug") {
				found = append(found, newIssue(debugLoggingID, issue.Low, n.DotPath(),
					"логирование в debug-режиме.",
					"Поменяйте режим на более избирательный (info+)."))
			}
			return
		}

		if key == "debug" {
			if b, ok := config.AsBool(n.Value); ok && b {
				found = append(found, newIssue(debugLoggingID, issue.Low, n.DotPath(),
					"debug-режим включён.",
					"Отключите debug в продакшене."))
			}
		}
	})
	return found
}
