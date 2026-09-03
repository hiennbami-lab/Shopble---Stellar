package comlock

import "errors"

var (
	OurLockErrorInvalidRedisDb = errors.New("invalid redis database")
)
