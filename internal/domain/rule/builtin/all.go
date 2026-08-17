package builtin

import "configguard/internal/domain/rule"

// All возвращает полный набор встроенных правил. Единственное место, где
// нужно зарегистрировать новое правило.
func All() []rule.Rule {
	return []rule.Rule{
		DebugLoggingRule{},
		PlaintextPasswordRule{},
		UnrestrictedBindRule{},
		TLSDisabledRule{},
		WeakAlgorithmRule{},
		PermissiveFileModeRule{},
		FilePermissionRule{},
	}
}
