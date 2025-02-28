package vm

import (
	"fmt"
	"math"

	"github.com/uganh16/golua/internal/debug"
	"github.com/uganh16/golua/internal/number"
	"github.com/uganh16/golua/internal/value"
	"github.com/uganh16/golua/pkg/lua"
)

type VM interface {
	ConstantRead(idx int) value.LuaValue

	RegisterRead(idx int) value.LuaValue
	RegisterWrite(idx int, val value.LuaValue)

	Fetch() Instruction
	AddPC(int)
}

func (i Instruction) Execute(vm VM) {
	switch i.Opcode() {
	case OP_MOVE: /* R(A) := R(B) */
		a, b, _ := i.ABC()
		vm.RegisterWrite(a, vm.RegisterRead(b))
	case OP_LOADK: /* R(A) := Kst(Bx) */
		a, bx := i.ABx()
		vm.RegisterWrite(a, vm.ConstantRead(bx))
	case OP_LOADKX: /* R(A) := Kst(extra arg) */
		a, _ := i.ABx()
		ax := vm.Fetch().Ax()
		vm.RegisterWrite(a, vm.ConstantRead(ax))
	case OP_LOADBOOL: /* R(A) := (Bool)B; if (C) pc++ */
		a, b, c := i.ABC()
		vm.RegisterWrite(a, value.NewBoolean(b != 0))
		if c != 0 {
			vm.AddPC(1)
		}
	case OP_LOADNIL: /* R(A), R(A+1), ..., R(A+B) := nil */
		a, b, _ := i.ABC()
		for b >= 0 {
			vm.RegisterWrite(a, value.Nil)
			b--
		}
	case OP_GETUPVAL:
	case OP_GETTABUP:
	case OP_GETTABLE:
	case OP_SETTABUP:
	case OP_SETUPVAL:
	case OP_SETTABLE:
	case OP_NEWTABLE:
	case OP_SELF:
	case OP_ADD: /* R(A) := RK(B) + RK(C) */
		binaryArith(i, vm, lua.OPADD)
	case OP_SUB: /* R(A) := RK(B) - RK(C) */
		binaryArith(i, vm, lua.OPSUB)
	case OP_MUL: /* R(A) := RK(B) * RK(C) */
		binaryArith(i, vm, lua.OPMUL)
	case OP_MOD: /* R(A) := RK(B) % RK(C) */
		binaryArith(i, vm, lua.OPMOD)
	case OP_POW: /* R(A) := RK(B) ^ RK(C) */
		binaryArith(i, vm, lua.OPPOW)
	case OP_DIV: /* R(A) := RK(B) / RK(C) */
		binaryArith(i, vm, lua.OPDIV)
	case OP_IDIV: /* R(A) := RK(B) // RK(C) */
		binaryArith(i, vm, lua.OPIDIV)
	case OP_BAND: /* R(A) := RK(B) & RK(C) */
		binaryArith(i, vm, lua.OPBAND)
	case OP_BOR: /* R(A) := RK(B) | RK(C) */
		binaryArith(i, vm, lua.OPBOR)
	case OP_BXOR: /* R(A) := RK(B) ~ RK(C) */
		binaryArith(i, vm, lua.OPBXOR)
	case OP_SHL: /* R(A) := RK(B) << RK(C) */
		binaryArith(i, vm, lua.OPSHL)
	case OP_SHR: /* R(A) := RK(B) >> RK(C) */
		binaryArith(i, vm, lua.OPSHR)
	case OP_UNM: /* R(A) := -R(B) */
		unaryArith(i, vm, lua.OPUNM)
	case OP_BNOT: /* R(A) := ~R(B) */
		unaryArith(i, vm, lua.OPBNOT)
	case OP_NOT: /* R(A) := not R(B) */
		a, b, _ := i.ABC()
		vm.RegisterWrite(a, value.NewBoolean(!value.ToBoolean(vm.RegisterRead(b))))
	case OP_LEN: /* R(A) := length of R(B) */
		a, b, _ := i.ABC()
		a += 1
		b += 1
		vm.RegisterWrite(a, Len(vm.RegisterRead(b)))
	case OP_CONCAT: /* R(A) := R(B).. ... ..R(C) */
		a, b, c := i.ABC()
		var vals []value.LuaValue
		for b <= c {
			vals = append(vals, vm.RegisterRead(b))
			b++
		}
		vm.RegisterWrite(a, Concat(vals))
	case OP_JMP: /* pc+=sBx; if (A) close all upvalues >= R(A - 1) */
		a, sbx := i.AsBx()
		vm.AddPC(sbx)
		if a != 0 {
			panic("todo!")
		}
	case OP_EQ: /* if ((RK(B) == RK(C)) ~= A) then pc++ */
		a, b, c := i.ABC()
		if value.Equal(getRK(vm, b), getRK(vm, c)) != (a != 0) {
			vm.AddPC(1)
		}
	case OP_LT: /* if ((RK(B) <  RK(C)) ~= A) then pc++ */
		a, b, c := i.ABC()
		if LessThan(getRK(vm, b), getRK(vm, c)) != (a != 0) {
			vm.AddPC(1)
		}
	case OP_LE: /* if ((RK(B) <= RK(C)) ~= A) then pc++ */
		a, b, c := i.ABC()
		if LessEqual(getRK(vm, b), getRK(vm, c)) != (a != 0) {
			vm.AddPC(1)
		}
	case OP_TEST: /* if not (R(A) <=> C) then pc++ */
		a, _, c := i.ABC()
		if value.ToBoolean(vm.RegisterRead(a)) != (c != 0) {
			vm.AddPC(1)
		}
	case OP_TESTSET: /* if (R(B) <=> C) then R(A) := R(B) else pc++ */
		a, b, c := i.ABC()
		if b := vm.RegisterRead(b); value.ToBoolean(b) == (c != 0) {
			vm.RegisterWrite(a, b)
		} else {
			vm.AddPC(1)
		}
	case OP_CALL:
	case OP_TAILCALL:
	case OP_RETURN:
	case OP_FORLOOP: /* R(A)+=R(A+2); if R(A) <?= R(A+1) then { pc+=sBx; R(A+3)=R(A) } */
		a, sbx := i.AsBx()
		initial := vm.RegisterRead(a)
		limit := vm.RegisterRead(a + 1)
		step := vm.RegisterRead(a + 2)
		initial = Arith(initial, step, lua.OPADD)
		vm.RegisterWrite(a, initial)
		if step, ok := value.ToNumber(step); ok {
			if step >= 0 && LessEqual(initial, limit) ||
				step < 0 && LessEqual(limit, initial) {
				vm.AddPC(sbx)
				vm.RegisterWrite(a+3, initial)
			}
		} else {
			panic(debug.RuntimeError("'for' step must be a number"))
		}
	case OP_FORPREP: /* R(A)-=R(A+2); pc+=sBx */
		a, sbx := i.AsBx()
		vm.RegisterWrite(a, Arith(vm.RegisterRead(a), vm.RegisterRead(a+2), lua.OPSUB))
		vm.AddPC(sbx)
	case OP_TFORCALL:
	case OP_TFORLOOP:
	case OP_SETLIST:
	case OP_CLOSURE:
	case OP_VARARG:
	case OP_EXTRAARG:
	}
}

