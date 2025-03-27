package state

import (
	"github.com/uganh16/golua/internal/conf"
	"github.com/uganh16/golua/pkg/lua"
)

func (L *luaState) stackPush(val luaValue) {
	if len(L.stack) >= L.ci.top {
		panic("stack overflow")
	}
	L.stack = append(L.stack, val)
}

func (L *luaState) stackPop() luaValue {
	L.stackCheck(1)
	newTop := len(L.stack) - 1
	val := L.stack[newTop]
	L.stack[newTop] = nil
	L.stack = L.stack[:newTop]
	return val
}

func (L *luaState) stackGet(idx int) (luaValue, bool) {
	top := len(L.stack)
	if idx > 0 {
		if idx > L.ci.top-(L.ci.cl+1) {
			panic("unacceptable index")
		} else if L.ci.cl+idx >= top {
			return nil, false
		} else {
			return L.stack[L.ci.cl+idx], true
		}
	} else if idx == lua.REGISTRYINDEX {
		return L.lG.lRegistry, true
	} else { // @todo upvalues
		if idx == 0 || -idx > top-(L.ci.cl+1) {
			panic("invalid index")
		}
		return L.stack[top+idx], true
	}
}

func (L *luaState) stackSet(idx int, val luaValue) {
	top := len(L.stack)
	if idx > 0 {
		if idx > L.ci.top-(L.ci.cl+1) {
			panic("unacceptable index")
		} else if L.ci.cl+idx >= top {
			panic("invalid index")
		} else {
			L.stack[L.ci.cl+idx] = val
		}
	} else if idx == lua.REGISTRYINDEX {
		L.lG.lRegistry = val
	} else { // @todo upvalues
		if idx == 0 || -idx > top {
			panic("invalid index")
		}
		L.stack[top+idx] = val
	}
}

func (L *luaState) stackGrow(n int) {
	size := cap(L.stack)
	if size > conf.LUAI_MAXSTACK { /* error after extra size? */
		// @todo luaD_throw(L, LUA_ERRERR);
	}
	needed := len(L.stack) + n + EXTRA_STACK
	newSize := 2 * size
	if newSize > conf.LUAI_MAXSTACK {
		newSize = conf.LUAI_MAXSTACK
	}
	if newSize < needed {
		newSize = needed
	}
	if newSize > conf.LUAI_MAXSTACK { /* stack overflow? */
		/* some space for error handling */
		L.stackRealloc(conf.LUAI_MAXSTACK + 200)
		panic(runtimeError("stack overflow"))
	} else {
		L.stackRealloc(newSize)
	}
}

func (L *luaState) stackRealloc(newSize int) {
	newStack := make([]luaValue, len(L.stack), newSize)
	copy(newStack, L.stack)
	L.stack = newStack
	L.stackLast = newSize - EXTRA_STACK
}

func (L *luaState) stackReverse(from, to int) {
	for from < to {
		L.stack[from], L.stack[to] = L.stack[to], L.stack[from]
		from++
		to--
	}
}

func (L *luaState) stackCheck(n int) {
	if n >= len(L.stack)-L.ci.cl {
		panic("not enough elements in the stack")
	}
}
