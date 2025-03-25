package state

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/uganh16/golua/pkg/lua"
)

func TestStack(t *testing.T) {
	L := New()
	if L.GetTop() != 0 {
		t.Errorf("Empty stack expected: %v", L.stack)
	}
	L.PushBoolean(true)
	printStack(L)
	L.PushInteger(10)
	printStack(L)
	L.PushNil()
	printStack(L)
	L.PushString("hello")
	printStack(L)
	L.PushValue(-4)
	printStack(L)
	L.Replace(3)
	printStack(L)
	L.SetTop(6)
	printStack(L)
	L.Remove(-3)
	printStack(L)
	L.SetTop(-5)
	printStack(L)
}

func TestLuaOp(t *testing.T) {
	L := New()
	L.PushInteger(1)
	L.PushString("2.0")
	L.PushString("3.0")
	L.PushNumber(4.0)
	printStack(L)

	L.Arith(lua.OPADD)
	printStack(L)
	L.Arith(lua.OPBNOT)
	printStack(L)
	L.Len(2)
	printStack(L)
	L.Concat(3)
	printStack(L)
	L.PushBoolean(L.Compare(1, 2, lua.OPEQ))
	printStack(L)
}

func TestLuaVM(t *testing.T) {
	fileName := filepath.Join(os.TempDir(), "luac.out")
	cmd := exec.Command("luac", "-o", fileName, "../../test/sum.lua")
	if err := cmd.Run(); err != nil {
		t.Errorf("Error running command: %v", err)
	}

	L := New()
	f, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer f.Close()
	L.Load(f, "", "b")
	L.Call(0, 1)
	printStack(L)
}

func TestTable(t *testing.T) {
	fileName := filepath.Join(os.TempDir(), "luac.out")
	cmd := exec.Command("luac", "-o", fileName, "../../test/test_table.lua")
	if err := cmd.Run(); err != nil {
		t.Errorf("Error running command: %v", err)
	}

	L := New()
	f, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer f.Close()
	L.Load(f, "", "b")
	L.Call(0, 1)
	printStack(L)
}

func TestLuaFunction(t *testing.T) {
	fileName := filepath.Join(os.TempDir(), "luac.out")
	cmd := exec.Command("luac", "-o", fileName, "../../test/max.lua")
	if err := cmd.Run(); err != nil {
		t.Errorf("Error running command: %v", err)
	}

	L := New()
	f, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer f.Close()
	L.Load(f, "", "b")
	L.Call(0, 0)
	printStack(L)
}

func printStack(L *luaState) {
	for idx := 1; idx <= L.GetTop(); idx++ {
		t := L.Type(idx)
		switch t {
		case lua.TBOOLEAN:
			fmt.Printf("[%t]", L.ToBoolean(idx))
		case lua.TNUMBER:
			fmt.Printf("[%g]", L.ToNumber(idx))
		case lua.TSTRING:
			fmt.Printf("[%q]", L.ToString(idx))
		default:
			fmt.Printf("[%s]", L.TypeName(t))
		}
	}
	fmt.Println()
}