func unaryArith(i Instruction, vm VM, op lua.ArithOp) {
	a, b, _ := i.ABC()
	val := vm.RegisterRead(b)
	vm.RegisterWrite(a, Arith(val, val, op))
}

func binaryArith(i Instruction, vm VM, op lua.ArithOp) {
	a, b, c := i.ABC()
	vm.RegisterWrite(a, Arith(getRK(vm, b), getRK(vm, c), op))
}

func getRK(vm VM, idx int) value.LuaValue {
	if idx > 0xff {
		return vm.ConstantRead(idx & 0xff)
	} else {
		return vm.RegisterRead(idx)
	}
}

func Arith(a, b value.LuaValue, op lua.ArithOp) value.LuaValue {
	var iFunc func(int64, int64) int64
	var fFunc func(float64, float64) float64

	switch op {
	case lua.OPADD:
		iFunc = func(a, b int64) int64 { return a + b }
		fFunc = func(a, b float64) float64 { return a + b }
	case lua.OPSUB:
		iFunc = func(a, b int64) int64 { return a - b }
		fFunc = func(a, b float64) float64 { return a - b }
	case lua.OPMUL:
		iFunc = func(a, b int64) int64 { return a * b }
		fFunc = func(a, b float64) float64 { return a * b }
	case lua.OPMOD:
		iFunc = number.IMod
		fFunc = number.FMod
	case lua.OPPOW:
		fFunc = math.Pow
	case lua.OPDIV:
		fFunc = func(a, b float64) float64 { return a / b }
	case lua.OPIDIV:
		iFunc = number.IFloorDiv
		fFunc = number.FFloorDiv
	case lua.OPBAND:
		iFunc = func(a, b int64) int64 { return a & b }
	case lua.OPBOR:
		iFunc = func(a, b int64) int64 { return a | b }
	case lua.OPBXOR:
		iFunc = func(a, b int64) int64 { return a ^ b }
	case lua.OPSHL:
		iFunc = number.ShiftLeft
	case lua.OPSHR:
		iFunc = number.ShiftRight
	case lua.OPUNM:
		iFunc = func(a, _ int64) int64 { return -a }
		fFunc = func(a, _ float64) float64 { return -a }
	case lua.OPBNOT:
		iFunc = func(a, _ int64) int64 { return ^a }
	default:
		panic(fmt.Sprintf("invalid arith op: %d", op))
	}

	if fFunc == nil { // bitwise operation
		if a, ok := value.ToInteger(a); ok {
			if b, ok := value.ToInteger(b); ok {
				return value.NewInteger(iFunc(a, b))
			}
		}
	} else {
		if iFunc != nil {
			if a.Type() == value.LUA_TNUMINT && b.Type() == value.LUA_TNUMINT {
				return value.NewInteger(iFunc(value.AsInteger(a), value.AsInteger(b)))
			}
		}

		if a, ok := value.ToNumber(a); ok {
			if b, ok := value.ToNumber(b); ok {
				return value.NewNumber(fFunc(a, b))
			}
		}
	}

	// @todo tm

	switch op {
	case lua.OPBAND, lua.OPBOR, lua.OPBXOR, lua.OPSHL, lua.OPSHR:
		_, ok1 := value.ToNumber(a)
		_, ok2 := value.ToNumber(b)
		if ok1 && ok2 {
			panic(debug.ToIntError(a, b))
		} else {
			panic(debug.OpIntError(a, b, "perform bitwise operation on"))
		}
	default:
		panic(debug.OpIntError(a, b, "perform arithmetic on"))
	}
}

func LessThan(a, b value.LuaValue) bool {
	if res, ok := value.LessThan(a, b); ok {
		return res
	}
	panic(debug.OrderError(a, b))
}

func LessEqual(a, b value.LuaValue) bool {
	if res, ok := value.LessEqual(a, b); ok {
		return res
	}
	panic(debug.OrderError(a, b))
}

func Len(val value.LuaValue) value.LuaValue {
	if res, ok := value.Len(val); ok {
		return value.NewInteger(int64(res))
	}
	panic(debug.TypeError(val, "get length of"))
}

func Concat(vals []value.LuaValue) value.LuaValue {
	b := vals[len(vals)-1]
	for i := len(vals) - 2; i >= 0; i-- {
		a := vals[i]
		if s1, ok := value.ToString(a); ok {
			if s2, ok := value.ToString(b); ok {
				b = value.NewString(s1 + s2)
				continue
			}
		}
		// @todo mt
		panic(debug.ConcatError(a, b))
	}
	return b
}
