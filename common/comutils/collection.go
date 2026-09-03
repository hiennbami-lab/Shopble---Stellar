package comutils

import "sort"

func MapToList[K comparable, V any, O any](m map[K]V, fn func(K, V) O) []O {
	list := make([]O, len(m))
	idx := 0
	for k, v := range m {
		output := fn(k, v)
		list[idx] = output
		idx++
	}
	return list
}

func ListToMap[A any, B any, C comparable](arr []A, fn func(A) (key C, val B)) map[C]B {
	m := make(map[C]B)
	for _, v := range arr {
		key, val := fn(v)
		m[key] = val
	}
	return m
}

func ListToMapArray[A any, B any, C comparable](arr []A, fn func(A) (key C, val B)) map[C][]B {
	result := make(map[C][]B)
	for _, item := range arr {
		k, v := fn(item)
		result[k] = append(result[k], v)
	}
	return result
}

func ToList[A any, B any](arr []A, fn func(A) B) []B {
	list := make([]B, len(arr))
	for idx := range arr {
		list[idx] = fn(arr[idx])
	}
	return list
}

func ToListSafe[A any, B any](arr []A, fn func(A) (B, error)) (_ []B, err error) {
	list := make([]B, len(arr))
	for idx := range arr {
		var value B
		value, err = fn(arr[idx])
		if err != nil {
			return
		}
		list[idx] = value
	}
	return list, nil
}

func SortBy[T any](slice []T, less func(a, b T) bool) {
	sort.Slice(slice, func(i, j int) bool {
		return less(slice[i], slice[j])
	})
}
