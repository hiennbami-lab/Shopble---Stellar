package comtypes

type HashSet[T comparable] map[T]struct{}

func NewHashSet[T comparable](items ...T) HashSet[T] {
	return NewHashSetFromList(items)
}

func NewHashSetFromList[T comparable](items []T) HashSet[T] {
	hashSet := make(HashSet[T], len(items))
	for i := 0; i < len(items); i++ {
		hashSet.Add(items[i])
	}
	return hashSet
}

func (s HashSet[T]) Add(v T) bool {
	if s.Contains(v) {
		return false
	}
	s[v] = struct{}{}
	return true
}

func (s HashSet[T]) Remove(v T) bool {
	if !s.Contains(v) {
		return false
	}
	delete(s, v)
	return true
}

func (s HashSet[T]) Contains(v T) bool {
	_, exists := s[v]
	return exists
}

func (s HashSet[T]) AsList() []T {
	items := make([]T, 0, len(s))
	for item := range s {
		items = append(items, item)
	}
	return items
}
