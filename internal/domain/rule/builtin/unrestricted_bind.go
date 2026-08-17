package builtin

import (
	"net"
	"strings"

	"configguard/internal/domain/config"
	"configguard/internal/domain/issue"
)

// UnrestrictedBindRule находит биндинг на все интерфейсы без ограничений.
type UnrestrictedBindRule struct{}

const unrestrictedBindID = "unrestricted-bind"

var bindKeys = map[string]struct{}{
	"host": {}, "bind": {}, "address": {}, "listen": {}, "interface": {},
}

var unrestrictedAddresses = map[string]struct{}{
	"0.0.0.0": {}, "::": {}, "[::]": {},
}

func (UnrestrictedBindRule) ID() string { return unrestrictedBindID }

func (UnrestrictedBindRule) Check(cfg *config.Config) []issue.Issue {
	var found []issue.Issue
	cfg.Walk(func(n config.Node) {
		key := strings.ToLower(n.Key)
		if _, ok := bindKeys[key]; !ok {
			return
		}
		s, ok := config.AsString(n.Value)
		if !ok {
			return
		}
		host := s
		if h, _, err := net.SplitHostPort(s); err == nil {
			host = h
		}
		if _, unrestricted := unrestrictedAddresses[host]; !unrestricted {
			return
		}
		found = append(found, newIssue(unrestrictedBindID, issue.Medium, n.DotPath(),
			"сервис слушает на "+s+" без ограничений по интерфейсам.",
			"Ограничьте адрес прослушивания конкретным интерфейсом или добавьте firewall-правила."))
	})
	return found
}
