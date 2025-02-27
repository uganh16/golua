package lua

import (
	"github.com/uganh16/golua/internal/state"
	"github.com/uganh16/golua/pkg/lua/operators"
	"github.com/uganh16/golua/pkg/lua/types"
)

const (
	LUA_VERSION_MAJOR = 5
	LUA_VERSION_MINOR = 3
)

/* mark for precompiled code ('<esc>Lua') */
const LUA_SIGNATURE = "\x1bLua"

type LuaState interface {
	/**
	 * basic stack manipulation
	 */
	AbsIndex(idx int) int
	GetTop() int
	SetTop(idx int)
	PushValue(idx int)
	Rotate(idx, n int)
	Copy(srcIdx, dstIdx int)
	CheckStack(n int) bool

	/**
	 * access functions (stack -> Go)
	 */
	IsNumber(idx int) bool
	IsString(idx int) bool
	IsInteger(idx int) bool
	Type(idx int) types.LuaType
	TypeName(t types.LuaType) string
	ToNumberX(idx int) (float64, bool)
	ToIntegerX(idx int) (int64, bool)
	ToBoolean(idx int) bool
	ToStringX(idx int) (string, bool)

	/**
	 * comparison and arithmetic functions
	 */
	Arith(op operators.ArithOp)
	Compare(idx1, idx2 int, op operators.CompareOp) bool

	/**
	 * push functions (Go -> stack)
	 */
	PushNil()
	PushNumber(n float64)
	PushInteger(i int64)
	PushString(s string)
	PushBoolean(b bool)

	/**
	 * miscellaneous functions
	 */
	Concat(n int)
	Len(idx int)

	/**
	 * some useful macros
	 */
	ToNumber(idx int) float64
	ToInteger(idx int) int64
	Pop(n int)
	IsNil(idx int) bool
	IsBoolean(idx int) bool
	IsNone(idx int) bool
	IsNoneOrNil(idx int) bool
	ToString(idx int) string
	Insert(idx int)
	Remove(idx int)
	Replace(idx int)
}

/**
 * state manipulation
 */

func NewState() LuaState {
	return state.New()
}
