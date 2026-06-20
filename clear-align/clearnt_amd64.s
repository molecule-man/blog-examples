//go:build goexperiment.simd

#include "textflag.h"

TEXT ·clearNT128(SB), NOSPLIT, $0-16
	MOVQ	p+0(FP), DI
	MOVQ	n+8(FP), CX
	SHRQ	$4, CX            // byte count -> 16-byte stores
	PXOR	X0, X0
loop128:
	MOVNTO	X0, (DI)
	ADDQ	$16, DI
	DECQ	CX
	JNZ	loop128
	SFENCE
	RET

TEXT ·clearNT256(SB), NOSPLIT, $0-16
	MOVQ	p+0(FP), DI
	MOVQ	n+8(FP), CX
	SHRQ	$5, CX            // byte count -> 32-byte stores
	VPXOR	Y0, Y0, Y0
loop256:
	VMOVNTDQ	Y0, (DI)
	ADDQ	$32, DI
	DECQ	CX
	JNZ	loop256
	SFENCE
	VZEROUPPER
	RET

TEXT ·clearNT512(SB), NOSPLIT, $0-16
	MOVQ	p+0(FP), DI
	MOVQ	n+8(FP), CX
	SHRQ	$6, CX            // byte count -> 64-byte stores
	VPXORQ	Z0, Z0, Z0
loop512:
	VMOVNTDQ	Z0, (DI)
	ADDQ	$64, DI
	DECQ	CX
	JNZ	loop512
	SFENCE
	VZEROUPPER
	RET
