package examples

import (
	"testing"
)

func BenchmarkDoConcurrentApi(b *testing.B) {

	for n := 0; n < b.N; n++ {
		DoConcurrentApi()
	}
}

func BenchmarkDoConcurrentSimpleApi(b *testing.B) {

	for n := 0; n < b.N; n++ {
		DoConcurrentSimpleApi()
	}
}
