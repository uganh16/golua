package value

import (
	"fmt"

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
	Type() types.LuaType
	String() string
}

type luaNil struct{}

var Nil luaNil

func (luaNil) Type() types.LuaType {
	return types.LUA_TNIL
}

func (luaNil) String() string {
	return "nil"
}

type luaBoolean bool

func NewBoolean(b bool) luaBoolean {
	return luaBoolean(b)
}

func (luaBoolean) Type() types.LuaType {
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

func (luaNumber) Type() types.LuaType {
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
	/* @todo string convertible to number? */
	default:
		return 0.0, false
	}
}

type luaInteger int64

func NewInteger(i int64) luaInteger {
	return luaInteger(i)
}

func (luaInteger) Type() types.LuaType {
	return LUA_TNUMINT
}

func (i luaInteger) String() string {
	return fmt.Sprintf("%d", int64(i))
}

func ToInteger(val LuaValue) (int64, bool) {
	switch val := val.(type) {
	case luaInteger:
		return int64(val), true
	/* @todo try to convert a value to an integer */
	default:
		return 0, false
	}
}

type luaString string

func NewString(s string) luaString {
	return luaString(s)
}

func (luaString) Type() types.LuaType {
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
