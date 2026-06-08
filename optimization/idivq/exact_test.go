package divbench

import "testing"

func TestReciprocalAndFastdivExact(t *testing.T) {
	for _, d := range divisors {
		recip := recipTable[idx(d)]
		inv := fastInv[idx(d)]
		for s := 0; s <= 0xffff; s++ {
			num := 256*s + d/2
			want := num / d
			if got := int((uint64(num) * recip) >> reciprocalShift); got != want {
				t.Fatalf("reciprocal d=%d s=%d: got %d want %d", d, s, got, want)
			}
			if got := int(inv.Div(uint32(num))); got != want {
				t.Fatalf("fastdiv d=%d s=%d: got %d want %d", d, s, got, want)
			}
		}
	}
}

func idx(d int) int {
	for i, v := range divisors {
		if v == d {
			return i
		}
	}
	panic("divisor not found")
}
