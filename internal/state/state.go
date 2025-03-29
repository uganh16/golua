package state

import (
	"fmt"
	"io"

	"github.com/uganh16/golua/internal/binary"
	"github.com/uganh16/golua/internal/conf"
	"github.com/uganh16/golua/pkg/lua"
)

/* test for pseudo index */
func isPseudo(idx int) bool {
	return idx <= lua.REGISTRYINDEX
}

/* extra stack space to handle TM calls and some other extras */
const EXTRA_STACK = 5

const BASIC_STACK_SIZE = 2 * lua.MINSTACK

type callInfo struct {
	cl   int       /* function index in the stack */
	top  int       /* top for this function */
	prev *callInfo /* dynamic call link */

	/* only for Lua functions */
	base int /* base for this function */
	pc   int

	nResults   int16 /* expected number of results from this function */
	callStatus uint16
}

/**
 * Bits in callInfo status
 */
const (
	CIST_OAH    = 1 << iota /* original value of 'allowhook' */
	CIST_LUA                /* call is running a Lua function */
	CIST_HOOKED             /* call is running a debug hook */
	CIST_FRESH              /* call is running on a fresh invocation */
	CIST_YPCALL
	CIST_TAIL
	CIST_HOOKYIELD
	CIST_LEQ
	CIST_FIN
)

type global_State struct {
	lRegistry luaValue
}

type luaState struct {
	stack     []luaValue
	stackLast int      /* last free slot in the stack */
	openUpval *upvalue /* list of open upvalues in this stack */
	baseCI    callInfo
	ci        *callInfo
	lG        *global_State
}

func New() *luaState {
	L := &luaState{
		stack:     make([]luaValue, 1, BASIC_STACK_SIZE),
		stackLast: BASIC_STACK_SIZE - EXTRA_STACK,
		baseCI: callInfo{
			cl:         0,
			top:        1 + lua.MINSTACK,
			prev:       nil,
			callStatus: 0,
		},
		lG: &global_State{},
	}
	L.ci = &L.baseCI

	/* init_registry: Create registry table and its predefined values */
	registry := newLuaTable(lua.RIDX_LAST, 0)
	L.lG.lRegistry = registry
	/* registry[lua.RIDX_MAINTHREAD] = L */
	registry.set(lua.Integer(lua.RIDX_MAINTHREAD), L)
	/* registry[lua.RIDX_GLOBALS] = table of globals */
	registry.set(lua.Integer(lua.RIDX_GLOBALS), newLuaTable(0, 0))

	return L
}

func (L *luaState) AbsIndex(idx int) int {
	if idx > 0 || isPseudo(idx) {
		return idx
	} else {
		return len(L.stack) - L.ci.cl + idx
	}
}

func (L *luaState) GetTop() int {
	return len(L.stack) - (L.ci.cl + 1)
}

func (L *luaState) SetTop(idx int) {
	cl := L.ci.cl
	var newTop int
	if idx >= 0 {
		if idx > L.stackLast-(cl+1) {
			panic("new top too large")
		}
		for len(L.stack) < (cl+1)+idx {
			L.stack = append(L.stack, nil)
		}
		newTop = (cl + 1) + idx
	} else {
		if -(idx + 1) > len(L.stack)-(cl+1) {
			panic("invalid new top")
		}
		newTop += len(L.stack) + idx + 1
	}
	for i := newTop; i < len(L.stack); i++ {
		L.stack[i] = nil
	}
	L.stack = L.stack[:newTop]
}

func (L *luaState) PushValue(idx int) {
	val, _ := L.stackGet(idx)
	L.stackPush(val)
}

