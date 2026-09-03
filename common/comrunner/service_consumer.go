package comrunner

import "context"

type ConsumeCloseFuncs []func()

func (c *ConsumeCloseFuncs) Register(closeFunc func()) {
	*c = append(*c, closeFunc)
}

type ConsumerService struct {
	ConsumerLoader func(closeFuncs *ConsumeCloseFuncs) error
	closeFuncs     ConsumeCloseFuncs
}

func (c ConsumerService) Name() string {
	return "consumer"
}

func (c *ConsumerService) Start() error {
	c.closeFuncs = make(ConsumeCloseFuncs, 0)
	return c.ConsumerLoader(&c.closeFuncs)
}

func (c ConsumerService) Stop(ctx context.Context) error {
	for _, safeClose := range c.closeFuncs {
		safeClose()
	}
	return nil
}

func (c ConsumerService) BeforeStop() {}
