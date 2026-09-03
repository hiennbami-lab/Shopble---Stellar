package comtypes

type KeyValue[K comparable, V any] map[K]V

func (src KeyValue[K, V]) Merge(target map[K]V) KeyValue[K, V] {
	mergedSrc := make(map[K]V)
	for key, val := range src {
		mergedSrc[key] = val
	}
	for key, val := range target {
		mergedSrc[key] = val
	}
	return mergedSrc
}
