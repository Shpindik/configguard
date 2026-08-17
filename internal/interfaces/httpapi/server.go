// Package httpapi реализует REST API поверх application/scanner.
package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"configguard/internal/application/scanner"
)

type Server struct {
	svc     *scanner.Service
	httpSrv *http.Server
}

func NewServer(svc *scanner.Service, addr string) *Server {
	s := &Server{svc: svc}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/scan", s.handleScan)
	mux.HandleFunc("GET /healthz", s.handleHealth)

	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// Handler возвращает http.Handler сервера — используется в тестах поверх
// httptest.NewServer, без реального сетевого прослушивания.
func (s *Server) Handler() http.Handler {
	return s.httpSrv.Handler
}

// Run запускает сервер и блокируется до отмены ctx, после чего выполняет
// graceful shutdown: даёт активным запросам до 10 секунд на завершение.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- s.httpSrv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Printf("[http] получен сигнал остановки, начинаю graceful shutdown (таймаут 10s)")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[http] graceful shutdown завершился с ошибкой: %v", err)
			return err
		}
		log.Printf("[http] graceful shutdown завершён")
		return nil
	}
}
