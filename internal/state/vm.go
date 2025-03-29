package state

import (
	"github.com/uganh16/golua/internal/bytecode"
	"github.com/uganh16/golua/internal/number"
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
			ax := p.Code[ci.pc].Ax() // @todo assert OP_EXTRAARG
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
		case bytecode.OP_GETUPVAL: /* R(A) := UpValue[B] */
			a, b, _ := i.ABC()
			L.setR(a, cl.upvals[b].get(L))
		case bytecode.OP_GETTABUP: /* R(A) := UpValue[B][RK(C)] */
			a, b, c := i.ABC()
			L.setR(a, L.getTable(cl.upvals[b].get(L), L.getRK(c)))
		case bytecode.OP_GETTABLE: /* R(A) := R(B)[RK(C)] */
			a, b, c := i.ABC()
			L.setR(a, L.getTable(L.getR(b), L.getRK(c)))
		case bytecode.OP_SETTABUP: /* UpValue[A][RK(B)] := RK(C) */
			a, b, c := i.ABC()
			L.setTable(cl.upvals[a].get(L), L.getRK(b), L.getRK(c))
		case bytecode.OP_SETUPVAL: /* UpValue[B] := R(A) */
			a, b, _ := i.ABC()
			cl.upvals[b].set(L, L.getR(a))
		case bytecode.OP_SETTABLE: /* R(A)[RK(B)] := RK(C) */
			a, b, c := i.ABC()
			L.setTable(L.getR(a), L.getRK(b), L.getRK(c))
		case bytecode.OP_NEWTABLE: /* R(A) := {} (size = B,C) */
			a, b, c := i.ABC()
			L.setR(a, newLuaTable(number.Fb2int(b), number.Fb2int(c)))
		case bytecode.OP_SELF: /* R(A+1) := R(B); R(A) := R(B)[RK(C)] */
			a, b, c := i.ABC()
			key := L.getRK(c).(string) /* key must be a string */
			t := L.getR(b)
			L.setR(a+1, t)
			L.setR(a, L.getTable(t, key))
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
			L.setR(a, _len(L.getR(b)))
		case bytecode.OP_CONCAT: /* R(A) := R(B).. ... ..R(C) */
			a, b, c := i.ABC()
			L.setR(a, _concat(L.stack[base+b:base+c+1]))
		case bytecode.OP_JMP: /* pc+=sBx; if (A) close all upvalues >= R(A - 1) */
			a, sbx := i.AsBx()
			ci.pc += sbx
			if a != 0 {
				L.closeUpvalues(base + a - 1)
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
					L.stack = L.stack[:ci.top] /* adjust results */
				}
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
			if len(cl.proto.Protos) > 0 {
				L.closeUpvalues(base)
			}
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
		case bytecode.OP_SETLIST: /* R(A)[(C-1)*FPF+i] := R(A+i), 1 <= i <= B */
			a, b, c := i.ABC()
			if b == 0 {
				b = len(L.stack) - (base + a) - 1
			}
			if c == 0 {
				c = p.Code[ci.pc].Ax() // @todo assert OP_EXTRAARG
				ci.pc++
			}
			t := L.getR(a).(*luaTable)
			idx := lua.Integer((c - 1) * bytecode.LFIELDS_PER_FLUSH)
			for j := 1; j <= b; j++ {
				idx++
				t.set(idx, L.getR(a+j))
			}
			L.stack = L.stack[:ci.top] /* correct top (in case of previous open call) */
		case bytecode.OP_CLOSURE: /* R(A) := closure(KPROTO[Bx]) */
			a, bx := i.ABx()
			p := cl.proto.Protos[bx]
			ncl := newLuaClosure(p)
			L.setR(a, ncl)
			for i, uv := range p.Upvalues { /* fill in its upvalues */
				if uv.InStack { /* upvalue refers to local variable? */
					ncl.upvals[i] = L.findUpvalue(base + int(uv.Idx))
				} else { /* get upvalue from enclosing function */
					ncl.upvals[i] = cl.upvals[uv.Idx]
				}
			}
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
	return L.stack[L.ci.base+idx]
}

func (L *luaState) setR(idx int, val luaValue) {
	L.stack[L.ci.base+idx] = val
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
