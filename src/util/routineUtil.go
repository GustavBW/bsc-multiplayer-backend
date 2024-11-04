package util

import "time"

func Delay(ms int, function func()) {
	go func() {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		function()
	}()
}

func DelayBlockingly[T any](ms int, function func() T) T {
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return function()
}
