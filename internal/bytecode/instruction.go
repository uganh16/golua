package bytecode

type Instruction uint32

const MAXARG_Bx = (1 << 18) - 1
const MAXARG_sBx = MAXARG_Bx >> 1

func (i Instruction) Opcode() int {
	return int(i & 0x3f)
}

func (i Instruction) ABC() (a, b, c int) {
	a = int((i >> 6) & 0xff)
	c = int((i >> 14) & 0x1ff)
	b = int((i >> 23) & 0x1ff)
	return
}

func (i Instruction) ABx() (a, bx int) {
	a = int((i >> 6) & 0xff)
	bx = int(i >> 14)
	return
}

func (i Instruction) AsBx() (a, sbx int) {
	a = int((i >> 6) & 0xff)
	sbx = int(i>>14) - MAXARG_sBx
	return
}

func (i Instruction) Ax() (ax int) {
	return int(i >> 6)
}

func (i Instruction) OpName() string {
	return opcodes[i.Opcode()].name
}

func (i Instruction) OpMode() byte {
	return opcodes[i.Opcode()].mode
}

func (i Instruction) BMode() byte {
	return opcodes[i.Opcode()].argBMode
}

func (i Instruction) CMode() byte {
	return opcodes[i.Opcode()].argCMode
}
