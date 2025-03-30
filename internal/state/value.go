package state

import (
	"fmt"
	"math"

	"github.com/uganh16/golua/internal/number"
	"github.com/uganh16/golua/pkg/lua"
)

type luaValue interface{}

var typeNames = [...]string{"no value", "nil", "boolean", "userdata", "number", "string", "table", "function", "userdata", "thread"}

func typeOf(val luaValue) lua.Type {
	switch val.(type) {
	case nil:
		return lua.TNIL
	case bool:
		return lua.TBOOLEAN
	case lua.Number, lua.Integer:
		return lua.TNUMBER
	case string:
		return lua.TSTRING
	case *luaTable:
		return lua.TTABLE
	case lua.GoFunction, *lClosure, *gClosure:
		return lua.TFUNCTION
	default:
		panic("not a Lua value")
	}
}

func typeName(val luaValue) string {
	if t, ok := val.(*luaTable); ok && t.__mt != nil {
		if name, ok := t.__mt.get("__name").(string); ok {
			return name
		}
	}
	return typeNames[typeOf(val)+1]
}

func (L *luaState) getMetatable(val luaValue) *luaTable {
	if t, ok := val.(*luaTable); ok {
		return t.__mt
	} else {
		return L.lG.mt[typeOf(val)]
	}
}

func (L *luaState) setMetatable(val luaValue, mt *luaTable) {
	if t, ok := val.(*luaTable); ok {
		t.__mt = mt
	} else {
		L.lG.mt[typeOf(val)] = mt
	}
}

func (L *luaState) getMetafield(val luaValue, field string) luaValue {
	if mt := L.getMetatable(val); mt != nil {
		return mt.get(field)
	}
	return nil
}

func toBoolean(val luaValue) bool {
	switch val := val.(type) {
	case nil:
		return false
	case bool:
		return val
	default:
		return true
	}
}

func toNumber(val luaValue) (lua.Number, bool) {
	switch val := val.(type) {
	case lua.Number:
		return val, true
	case lua.Integer:
		return lua.Number(val), true
	case string:
		return number.ParseFloat(val)
	default:
		return 0.0, false
	}
}

func toInteger(val luaValue) (lua.Integer, bool) {
	switch val := val.(type) {
	case lua.Integer:
		return val, true
	case lua.Number:
		return number.FloatToInteger(lua.Number(val))
	case string:
		if val, ok := number.ParseInteger(string(val)); ok {
			return val, ok
		}
		if val, ok := number.ParseFloat(string(val)); ok {
			return number.FloatToInteger(val)
		}
	}
	return 0, false
}

func toString(val luaValue) (string, bool) {
	switch val := val.(type) {
	case string:
		return val, true
	case lua.Number, lua.Integer:
		str := fmt.Sprintf("%v", val) // @todo
		return str, true
	default:
		return "", false
	}
}

func (L *luaState) callTM(f luaValue, args ...luaValue) luaValue {
	top := len(L.stack)
	L.stack = L.stack[:top+1+len(args)]
	L.stack[top] = f /* push function (assume EXTRA_STACK) */
	copy(L.stack[top+1:], args)
	/* @todo isLua? metamethod may yield only when called from Lua code */
	L.doCall(f, 2, 1)
	return L.stackPop()
}

func (L *luaState) callMetamethod(a, b luaValue, event string) (luaValue, bool) {
	var f luaValue
	if f = L.getMetafield(a, event); f == nil { /* try first operand */
		if f = L.getMetafield(b, event); f == nil { /* try second operand */
			return nil, false
		}
	}
	return L.callTM(f, a, b), true
}

