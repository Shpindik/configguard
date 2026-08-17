// Package cli собирает cobra-команды поверх application/scanner. Слой
// интерфейсов — единственное место, где приложение решает про os.Exit,
// stdout/stderr и разбор аргументов командной строки.
package cli

import (
	"github.com/spf13/cobra"

	"configguard/internal/application/scanner"
)

// NewRootCmd строит корневую команду со всеми подкомандами.
func NewRootCmd(svc *scanner.Service) *cobra.Command {
	root := &cobra.Command{
		Use:           "configguard",
		Short:         "Анализ конфигов веб-приложений на опасные настройки",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newScanCmd(svc))
	root.AddCommand(newServeCmd(svc))
	return root
}