func (L *luaState) Rotate(idx, n int) {
	top := len(L.stack)
	t := top - 1 // end of stack segment being rotated
	var p int    // start of segment
	if idx > 0 && idx <= top-(L.ci.cl+1) {
		p = L.ci.cl + idx
	} else if idx < 0 && -idx <= top-(L.ci.cl+1) {
		p = top + idx
	} else {
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
}

func (L *luaState) CheckStack(n int) bool {
	if n < 0 {
		panic("negative 'n'")
	}
	res := true
	top := len(L.stack)
	if L.stackLast-top < n {
		if top+EXTRA_STACK > conf.LUAI_MAXSTACK-n {
			res = false
		} else { /* try to grow stack */
			res = L.protectedRun(func() {
				L.stackGrow(n)
			})
		}
	}
	if res && L.ci.top < top+n {
		L.ci.top = top + n /* adjust frame top */
	}
	return res
}

func (L *luaState) IsNumber(idx int) bool {
	_, ok := L.ToNumberX(idx)
	return ok
}

func (L *luaState) IsString(idx int) bool {
	t := L.Type(idx)
	return t == lua.TSTRING || t == lua.TNUMBER
}

func (L *luaState) IsGoFunction(idx int) bool {
	val, _ := L.stackGet(idx)
	switch val.(type) {
	case lua.GoFunction, *gClosure:
		return true
	default:
		return false
	}
}

func (L *luaState) IsInteger(idx int) bool {
	val, _ := L.stackGet(idx)
	_, ok := val.(lua.Integer)
	return ok
}

func (L *luaState) Type(idx int) lua.Type {
	if val, ok := L.stackGet(idx); ok {
		return typeOf(val)
	}
	return lua.TNONE
}

func (L *luaState) TypeName(t lua.Type) string {
	if t < lua.TNONE || t >= lua.NUMTAGS {
		panic("invalid tag")
	}
	return typeNames[t+1]
}

func (L *luaState) ToNumberX(idx int) (lua.Number, bool) {
	val, _ := L.stackGet(idx)
	return toNumber(val)
}

func (L *luaState) ToIntegerX(idx int) (lua.Integer, bool) {
	val, _ := L.stackGet(idx)
	return toInteger(val)
}

func (L *luaState) ToBoolean(idx int) bool {
	val, _ := L.stackGet(idx)
	return toBoolean(val)
}

func (L *luaState) ToStringX(idx int) (string, bool) {
	val, _ := L.stackGet(idx)
	if str, ok := val.(string); ok {
		return str, true
	}
	str, ok := toString(val)
	if ok {
		L.stackSet(idx, str)
	}
	return str, ok
}

func (L *luaState) ToGoFunction(idx int) lua.GoFunction {
	val, _ := L.stackGet(idx)
	if f, ok := val.(lua.GoFunction); ok {
		return f
	} else if cl, ok := val.(*gClosure); ok {
		return cl.f
	} else {
		return nil
	}
}

func (L *luaState) Arith(op lua.ArithOp) {
	var a, b luaValue
	b = L.stackPop()
	if op != lua.OPUNM && op != lua.OPBNOT {
		a = L.stackPop()
	} else {
		a = b
	}
	L.stackPush(_arith(a, b, op))
}

func (L *luaState) Compare(idx1, idx2 int, op lua.CompareOp) bool {
	a, ok1 := L.stackGet(idx1)
	b, ok2 := L.stackGet(idx2)
	if !ok1 || !ok2 {
		return false
	}
	switch op {
	case lua.OPEQ:
		return _eq(a, b)
	case lua.OPLT:
		return _lt(a, b)
	case lua.OPLE:
		return _le(a, b)
	default:
		panic(fmt.Sprintf("invalid compare op: %d", op))
	}
}

func (L *luaState) PushNil() {
	L.stackPush(nil)
}

func (L *luaState) PushNumber(n lua.Number) {
	L.stackPush(n)
}

func (L *luaState) PushInteger(i lua.Integer) {
	L.stackPush(i)
}

func (L *luaState) PushString(s string) {
	L.stackPush(s)
}

func (L *luaState) PushGoClosure(f lua.GoFunction, n int) {
	if n == 0 {
		L.stackPush(f)
	} else {
		L.stackCheck(n)
		if n > MAXUPVAL {
			panic("upvalue index too large")
		}
		cl := newGoClosure(f, n)
		newTop := len(L.stack) - n
		for n > 0 {
			n--
			cl.upvalue[n] = L.stack[newTop+n]
			L.stack[newTop+n] = nil
		}
		L.stack = L.stack[:newTop]
		L.stackPush(cl)
	}
}

func (L *luaState) PushBoolean(b bool) {
	L.stackPush(b)
}

func (L *luaState) GetGlobal(name string) lua.Type {
	reg := L.lG.lRegistry.(*luaTable)
	return L.getTableAux(reg.get(lua.Integer(lua.RIDX_GLOBALS)), name)
}

func (L *luaState) GetTable(idx int) lua.Type {
	t, _ := L.stackGet(idx)
	k := L.stackPop() // @todo do not pop?
	return L.getTableAux(t, k)
}

func (L *luaState) GetField(idx int, k string) lua.Type {
	t, _ := L.stackGet(idx)
	return L.getTableAux(t, k)
}

func (L *luaState) GetI(idx int, n lua.Integer) lua.Type {
	t, _ := L.stackGet(idx)
	return L.getTableAux(t, n)
}

func (L *luaState) CreateTable(nArr, nRec int) {
	L.stackPush(newLuaTable(nArr, nRec))
}

func (L *luaState) SetGlobal(name string) {
	reg := L.lG.lRegistry.(*luaTable)
	v := L.stackPop()
	L.setTable(reg.get(lua.Integer(lua.RIDX_GLOBALS)), name, v)
}

func (L *luaState) SetTable(idx int) {
	t, _ := L.stackGet(idx)
	v := L.stackPop()
	k := L.stackPop()
	L.setTable(t, k, v)
}

func (L *luaState) SetField(idx int, k string) {
	t, _ := L.stackGet(idx)
	v := L.stackPop()
	L.setTable(t, k, v)
}

func (L *luaState) SetI(idx int, n lua.Integer) {
	t, _ := L.stackGet(idx)
	v := L.stackPop()
	L.setTable(t, n, v)
}

func (L *luaState) Call(nArgs, nResults int) {
	// @todo "cannot use continuations inside hooks"
	L.stackCheck(nArgs + 1)
	// @todo check L.status == LUA_OK
	if nResults != lua.MULTRET && L.ci.top-len(L.stack) < nResults-nArgs-1 {
		panic("results from function overflow current stack size")
	}
	f, _ := L.stackGet(-(nArgs + 1))
	// @todo need to prepare continuation?
	if !L.preCall(f, nArgs, nResults) { // --> luaD_callnoyield
		L.execute()
	}
	if nResults == lua.MULTRET && L.ci.top < len(L.stack) {
		L.ci.top = len(L.stack)
	}
}

func (L *luaState) Load(reader io.Reader, chunkName, mode string) int {
	if proto, err := binary.Undump(reader); err == nil {
		cl := newLuaClosure(proto)
		L.stackPush(cl)
		/* fill a closure with new closed upvalues */
		for i := range cl.upvals {
			cl.upvals[i] = &upvalue{
				level: -1, /* make it closed */
				value: nil,
			}
		}
		if len(cl.upvals) > 0 { /* does it have an upvalue? */
			/* get global table from registry */
			reg := L.lG.lRegistry.(*luaTable)
			gt := reg.get(lua.Integer(lua.RIDX_GLOBALS))
			/* set global table as 1st upvalue of 'cl' (may be LUA_ENV) */
			cl.upvals[0].value = gt
		}
	}
	// @todo
	return 0
}

func (L *luaState) Concat(n int) {
	L.stackCheck(n)
	if n == 0 {
		L.stackPush("")
	} else if n >= 2 {
		top := len(L.stack)
		res := _concat(L.stack[top-n : top])
		for i := 1; i <= n; i++ {
			L.stack[top-i] = nil
		}
		L.stack = L.stack[:top-n]
		L.stackPush(res)
	}
}

func (L *luaState) Len(idx int) {
	val, _ := L.stackGet(idx)
	L.stackPush(_len(val))
}

func (L *luaState) ToNumber(idx int) lua.Number {
	val, _ := L.ToNumberX(idx)
	return val
}

func (L *luaState) ToInteger(idx int) lua.Integer {
	val, _ := L.ToIntegerX(idx)
	return val
}

func (L *luaState) Pop(n int) {
	L.SetTop(-n - 1)
}

func (L *luaState) NewTable() {
	L.CreateTable(0, 0)
}

func (L *luaState) Register(n string, f lua.GoFunction) {
	L.PushGoFunction(f)
	L.SetGlobal(n)
}

func (L *luaState) PushGoFunction(f lua.GoFunction) {
	L.PushGoClosure(f, 0)
}

func (L *luaState) IsNil(idx int) bool {
	return L.Type(idx) == lua.TNIL
}

func (L *luaState) IsBoolean(idx int) bool {
	return L.Type(idx) == lua.TBOOLEAN
}

func (L *luaState) IsNone(idx int) bool {
	return L.Type(idx) == lua.TNONE
}

func (L *luaState) IsNoneOrNil(idx int) bool {
	return L.Type(idx) <= lua.TNIL
}

func (L *luaState) PushGlobalTable() {
	// @todo use RawGetI
	t, _ := L.stackGet(lua.REGISTRYINDEX)
	if t, ok := t.(*luaTable); ok {
		L.stackPush(t.get(lua.Integer(lua.RIDX_GLOBALS)))
	}
	panic("table expected")
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

func (L *luaState) protectedRun(f func()) (ok bool) {
	defer func() {
		switch x := recover().(type) {
		case nil:
			// no panic
		case runtimeError:
			ok = false
		default:
			panic(x)
		}
	}()
	f()
	return true
}

func (L *luaState) getTable(t, k luaValue) luaValue {
	if t, ok := t.(*luaTable); ok {
		return t.get(k)
	}
	panic(typeError(t, "index"))
}

func (L *luaState) getTableAux(t, k luaValue) lua.Type {
	v := L.getTable(t, k)
	L.stackPush(v)
	return typeOf(v)
}

func (L *luaState) setTable(t, k, v luaValue) {
	if t, ok := t.(*luaTable); ok {
		t.set(k, v)
		return
	}
	panic(typeError(t, "index"))
}

func (L *luaState) preCall(val luaValue, nArgs, nResults int) bool {
	var f lua.GoFunction
	top := len(L.stack)
	switch cl := val.(type) {
	case *gClosure:
		f = cl.f
		goto GoFunc
	case lua.GoFunction:
		f = cl
		goto GoFunc
	case *lClosure: /* Lua function: prepare its call */
		p := cl.proto
		frameSize := int(p.MaxStackSize)
		if L.stackLast-top < frameSize {
			L.stackGrow(frameSize)
		}
		nFixedArgs := int(p.NumParams)
		var base int
		stack := L.stack[:cap(L.stack)]
		if p.IsVararg {
			base = top
			/* move fixed parameters to final position */
			for i := 0; i < nFixedArgs && i < nArgs; i++ {
				stack[base+i] = L.stack[base-nArgs+i]
				L.stack[base-nArgs+i] = nil /* erase original copy (for GC) */
			}
		} else { /* non vararg function */
			base = top - nArgs
		}
		for i := nArgs; i < nFixedArgs; i++ {
			stack[base+i] = nil /* complete missing arguments */
		}
		L.stack = stack[:base+frameSize]
		L.ci = &callInfo{
			cl:         top - nArgs - 1,
			top:        base + frameSize,
			prev:       L.ci,
			base:       base,
			pc:         0,
			nResults:   int16(nResults),
			callStatus: CIST_LUA,
		}
		// @todo hookmask -> callhook
		return false
	default:
		// @todo mt
		panic(typeError(val, "call"))
	}
GoFunc:
	if L.stackLast-top < lua.MINSTACK {
		L.stackGrow(lua.MINSTACK)
	}
	L.ci = &callInfo{
		cl:         top - nArgs - 1,
		top:        top + lua.MINSTACK,
		prev:       L.ci,
		nResults:   int16(nResults),
		callStatus: 0,
	}
	// @todo hook
	n := f(L)
	L.stackCheck(n)
	L.postCall(len(L.stack)-n, n)
	return true
}

func (L *luaState) postCall(firstResult, nResults int) bool {
	ci := L.ci
	wanted := int(ci.nResults)
	// @todo L->hookmask
	L.ci = ci.prev
	/* move results to proper place */
	if wanted == lua.MULTRET {
		for i := 0; i < nResults; i++ {
			L.stack[ci.cl+i] = L.stack[firstResult+i]
		}
		L.stack = L.stack[:ci.cl+nResults] /* (!) */
		return false
	}
	for i := 0; i < wanted && i < nResults; i++ {
		L.stack[ci.cl+i] = L.stack[firstResult+i]
	}
	for i := nResults; i < wanted; i++ { /* complete wanted number of results */
		L.stack[ci.cl+i] = nil
	}
	L.stack = L.stack[:ci.cl+wanted]
	return true
}
