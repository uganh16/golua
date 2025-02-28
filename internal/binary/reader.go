package binary

import (
	"encoding/binary"
	"math"
	"os"

	"github.com/uganh16/golua/internal/value"
	"github.com/uganh16/golua/internal/value/closure"
	"github.com/uganh16/golua/internal/vm"
	"github.com/uganh16/golua/pkg/lua"
)

type reader struct {
	file *os.File
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

func (r *reader) readProto(order binary.ByteOrder, parentSource string) *closure.Proto {
	source := r.readString(order)
	if source == "" {
		source = parentSource
	}
	return &closure.Proto{
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

func (r *reader) readCode(order binary.ByteOrder) []vm.Instruction {
	code := make([]vm.Instruction, r.readUint32(order))
	for i := range code {
		code[i] = vm.Instruction(r.readUint32(order))
	}
	return code
}

func (r *reader) readConstants(order binary.ByteOrder) []value.LuaValue {
	constants := make([]value.LuaValue, r.readUint32(order))
	for i := range constants {
		switch r.readByte() {
		case lua.TNIL:
			constants[i] = value.Nil
		case lua.TBOOLEAN:
			constants[i] = value.NewBoolean(r.readByte() != 0)
		case value.LUA_TNUMINT:
			constants[i] = value.NewInteger(int64(r.readUint64(order)))
		case value.LUA_TNUMFLT:
			constants[i] = value.NewNumber(r.readFloat64(order))
		case value.LUA_TSHRSTR, value.LUA_TLNGSTR:
			constants[i] = value.NewString(r.readString(order))
		default:
			panic(bailoutF("corrupted"))
		}
	}
	return constants
}

func (r *reader) readUpvalues(order binary.ByteOrder) []closure.Upvalue {
	upvalues := make([]closure.Upvalue, r.readUint32(order))
	for i := range upvalues {
		upvalues[i] = closure.Upvalue{
			InStack: r.readByte(),
			Idx:     r.readByte(),
		}
	}
	return upvalues
}

func (r *reader) readProtos(order binary.ByteOrder, parentSource string) []*closure.Proto {
	protos := make([]*closure.Proto, r.readUint32(order))
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

func (r *reader) readLocVars(order binary.ByteOrder) []closure.LocVar {
	locVars := make([]closure.LocVar, r.readUint32(order))
	for i := range locVars {
		locVars[i] = closure.LocVar{
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
	_, err := r.file.Read(b)
	if err != nil {
		panic(bailoutF("truncated"))
	}
	return b
}
