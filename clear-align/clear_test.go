package clearalign

import "testing"

const words = 1 << 19

type (
	blk00 struct {
		_    [0]byte
		data [words]uint32
	}
	blk04 struct {
		_    [4]byte
		data [words]uint32
	}
	blk08 struct {
		_    [8]byte
		data [words]uint32
	}
	blk12 struct {
		_    [12]byte
		data [words]uint32
	}
	blk16 struct {
		_    [16]byte
		data [words]uint32
	}
	blk20 struct {
		_    [20]byte
		data [words]uint32
	}
	blk24 struct {
		_    [24]byte
		data [words]uint32
	}
	blk28 struct {
		_    [28]byte
		data [words]uint32
	}
)

//go:noinline
func clear00(b *blk00) { b.data = [words]uint32{} }

//go:noinline
func clear04(b *blk04) { b.data = [words]uint32{} }

//go:noinline
func clear08(b *blk08) { b.data = [words]uint32{} }

//go:noinline
func clear12(b *blk12) { b.data = [words]uint32{} }

//go:noinline
func clear16(b *blk16) { b.data = [words]uint32{} }

//go:noinline
func clear20(b *blk20) { b.data = [words]uint32{} }

//go:noinline
func clear24(b *blk24) { b.data = [words]uint32{} }

//go:noinline
func clear28(b *blk28) { b.data = [words]uint32{} }

func BenchmarkClear(b *testing.B) {
	b00, b04 := new(blk00), new(blk04)
	b08, b12 := new(blk08), new(blk12)
	b16, b20 := new(blk16), new(blk20)
	b24, b28 := new(blk24), new(blk28)

	cases := []struct {
		name string
		fn   func()
	}{
		{"off=00/mod8=0", func() { clear00(b00) }},
		{"off=04/mod8=4", func() { clear04(b04) }},
		{"off=08/mod8=0", func() { clear08(b08) }},
		{"off=12/mod8=4", func() { clear12(b12) }},
		{"off=16/mod8=0", func() { clear16(b16) }},
		{"off=20/mod8=4", func() { clear20(b20) }},
		{"off=24/mod8=0", func() { clear24(b24) }},
		{"off=28/mod8=4", func() { clear28(b28) }},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.SetBytes(words * 4)
			for b.Loop() {
				c.fn()
			}
		})
	}
}
