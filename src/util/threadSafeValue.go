package util

import "sync"

type SafeValue[T any] struct {
	v T
	sync.Mutex
}

// Gain access breifly through a callback function to a pointer to the value
// For mutation and other operations. However, kindly do not store
// the pointer for later use, as only through this function is access thread safe.
// You may copy or otherwise safely store the value or part of it.
func (sv *SafeValue[T]) Do(f func(t *T)) {
	sv.Lock()
	defer sv.Unlock()
	f(&sv.v)
}

// Non-blocking version of Do, however, this function initiates a new go routine to run the provided
// function whenever it becomes available.
func (sv *SafeValue[T]) DoLater(f func(t *T)) {
	go sv.Do(f)
}

func (sv *SafeValue[T]) Set(t T) {
	sv.Lock()
	defer sv.Unlock()
	sv.v = t
}
