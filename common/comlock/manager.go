package comlock

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
)

func Lock(ctx context.Context, key string, timeout, retryDelay time.Duration) (*OurMutex, error) {
	var (
		rs  = vDefaultRedsync.GetF()
		mux = rs.NewMutex(
			key,
			redsync.WithTries(int(timeout/retryDelay)),
			redsync.WithExpiry(timeout),
			redsync.WithRetryDelay(retryDelay),
		)
	)
	if err := mux.LockContext(ctx); err != nil {
		return nil, err
	}
	return mux, nil
}

func LockSimple(ctx context.Context, key string, params ...any) (*OurMutex, error) {
	if len(params) > 0 {
		key = fmt.Sprintf(key, params...)
	}
	var rs = vDefaultRedsync.GetF()
	return Lock(
		ctx,
		key,
		rs.defaultLockTimeout,
		rs.defaultRetryDelay,
	)
}
