package closure

import (
	"github.com/uganh16/golua/internal/value"
	"github.com/uganh16/golua/internal/vm"
)

type Proto struct {
	Source          string
	LineDefined     uint32
	LastLineDefined uint32
	NumParams       byte
	IsVararg        bool
	MaxStackSize    byte
	Code            []vm.Instruction
	Constants       []value.LuaValue
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
