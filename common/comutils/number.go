package comutils

type (
	NumberSignedInteger interface {
		int | int8 | int16 | int32 | int64
	}

	NumberUnsignedInteger interface {
		uint | uint8 | uint16 | uint32 | uint64
	}

	NumberInteger interface {
		NumberSignedInteger | NumberUnsignedInteger
	}

	NumberReal interface {
		float32 | float64
	}

	Number interface {
		NumberInteger | NumberReal
	}
)

func MinNumber[T Number](firstN T, numbers ...T) T {
	minN := firstN
	for _, n := range numbers {
		if n < minN {
			minN = n
		}
	}
	return minN
}

func MaxNumber[T Number](firstN T, numbers ...T) T {
	maxN := firstN
	for _, n := range numbers {
		if n > maxN {
			maxN = n
		}
	}
	return maxN
}
