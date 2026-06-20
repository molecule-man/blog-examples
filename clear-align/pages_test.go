// Reserve the pool first:  echo 64 | sudo tee /proc/sys/vm/nr_hugepages

//go:build goexperiment.simd && linux && amd64

package clearalign

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"syscall"
	"testing"
	"unsafe"

	"simd/archsimd"
)

const (
	madvNohugepage = 0xf     // MADV_NOHUGEPAGE
	mapHugetlb     = 0x40000 // MAP_HUGETLB
	twoMiB         = 1 << 21
)

func mapAligned(huge bool) (region, munmap []byte, err error) {
	if huge {
		m, err := syscall.Mmap(-1, 0, twoMiB, syscall.PROT_READ|syscall.PROT_WRITE,
			syscall.MAP_ANON|syscall.MAP_PRIVATE|mapHugetlb)
		if err != nil {
			return nil, nil, err
		}
		for i := 0; i < len(m); i += 4096 {
			m[i] = 0
		}
		return m, m, nil
	}
	m, err := syscall.Mmap(-1, 0, 2*twoMiB, syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		return nil, nil, err
	}
	base := uintptr(unsafe.Pointer(&m[0]))
	off := (twoMiB - base%twoMiB) % twoMiB
	region = m[off : off+twoMiB : off+twoMiB]
	if err := syscall.Madvise(region, madvNohugepage); err != nil {
		syscall.Munmap(m)
		return nil, nil, err
	}
	for i := 0; i < len(region); i += 4096 {
		region[i] = 0
	}
	return region, m, nil
}

var vmaRe = regexp.MustCompile(`^([0-9a-f]+)-([0-9a-f]+) `)

func smapsFieldKiB(tb testing.TB, addr uintptr, field string) int {
	data, err := os.ReadFile("/proc/self/smaps")
	if err != nil {
		tb.Fatal(err)
	}
	prefix := field + ":"
	inVMA := false
	for _, ln := range splitLines(string(data)) {
		if m := vmaRe.FindStringSubmatch(ln); m != nil {
			start, _ := strconv.ParseUint(m[1], 16, 64)
			end, _ := strconv.ParseUint(m[2], 16, 64)
			inVMA = uint64(addr) >= start && uint64(addr) < end
		} else if inVMA && len(ln) >= len(prefix) && ln[:len(prefix)] == prefix {
			var kib int
			fmt.Sscanf(ln, prefix+" %d kB", &kib)
			return kib
		}
	}
	return -1
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}

func BenchmarkPages(b *testing.B) {
	if !archsimd.X86.AVX2() {
		b.Skip("needs AVX2")
	}
	for _, huge := range []bool{false, true} {
		back := "4k"
		if huge {
			back = "huge"
		}
		region, munmap, err := mapAligned(huge)
		if err != nil {
			b.Run("backing="+back+"/UNAVAILABLE", func(b *testing.B) {
				b.Skipf("could not map %s buffer: %v (reserve a pool: echo 64 | sudo tee /proc/sys/vm/nr_hugepages)", back, err)
			})
			continue
		}
		p := unsafe.Pointer(&region[0])
		n := uintptr(len(region))
		kps := smapsFieldKiB(b, uintptr(p), "KernelPageSize")
		clears := []struct {
			name string
			fn   func(unsafe.Pointer, uintptr)
		}{
			{"cached", clearAVX2},
			{"nt", clearNT256},
		}
		for _, c := range clears {
			name := fmt.Sprintf("backing=%s/KernelPageSize=%dkiB/impl=%s", back, kps, c.name)
			b.Run(name, func(b *testing.B) {
				b.SetBytes(int64(n))
				for b.Loop() {
					c.fn(p, n)
				}
			})
		}
		syscall.Munmap(munmap)
	}
}
