package comqueue

import (
	"context"
	"fmt"
	"shopble/common/comerr"
	"shopble/common/comlog"
	"shopble/common/comretry"
	"runtime/debug"
	"time"

	"cloud.google.com/go/logging"
)

var listConsumer []*OurConsumer

type Subcriber struct {
	Name        string
	ExecuteFunc func(ctx context.Context, msg []byte) error
}
type ConsumeFuncMap map[string]Subcriber

type Option struct {
	Retryer comretry.Retryer
}

type OurConsumer struct {
	consumer   Consumer
	retryer    comretry.Retryer
	cancelFunc context.CancelFunc
}

func NewOurConsumer(creator ConsumerCreator, groupID string, topics []string, option *Option) (*OurConsumer, error) {
	if option.Retryer == nil {
		option.Retryer = &comretry.DefaultRetryer{}
	}
	consumer, err := creator(topics, groupID)
	if err != nil {
		return nil, err
	}
	ourConsumer := &OurConsumer{
		consumer:   consumer,
		retryer:    option.Retryer,
		cancelFunc: nil,
	}
	listConsumer = append(listConsumer, ourConsumer)
	return ourConsumer, nil
}

func (c *OurConsumer) Start(pollTimeout time.Duration, consumeFuncMap ConsumeFuncMap) error {
	if c.cancelFunc != nil {
		return fmt.Errorf("consumer has already started!")
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel
	go c.run(ctx, pollTimeout, consumeFuncMap)
	return nil
}

func (c *OurConsumer) Close() {
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
}

func (c *OurConsumer) run(ctx context.Context, pollTimeout time.Duration, consumerFuncMap ConsumeFuncMap) {
	ourLog := comlog.GetLog()
	defer func() {
		err := recover()
		if err == nil {
			return
		} else {
			defer func() {
				_ = c.consumer.CommitMessage(ctx)
			}()
			var (
				typeErr error
				ok      bool
			)
			typeErr, ok = err.(error)
			if !ok {
				typeErr = fmt.Errorf("%v", err)
			}
			ourLog.Log(logging.Entry{
				Payload: map[string]any{
					"error": typeErr,
					"stack": string(debug.Stack()),
				},
				Severity: logging.Error,
			})
		}
		panic(err)
	}()
	for {
		select {
		case <-ctx.Done():
			break
		default:
			jobname, err := c.processMessage(ctx, pollTimeout, consumerFuncMap)
			if err == nil {
				continue
			} else {
				if err == context.Canceled {
					break
				}
				ourError, ok := err.(comerr.Failure)
				if !ok {
					continue
				}
				fmt.Printf("Skip consuming msg on job<%s>\n", jobname)
				_ = c.consumer.CommitMessage(ctx)
				ourLog.Log(logging.Entry{
					Payload: map[string]any{
						"type":       "consumeFunc",
						"jobname":    jobname,
						"error":      ourError.Error(),
						"stacktrace": ourError.Stack(),
						"data":       ourError.Data().String(),
					},
					Severity: logging.Error,
				})
			}
		}
	}
}

func (c *OurConsumer) processMessage(ctx context.Context, pollTimeout time.Duration, consumeFuncMap ConsumeFuncMap) (jobaname string, err error) {
	ctx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()
	msg, err := c.consumer.FetchMessage(ctx)
	if err != nil {
		if err == context.DeadlineExceeded {
			err = nil
		}
		return
	}
	subConsume, ok := consumeFuncMap[msg.Key]
	if ok {
		fmt.Printf("Consumeing msg on job<%s>...\n", subConsume.Name)
		err = comretry.Execute(ctx, msg.Raw, subConsume.ExecuteFunc, c.retryer)
		if err != nil {
			return subConsume.Name, err
		}
		fmt.Printf("Finished consuming msg on job<%s>\n", subConsume.Name)
	}
	if err = c.consumer.CommitMessage(ctx); err != nil {
		return subConsume.Name, err
	}
	return
}
