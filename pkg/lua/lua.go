package lua

import "io"

const (
	VERSION_MAJOR = 5
	VERSION_MINOR = 3
)

/* mark for precompiled code ('<esc>Lua') */
const SIGNATURE = "\x1bLua"

/* option for multiple returns */
const MULTRET = -1

type Type int

/**
 * basic types
 */
const (
	TNONE = iota - 1 // -1
	TNIL
	TBOOLEAN
	TLIGHTUSERDATA
	TNUMBER
	TSTRING
	TTABLE
	TFUNCTION
	TUSERDATA
	TTHREAD

	NUMTAGS
)

/* minimum Lua stack available to a Go function */
const MINSTACK = 20

/* type of numbers in Lua */
type Number = float64

/* type for integer functions */
type Integer = int64

type ArithOp int

const (
	OPADD  ArithOp = iota // +
	OPSUB                 // -
	OPMUL                 // *
	OPMOD                 // %
	OPPOW                 // ^
	OPDIV                 // /
	OPIDIV                // //
	OPBAND                // &
	OPBOR                 // |
	OPBXOR                // ~
	OPSHL                 // <<
	OPSHR                 // >>
	OPUNM                 // - (unary minus)
	OPBNOT                // ~
)

type CompareOp int

const (
	OPEQ CompareOp = iota // ==
	OPLT                  // <
	OPLE                  // <=
)

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
	Type(idx int) Type
	TypeName(t Type) string
	ToNumberX(idx int) (Number, bool)
	ToIntegerX(idx int) (Integer, bool)
	ToBoolean(idx int) bool
	ToStringX(idx int) (string, bool)

	/**
	 * comparison and arithmetic functions
	 */
	Arith(op ArithOp)
	Compare(idx1, idx2 int, op CompareOp) bool

	/**
	 * push functions (Go -> stack)
	 */
	PushNil()
	PushNumber(n Number)
	PushInteger(i Integer)
	PushString(s string)
	PushBoolean(b bool)

	/**
	 * get functions (Lua -> stack)
	 */

	GetTable(idx int) Type
	GetField(idx int, k string) Type
	GetI(idx int, n Integer) Type

	CreateTable(nArr, nRec int)

	/**
	 * set functions (stack -> Lua)
	 */
	SetTable(idx int)
	SetField(idx int, k string)
	SetI(idx int, n Integer)

	/**
	 * 'load' and 'call' functions (load and run Lua code)
	 */
	Call(nArgs, nResults int)
	Load(reader io.Reader, chunkName, mode string) int

	/**
	 * miscellaneous functions
	 */
	Concat(n int)
	Len(idx int)

	/**
	 * some useful macros
	 */
	ToNumber(idx int) Number
	ToInteger(idx int) Integer
	Pop(n int)
	NewTable()
	IsNil(idx int) bool
	IsBoolean(idx int) bool
	IsNone(idx int) bool
	IsNoneOrNil(idx int) bool
	ToString(idx int) string
	Insert(idx int)
	Remove(idx int)
	Replace(idx int)
}
