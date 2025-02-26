package main

import (
	"fmt"
	"os"

	"github.com/uganh16/golua/internal/binary"
	"github.com/uganh16/golua/internal/value"
	"github.com/uganh16/golua/internal/vm"
	"github.com/uganh16/golua/pkg/lua"
)

func main() {
	for _, file := range os.Args[1:] {
		var p *value.Proto
		f, err := os.Open(file)
		if err == nil {
			p, err = binary.Undump(f)
			f.Close()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", file, err)
			continue
		}
		list(p)
	}
}

func list(p *value.Proto) {
	printHeader(p)
	printCode(p)
	printDebug(p)
	for _, p := range p.Protos {
		list(p)
	}
}

func printHeader(p *value.Proto) {
	funcType := "main"
	if p.LineDefined > 0 {
		funcType = "function"
	}

	source := p.Source
	if source == "" {
		source = "=?"
	}
	if source[0] == '@' || source[0] == '=' {
		source = source[1:]
	} else if source[0] == lua.LUA_SIGNATURE[0] {
		source = "(bstring)"
	} else {
		source = "(string)"
	}

	varargFlag := ""
	if p.IsVararg {
		varargFlag = "+"
	}

	fmt.Printf("\n%s <%s:%d,%d> (%d instruction%s)\n", funcType, source, p.LineDefined, p.LastLineDefined, len(p.Code), s(len(p.Code)))
	fmt.Printf("%d%s param%s, %d slot%s, %d upvalue%s, %d local%s, %d constant%s, %d function%s\n", p.NumParams, varargFlag, s(int(p.NumParams)), p.MaxStackSize, s(int(p.MaxStackSize)), len(p.Upvalues), s(len(p.Upvalues)), len(p.LocVars), s(len(p.LocVars)), len(p.Constants), s(len(p.Constants)), len(p.Protos), s(len(p.Protos)))
}

func printCode(p *value.Proto) {
	for pc, i := range p.Code {
		line := "-"
		if len(p.LineInfo) > pc {
			line = fmt.Sprintf("%d", p.LineInfo[pc])
		}
		fmt.Printf("\t%d\t[%s]\t%-9s\t", pc+1, line, i.OpName())
		switch i.OpMode() {
		case vm.IABC:
			a, b, c := i.ABC()
			fmt.Printf("%d", a)
			if i.BMode() != vm.OpArgN {
				if b > 0xff {
					fmt.Printf(" %d", -1-(b&0xff))
				} else {
					fmt.Printf(" %d", b)
				}
			}
			if i.CMode() != vm.OpArgN {
				if c > 0xff {
					fmt.Printf(" %d", -1-(c&0xff))
				} else {
					fmt.Printf(" %d", c)
				}
			}
		case vm.IABx:
			a, bx := i.ABx()
			fmt.Printf("%d", a)
			switch i.BMode() {
			case vm.OpArgK:
				fmt.Printf(" %d", -1-bx)
			case vm.OpArgU:
				fmt.Printf(" %d", bx)
			}
		case vm.IAsBx:
			a, sbx := i.AsBx()
			fmt.Printf("%d %d", a, sbx)
		case vm.IAx:
			ax := i.Ax()
			fmt.Printf("%d", -1-ax)
		}
		fmt.Printf("\n")
	}
}

func printDebug(p *value.Proto) {
	fmt.Printf("constants (%d):\n", len(p.Constants))
	for i, k := range p.Constants {
		fmt.Printf("\t%d\t%s\n", i+1, k)
	}

	fmt.Printf("locals (%d):\n", len(p.LocVars))
	for i, locVar := range p.LocVars {
		fmt.Printf("\t%d\t%s\t%d\t%d\n", i, locVar.VarName, locVar.StartPC+1, locVar.EndPC+1)
	}

	fmt.Printf("upvalues (%d):\n", len(p.Upvalues))
	for i, upvalue := range p.Upvalues {
		upvalueName := "-"
		if len(p.UpvalueNames) > 0 {
			upvalueName = p.UpvalueNames[i]
		}
		fmt.Printf("\t%d\t%s\t%d\t%d\n", i, upvalueName, upvalue.InStack, upvalue.Idx)
	}
}

func s(n int) string {
	if n != 1 {
		return "s"
	}
	return ""
}
