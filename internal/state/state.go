package state

import (
	"fmt"
	"math"

	"github.com/uganh16/golua/internal/debug"
	"github.com/uganh16/golua/internal/number"
	"github.com/uganh16/golua/internal/value"
	"github.com/uganh16/golua/pkg/lua/operators"
	"github.com/uganh16/golua/pkg/lua/types"
)

/* extra stack space to handle TM calls and some other extras */
const EXTRA_STACK = 5

type luaState struct {
	stack []value.LuaValue
}

func New() *luaState {
	return &luaState{
		stack: make([]value.LuaValue, 0, 20), // @todo
	}
}

func (L *luaState) AbsIndex(idx int) int {
	if idx > 0 {
		return idx
	}
	return idx + len(L.stack) + 1 // @todo
}

func (L *luaState) GetTop() int {
	return len(L.stack) // @todo
}

func (L *luaState) SetTop(idx int) {
	top := len(L.stack) // @todo
	if idx >= 0 {
		if idx > cap(L.stack) {
			panic("new top too large")
		}
		for top < idx {
			L.stack = append(L.stack, value.Nil)
			top++
		}
	} else {
		if -(idx + 1) > top {
			panic("invalid new top")
		}
		idx = top + idx + 1
	}
	for top > idx {
		top--
		L.stack[top] = nil
	}
	L.stack = L.stack[:top]
}

func (L *luaState) PushValue(idx int) {
	val, _ := L.stackGet(idx)
	L.stackPush(val)
}

func (L *luaState) Rotate(idx, n int) {
	t := len(L.stack) - 1    // end of stack segment being rotated
	p := L.AbsIndex(idx) - 1 // start of segment
	if p < 0 || p > t {
		panic("index not in the stack")
	}
	var m int // end of prefix
	if n >= 0 {
		m = t - n
	} else {
		m = p - n - 1
	}
	if m < p || m > t {
		panic("invalid 'n'")
	}
	L.stackReverse(p, m)   // reverse the prefix with length 'n'
	L.stackReverse(m+1, t) // reverse the suffix
	L.stackReverse(p, t)   // reverse the entire segment
}

func (L *luaState) Copy(srcIdx, dstIdx int) {
	val, _ := L.stackGet(srcIdx)
	L.stackSet(dstIdx, val)
	// @todo
}

func (L *luaState) CheckStack(n int) bool {
	if n < 0 {
		panic("negative 'n'")
	}
	if cap(L.stack)-len(L.stack) < n { // @todo
		/* try to grow stack */
		needed := len(L.stack) + n + EXTRA_STACK
		newSize := 2 * cap(L.stack)
		if newSize < needed {
			newSize = needed
		}
		if newSize > LUAI_MAXSTACK {
			return false
		}
		newStack := make([]value.LuaValue, len(L.stack), newSize)
		copy(newStack, L.stack)
		L.stack = newStack
	}
	// @todo adjust frame top
	return true
}

func (L *luaState) IsNumber(idx int) bool {
	_, ok := L.ToNumberX(idx)
	return ok
}

func (L *luaState) IsString(idx int) bool {
	t := L.Type(idx)
	return t == types.LUA_TSTRING || t == types.LUA_TNUMBER
}

func (L *luaState) IsInteger(idx int) bool {
	val, _ := L.stackGet(idx)
	return val.Type() == value.LUA_TNUMINT
}

func (L *luaState) Type(idx int) types.LuaType {
	if val, ok := L.stackGet(idx); ok {
		return value.NoVariantType(val.Type())
	}
	return types.LUA_TNONE
}

func (L *luaState) TypeName(t types.LuaType) string {
	if t < types.LUA_TNONE || t >= types.LUA_NUMTAGS {
		panic("invalid tag")
	}
	return value.TypeName(t)
}

func (L *luaState) ToNumberX(idx int) (float64, bool) {
	val, _ := L.stackGet(idx)
	return value.ToNumber(val)
}

func (L *luaState) ToIntegerX(idx int) (int64, bool) {
	val, _ := L.stackGet(idx)
	return value.ToInteger(val)
}

func (L *luaState) ToBoolean(idx int) bool {
	val, _ := L.stackGet(idx)
	return value.ToBoolean(val)
}

func (L *luaState) ToStringX(idx int) (string, bool) {
	val, _ := L.stackGet(idx)
	if str, ok := value.ToString(val); ok {
		L.stackSet(idx, value.NewString(str))
		return str, ok
	}
	return "", false
}

