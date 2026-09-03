package comretry

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

type Backoff struct {
	backoff *backoff.ExponentialBackOff
}

func NewBackoff(bo *backoff.ExponentialBackOff) *Backoff {
	if bo == nil {
		return &Backoff{
			backoff: NewDefaultBackoffOpts(),
		}
	}
	return &Backoff{
		backoff: bo,
	}
}

func (b Backoff) Reset() {
	b.backoff.Reset()
}

func (b Backoff) Decide() RetryDecision {
	wait := b.backoff.NextBackOff()
	if wait == backoff.Stop {
		return RetryDecisionRaise
	}
	time.Sleep(wait)
	return RetryDecisionYes
}
