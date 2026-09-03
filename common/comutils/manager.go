package comutils

func Contains[T comparable](src []T, target T) bool {
	for _, v := range src {
		if v == target {
			return true
		}
	}
	return false
}