func _arith(L *luaState, op lua.ArithOp, a, b luaValue) luaValue {
	var iFunc func(lua.Integer, lua.Integer) lua.Integer
	var fFunc func(lua.Number, lua.Number) lua.Number
	var event string

	switch op {
	case lua.OPADD:
		event = "__add"
		iFunc = func(a, b lua.Integer) lua.Integer { return a + b }
		fFunc = func(a, b lua.Number) lua.Number { return a + b }
	case lua.OPSUB:
		event = "__sub"
		iFunc = func(a, b lua.Integer) lua.Integer { return a - b }
		fFunc = func(a, b lua.Number) lua.Number { return a - b }
	case lua.OPMUL:
		event = "__mul"
		iFunc = func(a, b lua.Integer) lua.Integer { return a * b }
		fFunc = func(a, b lua.Number) lua.Number { return a * b }
	case lua.OPMOD:
		event = "__mod"
		iFunc = number.IMod
		fFunc = number.FMod
	case lua.OPPOW:
		event = "__pow"
		fFunc = math.Pow
	case lua.OPDIV:
		event = "__div"
		fFunc = func(a, b lua.Number) lua.Number { return a / b }
	case lua.OPIDIV:
		event = "__idiv"
		iFunc = number.IFloorDiv
		fFunc = number.FFloorDiv
	case lua.OPBAND:
		event = "__band"
		iFunc = func(a, b lua.Integer) lua.Integer { return a & b }
	case lua.OPBOR:
		event = "__bor"
		iFunc = func(a, b lua.Integer) lua.Integer { return a | b }
	case lua.OPBXOR:
		event = "__bxor"
		iFunc = func(a, b lua.Integer) lua.Integer { return a ^ b }
	case lua.OPSHL:
		event = "__shl"
		iFunc = number.ShiftLeft
	case lua.OPSHR:
		event = "__shr"
		iFunc = number.ShiftRight
	case lua.OPUNM:
		event = "__unm"
		iFunc = func(a, _ lua.Integer) lua.Integer { return -a }
		fFunc = func(a, _ lua.Number) lua.Number { return -a }
	case lua.OPBNOT:
		event = "__bnot"
		iFunc = func(a, _ lua.Integer) lua.Integer { return ^a }
	default:
		panic(fmt.Sprintf("invalid arith op: %d", op))
	}

	if fFunc == nil { // bitwise operation
		if a, ok := toInteger(a); ok {
			if b, ok := toInteger(b); ok {
				return iFunc(a, b)
			}
		}
	} else {
		if iFunc != nil {
			if a, ok := a.(lua.Integer); ok {
				if b, ok := b.(lua.Integer); ok {
					return iFunc(a, b)
				}
			}
		}

		if a, ok := toNumber(a); ok {
			if b, ok := toNumber(b); ok {
				return fFunc(a, b)
			}
		}
	}

	/* could not perform raw operation; try metamethod */
	if r, ok := L.callMetamethod(a, b, event); ok {
		return r
	}

	switch op {
	case lua.OPBAND, lua.OPBOR, lua.OPBXOR, lua.OPSHL, lua.OPSHR:
		_, ok1 := toNumber(a)
		_, ok2 := toNumber(b)
		if ok1 && ok2 {
			panic(toIntError(a, b))
		} else {
			panic(opIntError(a, b, "perform bitwise operation on"))
		}
	default:
		panic(opIntError(a, b, "perform arithmetic on"))
	}
}

func _eq(L *luaState, a, b luaValue) bool {
	switch a := a.(type) {
	case nil:
		return b == nil
	case bool:
		b, ok := b.(bool)
		return ok && a == b
	case string:
		b, ok := b.(string)
		return ok && a == b
	case lua.Integer:
		switch b := b.(type) {
		case lua.Integer:
			return a == b
		case lua.Number:
			return lua.Number(a) == b
		default:
			return false
		}
	case lua.Number:
		switch b := b.(type) {
		case lua.Number:
			return a == b
		case lua.Integer:
			return a == lua.Number(b)
		default:
			return false
		}
	case *luaTable:
		if b, ok := b.(*luaTable); ok {
			if a == b {
				return true
			} else if L != nil {
				if r, ok := L.callMetamethod(a, b, "__eq"); ok {
					return toBoolean(r)
				}
			}
		}
		return false
	default:
		return a == b
	}
}

func _lt(L *luaState, a, b luaValue) bool {
	switch a := a.(type) {
	case lua.Integer:
		switch b := b.(type) {
		case lua.Integer:
			return a < b
		case lua.Number:
			return lua.Number(a) < b
		}
	case lua.Number:
		switch b := b.(type) {
		case lua.Number:
			return a < b
		case lua.Integer:
			return a < lua.Number(b)
		}
	case string:
		if b, ok := b.(string); ok {
			return a < b
		}
	}
	if r, ok := L.callMetamethod(a, b, "__lt"); ok {
		return toBoolean(r)
	} else {
		panic(orderError(a, b))
	}
}

func _le(L *luaState, a, b luaValue) bool {
	switch a := a.(type) {
	case lua.Integer:
		switch b := b.(type) {
		case lua.Integer:
			return a <= b
		case lua.Number:
			return lua.Number(a) <= b
		}
	case lua.Number:
		switch b := b.(type) {
		case lua.Number:
			return a <= b
		case lua.Integer:
			return a <= lua.Number(b)
		}
	case string:
		if b, ok := b.(string); ok {
			return a <= b
		}
	}
	if r, ok := L.callMetamethod(a, b, "__le"); ok {
		return toBoolean(r)
	} else if r, ok := L.callMetamethod(b, a, "__lt"); ok {
		return !toBoolean(r)
	} else {
		panic(orderError(a, b))
	}
}

func _len(L *luaState, val luaValue) luaValue {
	if s, ok := val.(string); ok {
		return lua.Integer(len(s))
	} else if r, ok := L.callMetamethod(val, val, "__len"); ok { /* try metamethod */
		return r
	} else if t, ok := val.(*luaTable); ok {
		return lua.Integer(t.len())
	} else {
		panic(typeError(val, "get length of"))
	}
}

func _concat(L *luaState, vals []luaValue) luaValue {
	b := vals[len(vals)-1]
	for i := len(vals) - 2; i >= 0; i-- {
		a := vals[i]
		if s1, ok := toString(a); ok {
			if s2, ok := toString(b); ok {
				b = s1 + s2
				continue
			}
		}
		if r, ok := L.callMetamethod(a, b, "__concat"); ok {
			b = r
		} else {
			panic(concatError(a, b))
		}
	}
	return b
}
