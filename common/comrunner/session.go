package comrunner

import (
	"context"
	"time"

	"shopble/common/comlog"
)

type Option struct {
	GracefulTimeout time.Duration
}

func ParseOption() *Option {
	gracefulTimeout, _ := time.ParseDuration("")
	return &Option{
		GracefulTimeout: gracefulTimeout,
	}
}

type Session struct {
	Option   *Option
	Services []Service
	Log      *comlog.Logger
}

func NewSession(services ...Service) *Session {
	return &Session{
		Services: services,
		Option:   ParseOption(),
		Log:      &comlog.Logger{},
	}
}

func (s *Session) End(ctx context.Context) (errMap map[string]error) {
	errMap = make(map[string]error)
	for _, service := range s.Services {
		if err := service.Stop(ctx); err != nil {
			errMap[service.Name()] = err
		}
	}
	return
}
