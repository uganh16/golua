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
	case float64, int64:
		return lua.TNUMBER
	case string:
		return lua.TSTRING
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

func toNumber(val luaValue) (float64, bool) {
	switch val := val.(type) {
	case float64:
		return val, true
	case int64:
		return float64(val), true
	case string:
		return number.ParseFloat(val)
	default:
		return 0.0, false
	}
}

func toInteger(val luaValue) (int64, bool) {
	switch val := val.(type) {
	case int64:
		return val, true
	case float64:
		return number.FloatToInteger(float64(val))
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
	case float64, int64:
		str := fmt.Sprintf("%v", val) // @todo
		return str, true
	default:
		return "", false
	}
}

func _arith(a, b luaValue, op lua.ArithOp) luaValue {
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
		if a, ok := toInteger(a); ok {
			if b, ok := toInteger(b); ok {
				return iFunc(a, b)
			}
		}
	} else {
		if iFunc != nil {
			if a, ok := a.(int64); ok {
				if b, ok := b.(int64); ok {
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
	case int64:
		switch b := b.(type) {
		case int64:
			return a == b
		case float64:
			return float64(a) == b
		default:
			return false
		}
	case float64:
		switch b := b.(type) {
		case float64:
			return a == b
		case int64:
			return a == float64(b)
		default:
			return false
		}
	default:
		return a == b // @todo tm
	}
}

func _lt(a, b luaValue) bool {
	switch a := a.(type) {
	case int64:
		switch b := b.(type) {
		case int64:
			return a < b
		case float64:
			return float64(a) < b
		}
	case float64:
		switch b := b.(type) {
		case float64:
			return a < b
		case int64:
			return a < float64(b)
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
	case int64:
		switch b := b.(type) {
		case int64:
			return a <= b
		case float64:
			return float64(a) <= b
		}
	case float64:
		switch b := b.(type) {
		case float64:
			return a <= b
		case int64:
			return a <= float64(b)
		}
	case string:
		if b, ok := b.(string); ok {
			return a <= b
		}
	}
	// @todo tm
	panic(orderError(a, b))
}

func _len(val luaValue) int64 {
	if str, ok := val.(string); ok {
		return int64(len(str))
	}
	// @todo tm
	panic(typeError(val, "get length of"))
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
