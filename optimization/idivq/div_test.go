package divbench

import (
	"testing"

	"github.com/bmkessler/fastdiv"
)

var strides = []int{4, 7, 13, 31, 64, 127, 200, 256}
var divisors = [8]int{4, 7, 13, 31, 64, 127, 200, 256}

var sink int

var recipTable = func() [8]uint64 {
	var t [8]uint64
	for i, d := range divisors {
		t[i] = (uint64(1)<<reciprocalShift + uint64(d) - 1) / uint64(d)
	}
	return t
}()

var fastInv = func() [8]fastdiv.Uint32 {
	var t [8]fastdiv.Uint32
	for i, d := range divisors {
		t[i] = fastdiv.NewUint32(uint32(d))
	}
	return t
}()

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
	b.Run("div=reciprocal", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			stride := divisors[i&7]
			acc += int((uint64(256*(i&0xffff)+stride/2) * recipTable[i&7]) >> reciprocalShift)
			i++
		}
		sink = acc
	})
	b.Run("div=fastdiv", func(b *testing.B) {
		acc, i := 0, 0
		for b.Loop() {
			stride := divisors[i&7]
			n := uint32(256*(i&0xffff) + stride/2)
			acc += int(fastInv[i&7].Div(n))
			i++
		}
		sink = acc
	})
	b.Run("div=split", func(b *testing.B) {
		accInt, accFloat, i := 0, 0, 0
		for b.Loop() {
			s0 := divisors[i&7]
			s1 := divisors[(i+1)&7]
			accInt += (256*(i&0xffff) + s0/2) / s0                          // IDIVQ, port 1
			accFloat += int(float64(256*((i+1)&0xffff)+s1/2) / float64(s1)) // DIVSD, port 0
			i += 2
		}
		sink = accInt + accFloat
	})
	b.Run("div=split-uneven", func(b *testing.B) {
		accInt, accFloat, i := 0, 0, 0
		for b.Loop() {
			s0 := divisors[i&7]
			s1 := divisors[(i+1)&7]
			s2 := divisors[(i+2)&7]
			s3 := divisors[(i+3)&7]
			s4 := divisors[(i+4)&7]
			accInt += (256*(i&0xffff) + s0/2) / s0                          // IDIVQ
			accInt += (256*((i+1)&0xffff) + s1/2) / s1                      // IDIVQ
			accFloat += int(float64(256*((i+2)&0xffff)+s2/2) / float64(s2)) // DIVSD
			accFloat += int(float64(256*((i+3)&0xffff)+s3/2) / float64(s3)) // DIVSD
			accFloat += int(float64(256*((i+4)&0xffff)+s4/2) / float64(s4)) // DIVSD
			i += 5
		}
		sink = accInt + accFloat
	})
	b.Run("div=idivq-u5", func(b *testing.B) {
		a0, a1, i := 0, 0, 0
		for b.Loop() {
			s0, s1, s2, s3, s4 := divisors[i&7], divisors[(i+1)&7], divisors[(i+2)&7], divisors[(i+3)&7], divisors[(i+4)&7]
			a0 += (256*(i&0xffff) + s0/2) / s0
			a0 += (256*((i+1)&0xffff) + s1/2) / s1
			a1 += (256*((i+2)&0xffff) + s2/2) / s2
			a1 += (256*((i+3)&0xffff) + s3/2) / s3
			a1 += (256*((i+4)&0xffff) + s4/2) / s4
			i += 5
		}
		sink = a0 + a1
	})
	b.Run("div=float-u5", func(b *testing.B) {
		a0, a1, i := 0, 0, 0
		for b.Loop() {
			s0, s1, s2, s3, s4 := divisors[i&7], divisors[(i+1)&7], divisors[(i+2)&7], divisors[(i+3)&7], divisors[(i+4)&7]
			a0 += int(float64(256*(i&0xffff)+s0/2) / float64(s0))
			a0 += int(float64(256*((i+1)&0xffff)+s1/2) / float64(s1))
			a1 += int(float64(256*((i+2)&0xffff)+s2/2) / float64(s2))
			a1 += int(float64(256*((i+3)&0xffff)+s3/2) / float64(s3))
			a1 += int(float64(256*((i+4)&0xffff)+s4/2) / float64(s4))
			i += 5
		}
		sink = a0 + a1
	})
	b.Run("div=reciprocal-u5", func(b *testing.B) {
		a0, a1, i := 0, 0, 0
		for b.Loop() {
			s0, s1, s2, s3, s4 := divisors[i&7], divisors[(i+1)&7], divisors[(i+2)&7], divisors[(i+3)&7], divisors[(i+4)&7]
			a0 += int((uint64(256*(i&0xffff)+s0/2) * recipTable[i&7]) >> reciprocalShift)
			a0 += int((uint64(256*((i+1)&0xffff)+s1/2) * recipTable[(i+1)&7]) >> reciprocalShift)
			a1 += int((uint64(256*((i+2)&0xffff)+s2/2) * recipTable[(i+2)&7]) >> reciprocalShift)
			a1 += int((uint64(256*((i+3)&0xffff)+s3/2) * recipTable[(i+3)&7]) >> reciprocalShift)
			a1 += int((uint64(256*((i+4)&0xffff)+s4/2) * recipTable[(i+4)&7]) >> reciprocalShift)
			i += 5
		}
		sink = a0 + a1
	})
	b.Run("div=fastdiv-u5", func(b *testing.B) {
		a0, a1, i := 0, 0, 0
		for b.Loop() {
			s0, s1, s2, s3, s4 := divisors[i&7], divisors[(i+1)&7], divisors[(i+2)&7], divisors[(i+3)&7], divisors[(i+4)&7]
			a0 += int(fastInv[i&7].Div(uint32(256*(i&0xffff) + s0/2)))
			a0 += int(fastInv[(i+1)&7].Div(uint32(256*((i+1)&0xffff) + s1/2)))
			a1 += int(fastInv[(i+2)&7].Div(uint32(256*((i+2)&0xffff) + s2/2)))
			a1 += int(fastInv[(i+3)&7].Div(uint32(256*((i+3)&0xffff) + s3/2)))
			a1 += int(fastInv[(i+4)&7].Div(uint32(256*((i+4)&0xffff) + s4/2)))
			i += 5
		}
		sink = a0 + a1
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
