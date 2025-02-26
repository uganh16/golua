package binary

import (
	"fmt"
	"os"

	"github.com/uganh16/golua/internal/vm"
	"github.com/uganh16/golua/pkg/lua"
)

const LUAC_VERSION = lua.LUA_VERSION_MAJOR*16 + lua.LUA_VERSION_MINOR

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

const (
	LUA_TNIL     = 0x00
	LUA_TBOOLEAN = 0x01
	LUA_TNUMFLT  = 0x03
	LUA_TNUMINT  = 0x13
	LUA_TSHRSTR  = 0x04
	LUA_TLNGSTR  = 0x14
)

type Proto struct {
	Source          string
	LineDefined     uint32
	LastLineDefined uint32
	NumParams       byte
	IsVararg        bool
	MaxStackSize    byte
	Code            []vm.Instruction
	Constants       []interface{}
	Upvalues        []Upvalue
	Protos          []*Proto
	LineInfo        []uint32
	LocVars         []LocVar
	UpvalueNames    []string
}

type Upvalue struct {
	InStack byte
	Idx     byte
}

type LocVar struct {
	VarName string
	StartPC uint32
	EndPC   uint32
}

type bailout string

func bailoutF(format string, a ...any) bailout {
	return bailout(fmt.Sprintf(format, a...))
}

func Undump(file *os.File) (proto *Proto, err error) {
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

	r := &reader{file}
	order := r.checkHeader()
	r.readByte() // size_upvalues
	proto = r.readProto(order, "")
	return
}
