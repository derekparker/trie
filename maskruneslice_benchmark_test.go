package trie

import (
	"testing"
)

// Helper function to avoid optimization of benchmark results
var result uint64

func BenchmarkMaskRuneSliceEmpty(b *testing.B) {
	rs := []rune("")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}

func BenchmarkMaskRuneSliceShort(b *testing.B) {
	rs := []rune("abc")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}



func BenchmarkMaskRuneSliceMedium(b *testing.B) {
	rs := []rune("abcdefghijk")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}



func BenchmarkMaskRuneSliceLong(b *testing.B) {
	rs := []rune("abcdefghijklmnopqrstuvwxyz")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}



func BenchmarkMaskRuneSliceWord(b *testing.B) {
	rs := []rune("hello")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}



func BenchmarkMaskRuneSliceRepeated(b *testing.B) {
	rs := []rune("aaaaa")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}



func BenchmarkMaskRuneSliceRandom(b *testing.B) {
	rs := []rune("qwertyuiop")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}



// Benchmark with mixed case and non-ASCII characters
func BenchmarkMaskRuneSliceMixed(b *testing.B) {
	rs := []rune("Hello世界")
	b.ResetTimer()
	var r uint64
	for i := 0; i < b.N; i++ {
		r = maskruneslice(rs)
	}
	result = r
}
