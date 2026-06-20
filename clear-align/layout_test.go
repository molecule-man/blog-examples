package clearalign

import (
	"testing"
	"unsafe"
)

func TestBlockPageOffsets(t *testing.T) {
	cases := []struct {
		name string
		addr uintptr
		want uintptr
	}{
		{"blk00", uintptr(unsafe.Pointer(&new(blk00).data)), 0},
		{"blk04", uintptr(unsafe.Pointer(&new(blk04).data)), 4},
		{"blk08", uintptr(unsafe.Pointer(&new(blk08).data)), 8},
		{"blk12", uintptr(unsafe.Pointer(&new(blk12).data)), 12},
		{"blk16", uintptr(unsafe.Pointer(&new(blk16).data)), 16},
		{"blk20", uintptr(unsafe.Pointer(&new(blk20).data)), 20},
		{"blk24", uintptr(unsafe.Pointer(&new(blk24).data)), 24},
		{"blk28", uintptr(unsafe.Pointer(&new(blk28).data)), 28},
	}
	for _, c := range cases {
		gotPage := c.addr % 4096
		if gotPage != c.want {
			t.Errorf("%s: data page offset = %d, want %d (allocation not page-aligned?)",
				c.name, gotPage, c.want)
		}
		if want8 := c.want % 8; gotPage%8 != want8 {
			t.Errorf("%s: data offset mod 8 = %d, want %d", c.name, gotPage%8, want8)
		}
	}
}
