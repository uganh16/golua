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
	case *lClosure:
		return lua.TFUNCTION
	default:
		panic("not a Lua value")
	}
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

func _arith(a, b luaValue, op lua.ArithOp) luaValue {
	var iFunc func(lua.Integer, lua.Integer) lua.Integer
	var fFunc func(lua.Number, lua.Number) lua.Number

	switch op {
	case lua.OPADD:
		iFunc = func(a, b lua.Integer) lua.Integer { return a + b }
		fFunc = func(a, b lua.Number) lua.Number { return a + b }
	case lua.OPSUB:
		iFunc = func(a, b lua.Integer) lua.Integer { return a - b }
		fFunc = func(a, b lua.Number) lua.Number { return a - b }
	case lua.OPMUL:
		iFunc = func(a, b lua.Integer) lua.Integer { return a * b }
		fFunc = func(a, b lua.Number) lua.Number { return a * b }
	case lua.OPMOD:
		iFunc = number.IMod
		fFunc = number.FMod
	case lua.OPPOW:
		fFunc = math.Pow
	case lua.OPDIV:
		fFunc = func(a, b lua.Number) lua.Number { return a / b }
	case lua.OPIDIV:
		iFunc = number.IFloorDiv
		fFunc = number.FFloorDiv
	case lua.OPBAND:
		iFunc = func(a, b lua.Integer) lua.Integer { return a & b }
	case lua.OPBOR:
		iFunc = func(a, b lua.Integer) lua.Integer { return a | b }
	case lua.OPBXOR:
		iFunc = func(a, b lua.Integer) lua.Integer { return a ^ b }
	case lua.OPSHL:
		iFunc = number.ShiftLeft
	case lua.OPSHR:
		iFunc = number.ShiftRight
	case lua.OPUNM:
		iFunc = func(a, _ lua.Integer) lua.Integer { return -a }
		fFunc = func(a, _ lua.Number) lua.Number { return -a }
	case lua.OPBNOT:
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

	// @todo tm

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

func _eq(a, b luaValue) bool {
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
	default:
		return a == b // @todo tm
	}
}

func _lt(a, b luaValue) bool {
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
	// @todo tm
	panic(orderError(a, b))
}

func _le(a, b luaValue) bool {
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
	// @todo tm
	panic(orderError(a, b))
}

func _len(val luaValue) lua.Integer {
	if s, ok := val.(string); ok {
		return lua.Integer(len(s))
	} else if t, ok := val.(*luaTable); ok {
		return lua.Integer(t.len())
	} else {
		// @todo tm
		panic(typeError(val, "get length of"))
	}
}

func _concat(vals []luaValue) luaValue {
	b := vals[len(vals)-1]
	for i := len(vals) - 2; i >= 0; i-- {
		a := vals[i]
		if s1, ok := toString(a); ok {
			if s2, ok := toString(b); ok {
				b = s1 + s2
				continue
			}
		}
		// @todo mt
		panic(concatError(a, b))
	}
	return b
}
