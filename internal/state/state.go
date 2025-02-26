package state

import (
	"github.com/uganh16/golua/internal/value"
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
		return types.LuaType(int(val.Type()) & 0x0f)
	}
	return types.LUA_TNONE
}

func (L *luaState) TypeName(t types.LuaType) string {
	switch t {
	case types.LUA_TNONE:
		return "no value"
	case types.LUA_TNIL:
		return "nil"
	case types.LUA_TBOOLEAN:
		return "boolean"
	case types.LUA_TLIGHTUSERDATA:
		return "userdata"
	case types.LUA_TNUMBER:
		return "number"
	case types.LUA_TSTRING:
		return "string"
	case types.LUA_TTABLE:
		return "table"
	case types.LUA_TFUNCTION:
		return "function"
	case types.LUA_TUSERDATA:
		return "userdata"
	case types.LUA_TTHREAD:
		return "thread"
	default:
		panic("invalid tag")
	}
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
