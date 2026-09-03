package comrunner

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type Service interface {
	Name() string
	BeforeStop()
	Start() error
	Stop(ctx context.Context) error
}

func RunSession(session *Session) {
	var quitChan = make(chan os.Signal, 1)
	for i := range session.Services {
		service := session.Services[i]
		go func() {
			session.Log.Info("service `%v` has started\n", service.Name())
			err := service.Start()
			if err != nil {
				session.Log.Fatal("service `%v` run failed | err=%s\n", service.Name(), err.Error())
			}
		}()
	}
	signal.Notify(quitChan, os.Interrupt, syscall.SIGTERM)
	<-quitChan
	for i := range session.Services {
		session.Services[i].BeforeStop()
	}
	session.Log.Info("servies are shutting down...\n")
	ctx, cancel := context.WithTimeout(context.Background(), session.Option.GracefulTimeout)
	defer cancel()
	var (
		errMap      = session.End(ctx)
		stopOkCount = 0
	)
	for _, service := range session.Services {
		svcName := service.Name()
		if err, hasError := errMap[svcName]; !hasError {
			session.Log.Info("service `%v` has stopped\n", svcName)
			stopOkCount++
			continue
		} else {
			session.Log.Error("service `%v` has failed to stop | err=%s\n", svcName, err.Error())
		}
	}
	if stopOkCount < len(session.Services) {
		session.Log.Info("services haven't shut down properly (%v/%v)", stopOkCount, len(session.Services))
	} else {
		session.Log.Info("services have all shut down (%v/%v).", stopOkCount, len(session.Services))
	}
}
