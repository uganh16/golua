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
	ci := L.ci
	top := len(L.stack)
	if idx > 0 {
		if idx > ci.top-(ci.cl+1) {
			panic("unacceptable index")
		} else if ci.cl+idx >= top {
			return nil, false
		} else {
			return L.stack[ci.cl+idx], true
		}
	} else if idx == lua.REGISTRYINDEX {
		return L.lG.lRegistry, true
	} else if idx < lua.REGISTRYINDEX { /* upvalues */
		idx = lua.REGISTRYINDEX - idx
		if idx > MAXUPVAL+1 {
			panic("upvalue index too large")
		}
		f := L.stack[ci.cl]
		if _, ok := f.(*lua.GoFunction); ok {
			return nil, false
		}
		cl := f.(*gClosure)
		if idx <= len(cl.upvalue) {
			return cl.upvalue[idx-1], true
		} else {
			return nil, false
		}
	} else { /* negative index */
		if idx == 0 || -idx > top-(ci.cl+1) {
			panic("invalid index")
		}
		return L.stack[top+idx], true
	}
}

func (L *luaState) stackSet(idx int, val luaValue) {
	ci := L.ci
	top := len(L.stack)
	if idx > 0 {
		if idx > ci.top-(ci.cl+1) {
			panic("unacceptable index")
		} else if ci.cl+idx >= top {
			panic("invalid index")
		} else {
			L.stack[ci.cl+idx] = val
		}
	} else if idx == lua.REGISTRYINDEX {
		L.lG.lRegistry = val
	} else if idx < lua.REGISTRYINDEX { /* upvalues */
		idx = lua.REGISTRYINDEX - idx
		if idx > MAXUPVAL+1 {
			panic("upvalue index too large")
		}
		if cl, ok := L.stack[ci.cl].(*gClosure); ok {
			if idx <= len(cl.upvalue) {
				cl.upvalue[idx-1] = val
				return
			}
		}
		panic("invalid index")
	} else { /* negative index */
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
