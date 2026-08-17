package cli

import (
	"errors"
	"fmt"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"configguard/internal/application/scanner"
	"configguard/internal/interfaces/grpcapi"
	"configguard/internal/interfaces/httpapi"
)

func newServeCmd(svc *scanner.Service) *cobra.Command {
	var httpAddr string
	var grpcAddr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Запустить HTTP и/или gRPC сервер проверки конфигов",
		RunE: func(cmd *cobra.Command, args []string) error {
			if httpAddr == "" && grpcAddr == "" {
				return errors.New("укажите хотя бы один адрес: --http и/или --grpc")
			}
			return runServe(cmd, svc, httpAddr, grpcAddr)
		},
	}

	cmd.Flags().StringVar(&httpAddr, "http", "", "адрес HTTP REST API, например :8080")
	cmd.Flags().StringVar(&grpcAddr, "grpc", "", "адрес gRPC API, например :9090")
	return cmd
}

// runServe поднимает выбранные серверы и обеспечивает graceful shutdown:
// по SIGINT/SIGTERM оба сервера параллельно получают команду остановиться
// в пределах отведённого таймаута, не обрывая уже начатые запросы.
func runServe(cmd *cobra.Command, svc *scanner.Service, httpAddr, grpcAddr string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	out := cmd.OutOrStdout()

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	if httpAddr != "" {
		srv := httpapi.NewServer(svc, httpAddr)
		wg.Go(func() {
			_, _ = fmt.Fprintf(out, "HTTP API слушает на %s\n", httpAddr)
			if err := srv.Run(ctx); err != nil {
				errs <- fmt.Errorf("http: %w", err)
			}
		})
	}

	if grpcAddr != "" {
		srv := grpcapi.NewServer(svc, grpcAddr)
		wg.Go(func() {
			_, _ = fmt.Fprintf(out, "gRPC API слушает на %s\n", grpcAddr)
			if err := srv.Run(ctx); err != nil {
				errs <- fmt.Errorf("grpc: %w", err)
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
