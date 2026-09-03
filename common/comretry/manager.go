package comretry

import "context"

type Retryer interface {
	Reset()
	Decide() RetryDecision
	MaxRetry() uint
	MarkError(error)
}

type DefaultRetryer struct {
}

func (d *DefaultRetryer) Decide() RetryDecision {
	return RetryDecisionIgnore
}

func (d *DefaultRetryer) MarkError(err error) {}

func (d *DefaultRetryer) MaxRetry() uint {
	return 0
}

func (d *DefaultRetryer) Reset() {}

func Execute[M any, R Retryer](
	ctx context.Context,
	msg M, executeFn func(ctx context.Context, msg M) error,
	retryer R,
) error {
	return run(ctx, msg, executeFn, retryer, 0)
}

func run[M any, R Retryer](
	ctx context.Context,
	msg M, executeFn func(ctx context.Context, msg M) error,
	retryer R, retryCount uint,
) (err error) {
	if err = executeFn(ctx, msg); err == nil {
		return
	}
	retryer.MarkError(err)
	switch retryer.Decide() {
	case RetryDecisionYes:
		if retryCount > retryer.MaxRetry() {
			break
		}
		return run(ctx, msg, executeFn, retryer, retryCount+1)
	case RetryDecisionIgnore:
		err = nil
		return
	case RetryDecisionRaise:
		fallthrough
	default:
	}
	return err
}
