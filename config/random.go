package config

import (
	"math/big"
	"math/rand"

	"shopble/common/comutils"

	"github.com/google/uuid"
	"github.com/segmentio/ksuid"
)

func InitRandom() {
	var (
		seedBytes = comutils.RandomBytesF(8)
		seed      = new(big.Int).SetBytes(seedBytes)
		seedInt64 = seed.Int64()
	)
	rand.Seed(seedInt64)
	ksuid.SetRand(rand.New(rand.NewSource(seedInt64)))
	uuid.SetRand(rand.New(rand.NewSource(seedInt64)))
}
