package imghash

import "testing"

func BenchmarkDHash_ComputeImage(b *testing.B) {
	img := createGradientImage(256, 256, false)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ComputeImage(img)
	}
}

func BenchmarkDHash_HammingDistance(b *testing.B) {
	h1 := "8f00ff00ff00ff00"
	h2 := "8f00ff00ff00ff01"
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = Distance(h1, h2)
	}
}

func BenchmarkDHash_Similar(b *testing.B) {
	h1 := "8f00ff00ff00ff00"
	h2 := "8f00ff00ff00ff01"
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Similar(h1, h2, 5)
	}
}

func BenchmarkDHash_ComputeImage_RawUint64(b *testing.B) {
	d := NewDHash()
	img := createGradientImage(256, 256, false)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = d.ComputeFromImage(img)
	}
}

