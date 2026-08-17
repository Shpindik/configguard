// Command configguard — CLI/HTTP/gRPC утилита анализа конфигов на опасные
// настройки. main только собирает зависимости и запускает cobra-команду.
package main

import (
	"fmt"
	"os"

	"configguard/internal/application/scanner"
	"configguard/internal/domain/rule"
	"configguard/internal/domain/rule/builtin"
	"configguard/internal/infrastructure/parser"
	"configguard/internal/interfaces/cli"
)

func main() {
	rules := rule.NewRegistry(builtin.All()...)
	parsers := parser.NewRegistry(parser.JSONParser{}, parser.YAMLParser{})
	svc := scanner.NewService(rules, parsers)

	if err := cli.NewRootCmd(svc).Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
