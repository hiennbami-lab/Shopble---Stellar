package comtypes

import (
	"fmt"
	"sync"
)

type Singleton[T any] struct {
	mux        sync.Mutex
	isLoaded   bool
	instance   T
	loader     func() T
	safeLoader func() (T, error)
}

func NewSingleton[T any](loader func() T) *Singleton[T] {
	return &Singleton[T]{
		isLoaded: false,
		loader:   loader,
	}
}

func NewSingletonSafe[T any](loader func() (T, error)) *Singleton[T] {
	return &Singleton[T]{
		isLoaded:   false,
		safeLoader: loader,
	}
}

func (s *Singleton[T]) GetF() T {
	instance, err := s.Get()
	if err != nil {
		panic(err)
	}
	return instance
}

func (s *Singleton[T]) Get() (_ T, err error) {
	if s.isLoaded {
		return s.instance, nil
	}
	s.mux.Lock()
	defer s.mux.Unlock()

	if !s.isLoaded {
		if s.safeLoader != nil {
			s.instance, err = s.safeLoader()
			if err != nil {
				return
			}
		} else {
			s.instance = s.loader()
		}
		s.isLoaded = true
	}
	return s.instance, nil
}

type SingletonMap[T any] struct {
	mux         sync.Mutex
	instanceMap map[string]T
	loader      func(key fmt.Stringer) T
}

func NewSingletonMap[T any](loader func(key fmt.Stringer) T) *SingletonMap[T] {
	return &SingletonMap[T]{
		loader:      loader,
		instanceMap: make(map[string]T),
	}
}

func (s *SingletonMap[T]) Get(key fmt.Stringer) T {
	textKey := key.String()
	if value, ok := s.instanceMap[textKey]; ok {
		return value
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	value, ok := s.instanceMap[textKey]
	if !ok {
		value = s.loader(key)
		s.instanceMap[textKey] = value
	}
	return value
}
