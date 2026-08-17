package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"configguard/internal/application/dirscan"
	"configguard/internal/application/scanner"
	"configguard/internal/domain/issue"
	"configguard/internal/infrastructure/parser"
)

func newScanCmd(svc *scanner.Service) *cobra.Command {
	var silent bool
	var stdin bool

	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Проверить файл или директорию с конфигами",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hasIssues, err := runScan(cmd, svc, args, stdin)
			if err != nil {
				return err
			}
			if hasIssues && !silent {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&silent, "silent", "s", false, "не выходить с ошибкой при наличии проблем")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "прочитать конфигурацию из стандартного потока ввода")
	return cmd
}

func runScan(cmd *cobra.Command, svc *scanner.Service, args []string, stdin bool) (hasIssues bool, err error) {
	out := cmd.OutOrStdout()

	if stdin {
		report, err := svc.ScanReader(cmd.InOrStdin(), "stdin", parser.Unknown)
		if err != nil {
			return false, err
		}
		printReport(out, report)
		return report.HasIssues(), nil
	}

	if len(args) != 1 {
		return false, fmt.Errorf("укажите путь к файлу/директории первым аргументом или используйте --stdin")
	}
	path := args[0]

	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("не удалось открыть %q: %w", path, err)
	}

	if info.IsDir() {
		results, err := dirscan.Scan(context.Background(), svc, path, 0)
		if err != nil {
			return false, err
		}
		return printDirResults(out, results), nil
	}

	report, err := svc.ScanFile(path)
	if err != nil {
		return false, err
	}
	printReport(out, report)
	return report.HasIssues(), nil
}

func printReport(w io.Writer, r scanner.Report) {
	if !r.HasIssues() {
		_, _ = fmt.Fprintf(w, "%s: проблем не найдено\n", r.Source)
		return
	}
	_, _ = fmt.Fprintf(w, "%s:\n", r.Source)
	for _, iss := range r.Issues {
		printIssue(w, iss)
	}
}

func printIssue(w io.Writer, iss issue.Issue) {
	_, _ = fmt.Fprintf(w, "  %s\n", iss.String())
}

func printDirResults(w io.Writer, results []dirscan.Result) (hasIssues bool) {
	for _, res := range results {
		if res.Err != nil {
			_, _ = fmt.Fprintf(w, "%s: ошибка: %v\n", res.Path, res.Err)
			hasIssues = true
			continue
		}
		printReport(w, res.Report)
		hasIssues = hasIssues || res.Report.HasIssues()
	}
	return hasIssues
}
