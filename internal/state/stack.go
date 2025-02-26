package state

import "github.com/uganh16/golua/internal/value"

const LUAI_MAXSTACK = 1000000

func (L *luaState) stackPush(val value.LuaValue) {
	if len(L.stack) == cap(L.stack) {
		panic("stack overflow")
	}
	L.stack = append(L.stack, val)
}

func (L *luaState) stackGet(idx int) (value.LuaValue, bool) {
	top := len(L.stack)
	if idx > 0 {
		if idx > cap(L.stack) {
			panic("unacceptable index")
		} else if idx > top {
			return value.Nil, false
		} else {
			return L.stack[idx-1], true
		}
	} else { // @todo
		if idx == 0 || -idx > top {
			panic("invalid index")
		}
		return L.stack[top+idx], true
	}
}

func (L *luaState) stackSet(idx int, val value.LuaValue) {
	top := len(L.stack)
	if idx > 0 {
		if idx > cap(L.stack) {
			panic("unacceptable index")
		} else if idx > top {
			panic("invalid index")
		} else {
			L.stack[idx-1] = val
		}
	} else { // @todo
		if idx == 0 || -idx > top {
			panic("invalid index")
		}
		L.stack[idx-1] = val
	}
}

func (L *luaState) stackReverse(from, to int) {
	for from < to {
		L.stack[from], L.stack[to] = L.stack[to], L.stack[from]
		from++
		to--
	}
}
