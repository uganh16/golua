package value

import (
	"fmt"

	"github.com/uganh16/golua/internal/number"
	"github.com/uganh16/golua/pkg/lua/types"
)

/**
 * variant tags for strings
 */
const (
	LUA_TSHRSTR = types.LUA_TSTRING | (0 << 4)
	LUA_TLNGSTR = types.LUA_TSTRING | (1 << 4)
)

/**
 * variant tags for numbers
 */
const (
	LUA_TNUMFLT = types.LUA_TNUMBER | (0 << 4)
	LUA_TNUMINT = types.LUA_TNUMBER | (1 << 4)
)

type LuaValue interface {
	Type() int
	String() string
}

func NoVariantType(t int) types.LuaType {
	return types.LuaType(t & 0x0f)
}

type luaNil struct{}

var Nil luaNil

func (luaNil) Type() int {
	return types.LUA_TNIL
}

func (luaNil) String() string {
	return "nil"
}

type luaBoolean bool

func NewBoolean(b bool) luaBoolean {
	return luaBoolean(b)
}

func AsBoolean(val LuaValue) bool {
	return bool(val.(luaBoolean))
}

func (luaBoolean) Type() int {
	return types.LUA_TBOOLEAN
}

func (b luaBoolean) String() string {
	if b {
		return "true"
	}
	return "false"
}

func ToBoolean(val LuaValue) bool {
	switch val := val.(type) {
	case luaNil:
		return false
	case luaBoolean:
		return bool(val)
	default:
		return true
	}
}

type luaNumber float64

func NewNumber(n float64) luaNumber {
	return luaNumber(n)
}

func AsNumber(val LuaValue) float64 {
	return float64(val.(luaNumber))
}

func (luaNumber) Type() int {
	return LUA_TNUMFLT
}

func (n luaNumber) String() string {
	return fmt.Sprintf("%g", float64(n))
}

func ToNumber(val LuaValue) (float64, bool) {
	switch val := val.(type) {
	case luaNumber:
		return float64(val), true
	case luaInteger:
		return float64(val), true
	case luaString:
		return number.ParseFloat(string(val))
	default:
		return 0.0, false
	}
}

type luaInteger int64

func NewInteger(i int64) luaInteger {
	return luaInteger(i)
}

func AsInteger(val LuaValue) int64 {
	return int64(val.(luaInteger))
}

func (luaInteger) Type() int {
	return LUA_TNUMINT
}

func (i luaInteger) String() string {
	return fmt.Sprintf("%d", int64(i))
}

func ToInteger(val LuaValue) (int64, bool) {
	switch val := val.(type) {
	case luaInteger:
		return int64(val), true
	case luaNumber:
		return number.FloatToInteger(float64(val))
	case luaString:
		if val, ok := number.ParseInteger(string(val)); ok {
			return val, ok
		}
		if val, ok := number.ParseFloat(string(val)); ok {
			return number.FloatToInteger(val)
		}
	}
	return 0, false
}

type luaString string

func NewString(s string) luaString {
	return luaString(s)
}

func AsString(val LuaValue) string {
	return string(val.(luaString))
}

func (luaString) Type() int {
	return types.LUA_TSTRING
}

func (s luaString) String() string {
	return fmt.Sprintf("%q", string(s))
}

func ToString(val LuaValue) (string, bool) {
	switch val := val.(type) {
	case luaString:
		return string(val), true
	case luaNumber, luaInteger:
		str := fmt.Sprintf("%v", val)
		return str, true
	default:
		return "", false
	}
}

func Equal(a, b LuaValue) bool {
	switch a := a.(type) {
	case luaNil:
		return b == Nil
	case luaBoolean:
		b, ok := b.(luaBoolean)
		return ok && a == b
	case luaString:
		b, ok := b.(luaString)
		return ok && a == b
	case luaInteger:
		switch b := b.(type) {
		case luaInteger:
			return a == b
		case luaNumber:
			return luaNumber(a) == b
		default:
			return false
		}
	case luaNumber:
		switch b := b.(type) {
		case luaNumber:
			return a == b
		case luaInteger:
			return a == luaNumber(b)
		default:
			return false
		}
	default:
		return a == b
	}
}

func LessThan(a, b LuaValue) (bool, bool) {
	switch a := a.(type) {
	case luaString:
		if b, ok := b.(luaString); ok {
			return a < b, true
		}
	case luaInteger:
		switch b := b.(type) {
		case luaInteger:
			return a < b, true
		case luaNumber:
			return luaNumber(a) < b, true
		}
	case luaNumber:
		switch b := b.(type) {
		case luaNumber:
			return a < b, true
		case luaInteger:
			return a < luaNumber(b), true
		}
	}
	return false, false
}

func LessEqual(a, b LuaValue) (bool, bool) {
	switch a := a.(type) {
	case luaString:
		if b, ok := b.(luaString); ok {
			return a <= b, true
		}
	case luaInteger:
		switch b := b.(type) {
		case luaInteger:
			return a <= b, true
		case luaNumber:
			return luaNumber(a) <= b, true
		}
	case luaNumber:
		switch b := b.(type) {
		case luaNumber:
			return a <= b, true
		case luaInteger:
			return a <= luaNumber(b), true
		}
	}
	return false, false
}

var typeNames = [...]string{"no value", "nil", "boolean", "userdata", "number", "string", "table", "function", "userdata", "thread"}

func TypeName(t types.LuaType) string {
	return typeNames[t+1]
}

func ValueTypeName(val LuaValue) string {
	return TypeName(NoVariantType(val.Type()))
}
