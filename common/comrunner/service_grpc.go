package comrunner

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"google.golang.org/grpc"
)

type grpcSvc struct {
	*grpc.Server
	listener net.Listener
}

func NewGrpcService(server *grpc.Server, listener net.Listener) Service {
	return &grpcSvc{
		Server:   server,
		listener: listener,
	}
}

func (s *grpcSvc) Name() string {
	return "grpc_svc"
}

func (s *grpcSvc) Start() error {
	err := s.Server.Serve(s.listener)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http: strat server: %w", err)
	}
	return nil
}

func (s *grpcSvc) Stop(ctx context.Context) error {
	s.Server.GracefulStop()
	return nil
}

func (s *grpcSvc) BeforeStop() {
}
