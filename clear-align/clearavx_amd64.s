//go:build goexperiment.simd

#include "textflag.h"

TEXT ·clearAVX2(SB), NOSPLIT, $0-16
	MOVQ	p+0(FP), DI
	MOVQ	n+8(FP), CX
	SHRQ	$6, CX            // byte count -> 64-byte iterations
	VPXOR	Y0, Y0, Y0
loop:
	VMOVDQU	Y0, 0(DI)
	VMOVDQU	Y0, 32(DI)
	ADDQ	$64, DI
	DECQ	CX
	JNZ	loop
	VZEROUPPER
	RET

TEXT ·clearAVX2PFT0(SB), NOSPLIT, $0-16
	MOVQ	p+0(FP), DI
	MOVQ	n+8(FP), CX
	SHRQ	$6, CX
	VPXOR	Y0, Y0, Y0
loopt0:
	PREFETCHT0	512(DI)
	VMOVDQU	Y0, 0(DI)
	VMOVDQU	Y0, 32(DI)
	ADDQ	$64, DI
	DECQ	CX
	JNZ	loopt0
	VZEROUPPER
	RET

TEXT ·clearAVX2PFW(SB), NOSPLIT, $0-16
	MOVQ	p+0(FP), DI
	MOVQ	n+8(FP), CX
	SHRQ	$6, CX
	VPXOR	Y0, Y0, Y0
loopw:
	BYTE	$0x0f
	BYTE	$0x0d
	BYTE	$0x8f
	BYTE	$0x00
	BYTE	$0x02
	BYTE	$0x00
	BYTE	$0x00             // PREFETCHW 512(DI)
	VMOVDQU	Y0, 0(DI)
	VMOVDQU	Y0, 32(DI)
	ADDQ	$64, DI
	DECQ	CX
	JNZ	loopw
	VZEROUPPER
	RET
