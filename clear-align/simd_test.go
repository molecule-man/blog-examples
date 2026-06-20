//go:build goexperiment.simd

package clearalign

import (
	"simd/archsimd"
	"unsafe"
)

func bytesOf(blk *blk00) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&blk.data)), words*4)
}

//go:noinline
func clearSIMD128(blk *blk00) {
	buf := bytesOf(blk)
	z := archsimd.BroadcastUint8x16(0)
	for i := 0; i+16 <= len(buf); i += 16 {
		z.StoreSlice(buf[i : i+16])
	}
}

//go:noinline
func clearSIMD256(blk *blk00) {
	buf := bytesOf(blk)
	z := archsimd.BroadcastUint8x32(0)
	for i := 0; i+32 <= len(buf); i += 32 {
		z.StoreSlice(buf[i : i+32])
	}
}

//go:noinline
func clearSIMD512(blk *blk00) {
	buf := bytesOf(blk)
	z := archsimd.BroadcastUint8x64(0)
	for i := 0; i+64 <= len(buf); i += 64 {
		z.StoreSlice(buf[i : i+64])
	}
}

func clearNT128(p unsafe.Pointer, n uintptr)
func clearNT256(p unsafe.Pointer, n uintptr)
func clearNT512(p unsafe.Pointer, n uintptr)

func clearAVX2(p unsafe.Pointer, n uintptr)
func clearAVX2PFT0(p unsafe.Pointer, n uintptr)
func clearAVX2PFW(p unsafe.Pointer, n uintptr)

func nt(fn func(unsafe.Pointer, uintptr)) func(*blk00) {
	return func(blk *blk00) { fn(unsafe.Pointer(&blk.data), words*4) }
}

func init() {
	if archsimd.X86.AVX() {
		storeVariants = append(storeVariants, storeVariant{"simd128_sse", clearSIMD128})
	}
	if archsimd.X86.AVX2() {
		storeVariants = append(storeVariants, storeVariant{"simd256_avx2", clearSIMD256})
	}
	if archsimd.X86.AVX512() {
		storeVariants = append(storeVariants, storeVariant{"simd512_avx512", clearSIMD512})
	}

	if archsimd.X86.AVX2() {
		storeVariants = append(storeVariants,
			storeVariant{"avx2_hand", nt(clearAVX2)},
			storeVariant{"avx2_hand_pft0", nt(clearAVX2PFT0)},
			storeVariant{"avx2_hand_pfw", nt(clearAVX2PFW)},
		)
	}

	storeVariants = append(storeVariants, storeVariant{"nt128_sse", nt(clearNT128)})
	if archsimd.X86.AVX2() {
		storeVariants = append(storeVariants, storeVariant{"nt256_avx2", nt(clearNT256)})
	}
	if archsimd.X86.AVX512() {
		storeVariants = append(storeVariants, storeVariant{"nt512_avx512", nt(clearNT512)})
	}
}
