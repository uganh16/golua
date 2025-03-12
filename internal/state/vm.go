package state

import (
	"github.com/uganh16/golua/internal/bytecode"
	"github.com/uganh16/golua/pkg/lua"
)

func (L *luaState) execute() {
	ci := L.ci
	ci.callStatus |= CIST_FRESH
newFrame:
	cl := L.stack[ci.cl].(*lClosure)
	p := cl.proto
	base := ci.base
	for {
		i := p.Code[ci.pc]
		ci.pc++
		// @todo L->hookmask
		switch opcode := i.Opcode(); opcode {
		case bytecode.OP_MOVE: /* R(A) := R(B) */
			a, b, _ := i.ABC()
			L.setR(a, L.getR(b))
		case bytecode.OP_LOADK: /* R(A) := Kst(Bx) */
			a, bx := i.ABx()
			L.setR(a, L.getK(bx))
		case bytecode.OP_LOADKX: /* R(A) := Kst(extra arg) */
			a, _ := i.ABx()
			ax := p.Code[ci.pc].Ax()
			ci.pc++
			L.setR(a, L.getK(ax))
		case bytecode.OP_LOADBOOL: /* R(A) := (Bool)B; if (C) pc++ */
			a, b, c := i.ABC()
			L.setR(a, b != 0)
			if c != 0 {
				ci.pc++
			}
		case bytecode.OP_LOADNIL: /* R(A), R(A+1), ..., R(A+B) := nil */
			a, b, _ := i.ABC()
			for b >= 0 {
				L.setR(a, nil)
				b--
			}
		case bytecode.OP_GETUPVAL:
		case bytecode.OP_GETTABUP:
		case bytecode.OP_GETTABLE:
		case bytecode.OP_SETTABUP:
		case bytecode.OP_SETUPVAL:
		case bytecode.OP_SETTABLE:
		case bytecode.OP_NEWTABLE:
		case bytecode.OP_SELF:
		case
			bytecode.OP_ADD,  /* R(A) := RK(B) + RK(C) */
			bytecode.OP_SUB,  /* R(A) := RK(B) - RK(C) */
			bytecode.OP_MUL,  /* R(A) := RK(B) * RK(C) */
			bytecode.OP_MOD,  /* R(A) := RK(B) % RK(C) */
			bytecode.OP_POW,  /* R(A) := RK(B) ^ RK(C) */
			bytecode.OP_DIV,  /* R(A) := RK(B) / RK(C) */
			bytecode.OP_IDIV, /* R(A) := RK(B) // RK(C) */
			bytecode.OP_BAND, /* R(A) := RK(B) & RK(C) */
			bytecode.OP_BOR,  /* R(A) := RK(B) | RK(C) */
			bytecode.OP_BXOR, /* R(A) := RK(B) ~ RK(C) */
			bytecode.OP_SHL,  /* R(A) := RK(B) << RK(C) */
			bytecode.OP_SHR:  /* R(A) := RK(B) >> RK(C) */
			a, b, c := i.ABC()
			L.setR(a, _arith(L.getRK(b), L.getRK(c), lua.ArithOp(opcode-bytecode.OP_ADD)))
		case
			bytecode.OP_UNM,  /* R(A) := -R(B) */
			bytecode.OP_BNOT: /* R(A) := ~R(B) */
			a, b, _ := i.ABC()
			val := L.getR(b)
			L.setR(a, _arith(val, val, lua.ArithOp(opcode-bytecode.OP_ADD)))
		case bytecode.OP_NOT: /* R(A) := not R(B) */
			a, b, _ := i.ABC()
			L.setR(a, !toBoolean(L.getR(b)))
		case bytecode.OP_LEN: /* R(A) := length of R(B) */
			a, b, _ := i.ABC()
			a += 1
			b += 1
			L.setR(a, _len(L.getR(b)))
		case bytecode.OP_CONCAT: /* R(A) := R(B).. ... ..R(C) */
			a, b, c := i.ABC()
			L.setR(a, _concat(L.stack[b:c+1]))
		case bytecode.OP_JMP: /* pc+=sBx; if (A) close all upvalues >= R(A - 1) */
			a, sbx := i.AsBx()
			ci.pc += sbx
			if a != 0 {
				panic("todo!")
			}
		case bytecode.OP_EQ: /* if ((RK(B) == RK(C)) ~= A) then pc++ */
			a, b, c := i.ABC()
			if _eq(L.getRK(b), L.getRK(c)) != (a != 0) {
				ci.pc++
			}
		case bytecode.OP_LT: /* if ((RK(B) <  RK(C)) ~= A) then pc++ */
			a, b, c := i.ABC()
			if _lt(L.getRK(b), L.getRK(c)) != (a != 0) {
				ci.pc++
			}
		case bytecode.OP_LE: /* if ((RK(B) <= RK(C)) ~= A) then pc++ */
			a, b, c := i.ABC()
			if _le(L.getRK(b), L.getRK(c)) != (a != 0) {
				ci.pc++
			}
		case bytecode.OP_TEST: /* if not (R(A) <=> C) then pc++ */
			a, _, c := i.ABC()
			if toBoolean(L.getR(a)) != (c != 0) {
				ci.pc++
			}
		case bytecode.OP_TESTSET: /* if (R(B) <=> C) then R(A) := R(B) else pc++ */
			a, b, c := i.ABC()
			if b := L.getR(b); toBoolean(b) == (c != 0) {
				L.setR(a, b)
			} else {
				ci.pc++
			}
		case bytecode.OP_CALL: /* R(A), ... ,R(A+C-2) := R(A)(R(A+1), ... ,R(A+B-1)) */
			a, b, c := i.ABC()
			nResults := c - 1
			if b >= 1 {
				L.stack = L.stack[:base+a+b]
			} /* (!) else previous instruction set top */
			if L.preCall(L.getR(a), len(L.stack)-(base+a)-1, nResults) { /* Go function? */
				if nResults >= 0 {
					// @todo adjust results
				}
				// @todo update 'base'
			} else { /* Lua function */
				ci = L.ci
				goto newFrame
			}
		case bytecode.OP_TAILCALL: /* return R(A)(R(A+1), ... ,R(A+B-1)) */
			a, b, _ := i.ABC()
			if b >= 1 {
				L.stack = L.stack[:base+a+b]
			} /* (!) else previous instruction set top */
			if !L.preCall(L.getR(a), len(L.stack)-(base+a)-1, lua.MULTRET) { /* Go function? */
				ci = L.ci
				goto newFrame
				/* @todo tail call: put called frame (n) in place of caller one (o) */
			}
		case bytecode.OP_RETURN: /* return R(A), ... ,R(A+B-2) */
			a, b, _ := i.ABC()
			// @todo luaF_close(L, base)
			firstResult := base + a
			var nResults int
			if b != 0 {
				nResults = b - 1
			} else {
				nResults = len(L.stack) - firstResult
			}
			done := L.postCall(firstResult, nResults)
			if ci.callStatus&CIST_FRESH != 0 {
				return
			} else {
				ci = L.ci
				if done {
					L.stack = L.stack[:ci.top]
				}
				goto newFrame
			}
		case bytecode.OP_FORLOOP: /* R(A)+=R(A+2); if R(A) <?= R(A+1) then { pc+=sBx; R(A+3)=R(A) } */
			a, sbx := i.AsBx()
			initial := L.getR(a)
			limit := L.getR(a + 1)
			step := L.getR(a + 2)
			initial = _arith(initial, step, lua.OPADD)
			L.setR(a, initial)
			if step, ok := toNumber(step); ok {
				if step >= 0 && _le(initial, limit) ||
					step < 0 && _le(limit, initial) {
					ci.pc += sbx
					L.setR(a+3, initial)
				}
			} else {
				panic(runtimeError("'for' step must be a number"))
			}
		case bytecode.OP_FORPREP: /* R(A)-=R(A+2); pc+=sBx */
			a, sbx := i.AsBx()
			L.setR(a, _arith(L.getR(a), L.getR(a+2), lua.OPSUB))
			ci.pc += sbx
		case bytecode.OP_TFORCALL:
		case bytecode.OP_TFORLOOP:
		case bytecode.OP_SETLIST:
		case bytecode.OP_CLOSURE: /* R(A) := closure(KPROTO[Bx]) */
			a, bx := i.ABx()
			L.setR(a, newLuaClosure(cl.proto.Protos[bx]))
		case bytecode.OP_VARARG: /* R(A), R(A+1), ..., R(A+B-2) = vararg */
			a, b, _ := i.ABC()
			nResults := b - 1 /* required results */
			n := (base - ci.cl - 1) - int(p.NumParams)
			if n < 0 { /* less arguments than parameters? */
				n = 0 /* no vararg arguments */
			}
			if nResults < 0 {
				if L.stackLast-len(L.stack) < n {
					L.stackGrow(n)
				}
				nResults = n
				L.stack = L.stack[:base+a+n] /* (!) */
			}
			for j := 0; j < nResults && j < n; j++ {
				L.setR(a+j, L.stack[base-n+j])
			}
			for j := n; j < nResults; j++ { /* complete required results with nil */
				L.setR(a+j, nil)
			}
		case bytecode.OP_EXTRAARG:
		}
	}
}

func (L *luaState) getR(idx int) luaValue {
	val, _ := L.stackGet(idx + 1)
	return val
}

func (L *luaState) setR(idx int, val luaValue) {
	L.stackSet(idx+1, val)
}

func (L *luaState) getK(idx int) luaValue {
	return L.stack[L.ci.cl].(*lClosure).proto.Constants[idx]
}

func (L *luaState) getRK(idx int) luaValue {
	if idx > 0xff {
		return L.getK(idx & 0xff)
	} else {
		return L.getR(idx)
	}
}
