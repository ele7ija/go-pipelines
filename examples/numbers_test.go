package examples

import (
	"testing"
)

func BenchmarkDoConcurrentApi(b *testing.B) {

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		DoConcurrentApi()
	}
}

func BenchmarkDoConcurrentSimpleApi(b *testing.B) {

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		DoConcurrentSimpleApi()
	}
}
