package comretry

import "github.com/cenkalti/backoff/v4"

// Retry decision options
type RetryDecision byte

const (
	_ RetryDecision = iota
	RetryDecisionYes
	RetryDecisionIgnore
	RetryDecisionRaise
)

// Default backoff option
func NewDefaultBackoffOpts() *backoff.ExponentialBackOff {
	return &backoff.ExponentialBackOff{
		InitialInterval:     backoff.DefaultInitialInterval,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         backoff.DefaultMaxInterval,
		MaxElapsedTime:      backoff.DefaultMaxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
}
