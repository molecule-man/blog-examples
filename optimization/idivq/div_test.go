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

var fnum [256]float64
var fden [256]float64
var fsink float64

func init() {
	x := uint64(12345)
	for i := range fnum {
		x = x*6364136223846793005 + 1442695040888963407 // LCG, runtime values
		fnum[i] = float64((x >> 40) & ((1 << 24) - 1))
		fden[i] = float64(4 + (x>>8)&255)
	}
}

func BenchmarkDivThroughput(b *testing.B) {
	acc, i := 0.0, 0
	for b.Loop() {
		acc += fnum[i&255] / fden[i&255]
		i++
	}
	fsink = acc
}

const reciprocalShift = 40

func BenchmarkDivSameDivisor(b *testing.B) {
	stride := divisors[3] // 31, a runtime value
	recip := (uint64(1)<<reciprocalShift + uint64(stride) - 1) / uint64(stride)

	b.Run("div=idivq", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			acc += (256*(i&0xffff) + stride/2) / stride
			i++
		}
		sink = acc
	})
	b.Run("div=float", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			acc += int(float64(256*(i&0xffff)+stride/2) / float64(stride))
			i++
		}
		sink = acc
	})
	// Correctness wasn't verified. Careful
	b.Run("div=reciprocal", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			acc += int((uint64(256*(i&0xffff)+stride/2) * recip) >> reciprocalShift)
			i++
		}
		sink = acc
	})
}
