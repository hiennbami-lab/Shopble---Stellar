package comrunner

import (
	"context"
	"fmt"
	"net/http"
)

type httpsvc struct {
	*http.Server
}

func NewHttpService(s *http.Server) Service {
	return &httpsvc{s}
}

func (s *httpsvc) Name() string {
	return "http_svc"
}

func (s *httpsvc) Start() error {
	err := s.Server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http: strat server: %w", err)
	}
	return nil
}

func (s *httpsvc) Stop(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

func (s *httpsvc) BeforeStop() {
}
