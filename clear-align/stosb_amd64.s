#include "textflag.h"

TEXT ·stosbClear(SB), NOSPLIT, $0-16
	MOVQ	p+0(FP), DI
	MOVQ	n+8(FP), CX
	XORL	AX, AX
	REP;	STOSB
	RET
