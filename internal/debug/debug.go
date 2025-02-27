package debug

import (
	"fmt"

	"github.com/uganh16/golua/internal/value"
)

type RuntimeError string

func TypeError(val value.LuaValue, op string) RuntimeError {
	t := value.ValueTypeName(val)
	return RuntimeError(fmt.Sprintf("attempt to %s a %s value%s", op, t, "")) // @todo
}

func ConcatError(val1, val2 value.LuaValue) RuntimeError {
	if _, ok := value.ToString(val1); ok {
		val1 = val2
	}
	return TypeError(val1, "concatenate")
}

func OpIntError(val1, val2 value.LuaValue, msg string) RuntimeError {
	if _, ok := value.ToNumber(val1); !ok {
		val2 = val1
	}
	return TypeError(val2, msg)
}

func ToIntError(val1, val2 value.LuaValue) RuntimeError {
	if _, ok := value.ToInteger(val1); !ok {
		val2 = val1
	}
	return RuntimeError(fmt.Sprintf("number%s has no integer representation", "")) // @todo
}

func OrderError(val1, val2 value.LuaValue) RuntimeError {
	t1 := value.ValueTypeName(val1)
	t2 := value.ValueTypeName(val2)
	if t1 == t2 {
		return RuntimeError(fmt.Sprintf("attempt to compare two %s values", t1))
	} else {
		return RuntimeError(fmt.Sprintf("attempt to compare %s with %s", t1, t2))
	}
}
