package clearalign

import (
	"testing"
	"unsafe"
)

func stosbClear(p unsafe.Pointer, n uintptr)

type storeVariant struct {
	name string
	fn   func(blk *blk00)
}

var storeVariants = []storeVariant{
	{"stosq", func(blk *blk00) { clear00(blk) }},
	{"stosb", func(blk *blk00) { stosbClear(unsafe.Pointer(&blk.data), words*4) }},
}

func BenchmarkStore(b *testing.B) {
	blk := new(blk00)
	for _, v := range storeVariants {
		b.Run("impl="+v.name, func(b *testing.B) {
			b.SetBytes(words * 4)
			for b.Loop() {
				v.fn(blk)
			}
		})
	}
}
