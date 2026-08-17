package grpcapi

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"configguard/internal/application/scanner"
	pb "configguard/internal/genproto/configguard/v1"
)

type Server struct {
	addr    string
	grpcSrv *grpc.Server
}

func NewServer(svc *scanner.Service, addr string) *Server {
	grpcSrv := grpc.NewServer()
	pb.RegisterScannerServiceServer(grpcSrv, &handler{svc: svc})
	// reflection нужен для grpcurl/grpcui без .proto под рукой (dev/debug).
	reflection.Register(grpcSrv)
	return &Server{addr: addr, grpcSrv: grpcSrv}
}

// Run запускает сервер и блокируется до отмены ctx, после чего выполняет
// graceful stop: даёт активным RPC до 10 секунд на завершение, затем
// прерывает принудительно.
func (s *Server) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("grpcapi: не удалось начать слушать %s: %w", s.addr, err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- s.grpcSrv.Serve(lis) }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Printf("[grpc] получен сигнал остановки, начинаю graceful shutdown (таймаут 10s)")
		stopped := make(chan struct{})
		go func() {
			s.grpcSrv.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
			log.Printf("[grpc] graceful shutdown завершён")
		case <-time.After(10 * time.Second):
			log.Printf("[grpc] таймаут graceful shutdown, принудительная остановка")
			s.grpcSrv.Stop()
		}
		return nil
	}
}
