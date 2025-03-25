package binary

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/uganh16/golua/internal/bytecode"
	"github.com/uganh16/golua/pkg/lua"
)

type reader struct {
	in io.Reader
}

func (r *reader) checkHeader() binary.ByteOrder {
	r.checkLiteral(lua.SIGNATURE, "not a")
	if r.readByte() != LUAC_VERSION {
		panic(bailoutF("version mismatch in"))
	}
	if r.readByte() != LUAC_FORMAT {
		panic(bailoutF("format mismatch in"))
	}
	r.checkLiteral(LUAC_DATA, "corrupted")
	r.checkSize(INT_SIZE, "int")
	r.checkSize(SIZE_T_SIZE, "size_t")
	r.checkSize(INSTRUCTION_SIZE, "Instruction")
	r.checkSize(LUA_INTEGER_SIZE, "lua_Integer")
	r.checkSize(LUA_NUMBER_SIZE, "lua_Number")
	var order binary.ByteOrder
	b := r.readBytes(LUA_INTEGER_SIZE)
	if binary.LittleEndian.Uint64(b) == LUAC_INT {
		order = binary.LittleEndian
	} else if binary.BigEndian.Uint64(b) == LUAC_INT {
		order = binary.BigEndian
	} else {
		panic(bailoutF("corrupted"))
	}
	if r.readFloat64(order) != LUAC_NUM {
		panic(bailoutF("float format mismatch in"))
	}
	return order
}

func (r *reader) checkLiteral(s string, msg string) {
	if string(r.readBytes(uint(len(s)))) != s {
		panic(bailoutF(msg))
	}
}

func (r *reader) checkSize(size byte, name string) {
	if r.readByte() != size {
		panic(bailoutF("%s size mismatch in", name))
	}
}

func (r *reader) readProto(order binary.ByteOrder, parentSource string) *Proto {
	source := r.readString(order)
	if source == "" {
		source = parentSource
	}
	return &Proto{
		Source:          source,
		LineDefined:     r.readUint32(order),
		LastLineDefined: r.readUint32(order),
		NumParams:       r.readByte(),
		IsVararg:        r.readByte() != 0,
		MaxStackSize:    r.readByte(),
		Code:            r.readCode(order),
		Constants:       r.readConstants(order),
		Upvalues:        r.readUpvalues(order),
		Protos:          r.readProtos(order, source),
		LineInfo:        r.readLineInfo(order),
		LocVars:         r.readLocVars(order),
		UpvalueNames:    r.readUpvalueNames(order),
	}
}

func (r *reader) readCode(order binary.ByteOrder) []bytecode.Instruction {
	code := make([]bytecode.Instruction, r.readUint32(order))
	for i := range code {
		code[i] = bytecode.Instruction(r.readUint32(order))
	}
	return code
}

func (r *reader) readConstants(order binary.ByteOrder) []interface{} {
	constants := make([]interface{}, r.readUint32(order))
	for i := range constants {
		switch r.readByte() {
		case lua.TNIL:
			constants[i] = nil
		case lua.TBOOLEAN:
			constants[i] = r.readByte() != 0
		case LUA_TNUMINT:
			constants[i] = lua.Integer(r.readUint64(order))
		case LUA_TNUMFLT:
			constants[i] = lua.Number(r.readFloat64(order))
		case LUA_TSHRSTR, LUA_TLNGSTR:
			constants[i] = r.readString(order)
		default:
			panic(bailoutF("corrupted"))
		}
	}
	return constants
}

func (r *reader) readUpvalues(order binary.ByteOrder) []Upvalue {
	upvalues := make([]Upvalue, r.readUint32(order))
	for i := range upvalues {
		upvalues[i] = Upvalue{
			InStack: r.readByte(),
			Idx:     r.readByte(),
		}
	}
	return upvalues
}

func (r *reader) readProtos(order binary.ByteOrder, parentSource string) []*Proto {
	protos := make([]*Proto, r.readUint32(order))
	for i := range protos {
		protos[i] = r.readProto(order, parentSource)
	}
	return protos
}

func (r *reader) readLineInfo(order binary.ByteOrder) []uint32 {
	lineInfo := make([]uint32, r.readUint32(order))
	for i := range lineInfo {
		lineInfo[i] = r.readUint32(order)
	}
	return lineInfo
}

func (r *reader) readLocVars(order binary.ByteOrder) []LocVar {
	locVars := make([]LocVar, r.readUint32(order))
	for i := range locVars {
		locVars[i] = LocVar{
			VarName: r.readString(order),
			StartPC: r.readUint32(order),
			EndPC:   r.readUint32(order),
		}
	}
	return locVars
}

func (r *reader) readUpvalueNames(order binary.ByteOrder) []string {
	upvalueNames := make([]string, r.readUint32(order))
	for i := range upvalueNames {
		upvalueNames[i] = r.readString(order)
	}
	return upvalueNames
}

func (r *reader) readFloat64(order binary.ByteOrder) float64 {
	return math.Float64frombits(r.readUint64(order))
}

func (r *reader) readUint32(order binary.ByteOrder) uint32 {
	return order.Uint32(r.readBytes(4))
}

func (r *reader) readUint64(order binary.ByteOrder) uint64 {
	return order.Uint64(r.readBytes(8))
}

func (r *reader) readString(order binary.ByteOrder) string {
	n := uint(r.readByte())
	if n == 0 {
		return ""
	}
	if n == 0xff { /* long string */
		n = uint(r.readUint64(order))
	}
	return string(r.readBytes(n - 1))
}

func (r *reader) readByte() byte {
	return r.readBytes(1)[0]
}

func (r *reader) readBytes(n uint) []byte {
	b := make([]byte, n)
	_, err := r.in.Read(b)
	if err != nil {
		panic(bailoutF("truncated"))
	}
	return b
}
