package state

import (
	"fmt"
)

type runtimeError string

func typeError(val luaValue, op string) runtimeError {
	return runtimeError(fmt.Sprintf("attempt to %s a %s value%s", op, typeName(val), "")) // @todo varinfo
}

func concatError(val1, val2 luaValue) runtimeError {
	if _, ok := toString(val1); ok {
		val1 = val2
	}
	return typeError(val1, "concatenate")
}

func opIntError(val1, val2 luaValue, msg string) runtimeError {
	if _, ok := toNumber(val1); !ok {
		val2 = val1
	}
	return typeError(val2, msg)
}

func toIntError(val1, val2 luaValue) runtimeError {
	if _, ok := toInteger(val1); !ok {
		val2 = val1
	}
	return runtimeError(fmt.Sprintf("number%s has no integer representation", "")) // @todo varinfo
}

func orderError(val1, val2 luaValue) runtimeError {
	t1 := typeName(val1)
	t2 := typeName(val2)
	if t1 == t2 {
		return runtimeError(fmt.Sprintf("attempt to compare two %s values", t1))
	} else {
		return runtimeError(fmt.Sprintf("attempt to compare %s with %s", t1, t2))
	}
}
