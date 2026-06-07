package divbench

import "testing"

var strides = []int{4, 7, 13, 31, 64, 127, 200, 256}
var divisors = [8]int{4, 7, 13, 31, 64, 127, 200, 256}

var sink int

//go:noinline
func divInt(sum, stride int) int { return (256*sum + stride/2) / stride }

//go:noinline
func divFloat(sum, stride int) int { return int(float64(256*sum+stride/2) / float64(stride)) }

func BenchmarkDiv(b *testing.B) {
	b.Run("div=idivq", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			acc += divInt(i&0xffff, strides[i&7])
			i++
		}
		sink = acc
	})
	b.Run("div=float", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			acc += divFloat(i&0xffff, strides[i&7])
			i++
		}
		sink = acc
	})
}

func BenchmarkDivInline(b *testing.B) {
	b.Run("div=idivq", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			stride := divisors[i&7]
			acc += (256*(i&0xffff) + stride/2) / stride
			i++
		}
		sink = acc
	})
	b.Run("div=float", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			stride := divisors[i&7]
			acc += int(float64(256*(i&0xffff)+stride/2) / float64(stride))
			i++
		}
		sink = acc
	})
}
