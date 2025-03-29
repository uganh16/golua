package state

import (
	"github.com/uganh16/golua/internal/binary"
	"github.com/uganh16/golua/pkg/lua"
)

/**
 * maximum number of upvalues in a closure (both Go and Lua)
 */
const MAXUPVAL = 255

type upvalue struct {
	level int
	next  *upvalue /* linked list (when open) */
	value luaValue /* the value (when closed) */
}

func (uv *upvalue) get(L *luaState) luaValue {
	if uv.level < 0 {
		return uv.value
	} else {
		return L.stack[uv.level]
	}
}

func (uv *upvalue) set(L *luaState, val luaValue) {
	if uv.level < 0 {
		uv.value = val
	} else {
		L.stack[uv.level] = val
	}
}

type lClosure struct {
	proto  *binary.Proto
	upvals []*upvalue /* list of upvalues */
}

type gClosure struct {
	f       lua.GoFunction
	upvalue []luaValue
}

func newLuaClosure(proto *binary.Proto) *lClosure {
	cl := &lClosure{proto: proto}
	if nUpvalues := len(proto.Upvalues); nUpvalues > 0 {
		cl.upvals = make([]*upvalue, nUpvalues)
	}
	return cl
}

func newGoClosure(f lua.GoFunction, nUpvals int) *gClosure {
	return &gClosure{
		f:       f,
		upvalue: make([]luaValue, nUpvals),
	}
}

func (L *luaState) findUpvalue(level int) *upvalue {
	pp := &L.openUpval
	for *pp != nil && (*pp).level >= level {
		if p := *pp; p.level == level { /* found a corresponding upvalue? */
			return p
		} else {
			pp = &p.next
		}
	}
	/* not found: create a new upvalue */
	uv := &upvalue{
		level: level, /* current value lives in the stack */
		next:  *pp,   /* link it to list of open upvalues */
	}
	*pp = uv
	return uv
}

func (L *luaState) closeUpvalues(level int) {
	for L.openUpval != nil && L.openUpval.level >= level {
		uv := L.openUpval
		uv.value = L.stack[uv.level]
		uv.level = -1
		uv.next = nil
		L.openUpval = uv.next
	}
}
