package comtypes

type ISetOrder[K comparable, V any] interface {
	Add(key K, value V)
	Contains(key K) (int, bool)
	ContainsAndGet(key K) (V, bool)
	AsList() []struct {
		Key   K
		Value V
	}
	Entries() Entry[K, V]
	AddOrReplace(key K, value V)
	Remove(key K)
}

type Entry[K comparable, V any] struct {
	keys   []K
	values []V
}

func (e Entry[K, V]) Keys() []K {
	return e.keys
}

func (e Entry[K, V]) Values() []V {
	return e.values
}

type SetOrder[K comparable, V any] map[int]struct {
	Key   K
	Value V
}

type SetOrderSize[K comparable, V any] struct {
	SetOrder[K, V]
	size int
}

func (m SetOrderSize[K, V]) len() int {
	return m.size
}

func NewSetOrder[K comparable, V any](len ...int) ISetOrder[K, V] {
	for idx := range len {
		if len[idx] > 0 {
			return &SetOrderSize[K, V]{
				SetOrder: make(SetOrder[K, V], len[idx]),
				size:     len[idx],
			}
		}
	}
	return make(SetOrder[K, V])
}

func (m SetOrderSize[K, V]) Add(key K, value V) {
	switch {
	case m.len() == -1:
		break
	case len(m.SetOrder) > m.len():
		return
	}
	m.SetOrder.Add(key, value)
}

func (m SetOrder[K, V]) Add(key K, value V) {
	if _, ok := m.Contains(key); ok {
		return
	}
	m[len(m)] = struct {
		Key   K
		Value V
	}{Key: key, Value: value}
}

func (m SetOrder[K, V]) AddOrReplace(key K, value V) {
	idx, ok := m.Contains(key)
	if ok {
		m[idx] = struct {
			Key   K
			Value V
		}{Key: key, Value: value}
	} else {
		m.Add(key, value)
	}
}

func (m SetOrder[K, V]) Remove(key K) {
	for idx := range m {
		if m[idx].Key == key {
			delete(m, idx)
			return
		}
	}
}

func (m SetOrder[K, V]) Contains(key K) (int, bool) {
	for idx, v := range m {
		if v.Key == key {
			return idx, true
		}
	}
	return -1, false
}

func (m SetOrder[K, V]) ContainsAndGet(key K) (V, bool) {
	for _, v := range m {
		if v.Key == key {
			return v.Value, true
		}
	}
	return *new(V), false
}

func (m SetOrder[K, V]) AsList() []struct {
	Key   K
	Value V
} {
	list := make([]struct {
		Key   K
		Value V
	}, len(m))
	for idx := 0; idx < len(m); idx++ {
		list[idx] = m[idx]
	}
	return list
}

func (m SetOrder[K, V]) Entries() Entry[K, V] {
	var (
		keys   = make([]K, len(m))
		values = make([]V, len(m))
	)
	for idx := 0; idx < len(m); idx++ {
		keys[idx] = m[idx].Key
		values[idx] = m[idx].Value
	}
	return Entry[K, V]{
		keys:   keys,
		values: values,
	}
}
