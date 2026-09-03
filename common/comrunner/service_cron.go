package comrunner

import (
	"context"
	"shopble/common/comutils"
	"time"

	"github.com/robfig/cron/v3"
)

type (
	CronTask struct {
		Spec string
		Task func()
	}
	ListCronTask []CronTask
)

type cronsvc struct {
	cron *cron.Cron
	list ListCronTask
}

func (c *cronsvc) Name() string {
	return "cron"
}

func (c *cronsvc) Start() error {
	for _, cronTask := range c.list {
		_, err := c.cron.AddFunc(cronTask.Spec, cronTask.Task)
		comutils.PanicOnError(err)
	}
	c.cron.Start()
	return nil
}

func (c *cronsvc) Stop(ctx context.Context) error {
	c.cron.Stop()
	return nil
}

func (c *cronsvc) BeforeStop() {}

func NewCronService(funcMap ListCronTask) Service {
	return &cronsvc{
		cron: cron.New(
			cron.WithLocation(time.UTC),
			cron.WithLogger(cron.DefaultLogger),
			cron.WithChain(
				cron.SkipIfStillRunning(cron.DefaultLogger),
			),
		),
		list: funcMap,
	}
}
