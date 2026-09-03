package comlock

import (
	"context"

	"github.com/go-redsync/redsync/v4"
)

type OurMutex struct {
	*redsync.Mutex
}

func NewMutex(mux *redsync.Mutex) *OurMutex {
	return &OurMutex{
		Mutex: mux,
	}
}

func (m *OurMutex) Lock(ctx context.Context) (err error) {
	err = m.Mutex.LockContext(ctx)
	if err != nil {
		return
	}
	return nil
}

func (m *OurMutex) Unlock(ctx context.Context) (ok bool, err error) {
	ok, err = m.Mutex.UnlockContext(ctx)
	if err != nil {
		return
	}
	return ok, nil
}
