package binary

import (
	"fmt"
	"io"

	"github.com/uganh16/golua/internal/bytecode"
	"github.com/uganh16/golua/pkg/lua"
)

const LUAC_VERSION = lua.VERSION_MAJOR*16 + lua.VERSION_MINOR

/* this is the official format */
const LUAC_FORMAT = 0

/* data to catch conversion errors */
const LUAC_DATA = "\x19\x93\r\n\x1a\n"

const (
	INT_SIZE         = 4
	SIZE_T_SIZE      = 8
	INSTRUCTION_SIZE = 4
	LUA_INTEGER_SIZE = 8
	LUA_NUMBER_SIZE  = 8
)

const (
	LUAC_INT = 0x5678
	LUAC_NUM = 370.5
)

/**
 * variant tags for strings
 */
const (
	LUA_TSHRSTR = lua.TSTRING | (0 << 4)
	LUA_TLNGSTR = lua.TSTRING | (1 << 4)
)

/**
 * variant tags for numbers
 */
const (
	LUA_TNUMFLT = lua.TNUMBER | (0 << 4)
	LUA_TNUMINT = lua.TNUMBER | (1 << 4)
)

type bailout string

func bailoutF(format string, a ...any) bailout {
	return bailout(fmt.Sprintf(format, a...))
}

type Proto struct {
	Source          string
	LineDefined     uint32
	LastLineDefined uint32
	NumParams       byte
	IsVararg        bool
	MaxStackSize    byte
	Code            []bytecode.Instruction
	Constants       []interface{}
	Upvalues        []Upvalue
	Protos          []*Proto
	LineInfo        []uint32
	LocVars         []LocVar
	UpvalueNames    []string
}

type Upvalue struct {
	InStack bool
	Idx     byte
}

type LocVar struct {
	VarName string
	StartPC uint32
	EndPC   uint32
}

func Undump(in io.Reader) (proto *Proto, err error) {
	defer func() {
		switch x := recover().(type) {
		case nil:
			/* no panic */
		case bailout:
			err = fmt.Errorf("%s precompiled chunk", x)
		default:
			panic(x)
		}
	}()

	r := &reader{in}
	order := r.checkHeader()
	r.readByte() // size_upvalues
	proto = r.readProto(order, "")
	return
}