func (L *luaState) Arith(op operators.ArithOp) {
	var a, b, r value.LuaValue
	b = L.stackPop()
	if op != operators.LUA_OPUNM && op != operators.LUA_OPBNOT {
		a = L.stackPop()
	} else {
		a = b
	}

	var iFunc func(int64, int64) int64
	var fFunc func(float64, float64) float64

	switch op {
	case operators.LUA_OPADD:
		iFunc = func(a, b int64) int64 { return a + b }
		fFunc = func(a, b float64) float64 { return a + b }
	case operators.LUA_OPSUB:
		iFunc = func(a, b int64) int64 { return a - b }
		fFunc = func(a, b float64) float64 { return a - b }
	case operators.LUA_OPMUL:
		iFunc = func(a, b int64) int64 { return a * b }
		fFunc = func(a, b float64) float64 { return a * b }
	case operators.LUA_OPMOD:
		iFunc = number.IMod
		fFunc = number.FMod
	case operators.LUA_OPPOW:
		fFunc = math.Pow
	case operators.LUA_OPDIV:
		fFunc = func(a, b float64) float64 { return a / b }
	case operators.LUA_OPIDIV:
		iFunc = number.IFloorDiv
		fFunc = number.FFloorDiv
	case operators.LUA_OPBAND:
		iFunc = func(a, b int64) int64 { return a & b }
	case operators.LUA_OPBOR:
		iFunc = func(a, b int64) int64 { return a | b }
	case operators.LUA_OPBXOR:
		iFunc = func(a, b int64) int64 { return a ^ b }
	case operators.LUA_OPSHL:
		iFunc = number.ShiftLeft
	case operators.LUA_OPSHR:
		iFunc = number.ShiftRight
	case operators.LUA_OPUNM:
		iFunc = func(a, _ int64) int64 { return -a }
		fFunc = func(a, _ float64) float64 { return -a }
	case operators.LUA_OPBNOT:
		iFunc = func(a, _ int64) int64 { return ^a }
	default:
		panic(fmt.Sprintf("invalid arith op: %d", op))
	}

	if fFunc == nil { // bitwise operation
		if a, ok := value.ToInteger(a); ok {
			if b, ok := value.ToInteger(b); ok {
				r = value.NewInteger(iFunc(a, b))
			}
		}
	} else {
		if iFunc != nil {
			if a.Type() == value.LUA_TNUMINT && b.Type() == value.LUA_TNUMINT {
				r = value.NewInteger(iFunc(value.AsInteger(a), value.AsInteger(b)))
			}
		}

		if r == nil {
			if a, ok := value.ToNumber(a); ok {
				if b, ok := value.ToNumber(b); ok {
					r = value.NewNumber(fFunc(a, b))
				}
			}
		}
	}

	if r != nil {
		L.stackPush(r)
	} else {
		switch op {
		case operators.LUA_OPBAND, operators.LUA_OPBOR, operators.LUA_OPBXOR, operators.LUA_OPSHL, operators.LUA_OPSHR:
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
}

func (L *luaState) Compare(idx1, idx2 int, op operators.CompareOp) bool {
	a, ok1 := L.stackGet(idx1)
	b, ok2 := L.stackGet(idx2)
	if !ok1 || !ok2 {
		return false
	}
	switch op {
	case operators.LUA_OPEQ:
		return value.Equal(a, b)
	case operators.LUA_OPLT:
		if r, ok := value.LessThan(a, b); ok {
			return r
		}
	case operators.LUA_OPLE:
		if r, ok := value.LessEqual(a, b); ok {
			return r
		}
	default:
		panic(fmt.Sprintf("invalid compare op: %d", op))
	}
	panic(debug.OrderError(a, b))
}

func (L *luaState) PushNil() {
	L.stackPush(value.Nil)
}

func (L *luaState) PushNumber(n float64) {
	L.stackPush(value.NewNumber(n))
}

func (L *luaState) PushInteger(i int64) {
	L.stackPush(value.NewInteger(i))
}

func (L *luaState) PushString(s string) {
	L.stackPush(value.NewString(s))
}

func (L *luaState) PushBoolean(b bool) {
	L.stackPush(value.NewBoolean(b))
}

func (L *luaState) Concat(n int) {
	if n == 0 {
		L.stackPush(value.NewString(""))
	} else if n >= 2 {
		b := L.stackPop()
		for n > 1 {
			a := L.stackPop()
			n--
			if s1, ok := value.ToString(a); ok {
				if s2, ok := value.ToString(b); ok {
					b = value.NewString(s1 + s2)
					continue
				}
			}
			// @todo mt
			panic(debug.ConcatError(a, b))
		}
		L.stackPush(b)
	}
}

func (L *luaState) Len(idx int) {
	val, _ := L.stackGet(idx)
	switch val.Type() {
	case types.LUA_TSTRING:
		L.stackPush(value.NewInteger(int64(len(value.AsString(val)))))
	default:
		debug.TypeError(val, "get length of")
	}
}

func (L *luaState) ToNumber(idx int) float64 {
	val, _ := L.ToNumberX(idx)
	return val
}

func (L *luaState) ToInteger(idx int) int64 {
	val, _ := L.ToIntegerX(idx)
	return val
}

func (L *luaState) Pop(n int) {
	L.SetTop(-n - 1)
}

func (L *luaState) IsNil(idx int) bool {
	return L.Type(idx) == types.LUA_TNIL
}

func (L *luaState) IsBoolean(idx int) bool {
	return L.Type(idx) == types.LUA_TBOOLEAN
}

func (L *luaState) IsNone(idx int) bool {
	return L.Type(idx) == types.LUA_TNONE
}

func (L *luaState) IsNoneOrNil(idx int) bool {
	return L.Type(idx) <= types.LUA_TNIL
}

func (L *luaState) ToString(idx int) string {
	val, _ := L.ToStringX(idx)
	return val
}

func (L *luaState) Insert(idx int) {
	L.Rotate(idx, 1)
}

func (L *luaState) Remove(idx int) {
	L.Rotate(idx, -1)
	L.Pop(1)
}

func (L *luaState) Replace(idx int) {
	L.Copy(-1, idx)
	L.Pop(1)
}
