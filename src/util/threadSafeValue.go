package util

import "sync"

type SafeValue[T any] struct {
	v T
	sync.Mutex
}

func (sv *SafeValue[T]) Do(f func(t T)) {
	sv.Lock()
	defer sv.Unlock()
	f(sv.v)
}

func (sv *SafeValue[T]) Set(t T) {
	sv.Lock()
	defer sv.Unlock()
	sv.v = t
}
