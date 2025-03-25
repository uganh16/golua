package number

import (
	"math"
	"strconv"

	"github.com/uganh16/golua/pkg/lua"
)

func IFloorDiv(a, b lua.Integer) lua.Integer {
	if a > 0 && b > 0 || a < 0 && b < 0 || a%b == 0 {
		return a / b
	} else {
		return a/b - 1
	}
}

func FFloorDiv(a, b lua.Number) lua.Number {
	return math.Floor(a / b)
}

func IMod(a, b lua.Integer) lua.Integer {
	return a - IFloorDiv(a, b)*b
}

func FMod(a, b lua.Number) lua.Number {
	return a - FFloorDiv(a, b)*b
}

func ShiftLeft(a, n lua.Integer) lua.Integer {
	if n >= 0 {
		return a << uint64(n)
	} else {
		return ShiftRight(a, -n)
	}
}

func ShiftRight(a, n lua.Integer) lua.Integer {
	if n >= 0 {
		return lua.Integer(uint64(a) >> uint64(n))
	} else {
		return ShiftLeft(a, -n)
	}
}

func FloatToInteger(f lua.Number) (lua.Integer, bool) {
	i := lua.Integer(f)
	return i, lua.Number(i) == f
}

func ParseInteger(s string) (lua.Integer, bool) {
	i, err := strconv.ParseInt(s, 10, 64)
	return i, err == nil
}

func ParseFloat(s string) (lua.Number, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

/**
 * converts an integer to a "floating point byte", represented as
 * (eeeeexxx), where the real value is (1xxx) * 2^(eeeee - 1) if
 * eeeee != 0 and (xxx) otherwise.
 */
func Int2fb(x uint) int {
	e := 0 /* exponent */
	if x < 8 {
		return int(x)
	}
	for x >= (8 << 4) { /* coarse steps */
		x = (x + 0xf) >> 4 /* x = ceil(x / 16) */
		e += 4
	}
	for x >= (8 << 1) { /* fine steps */
		x = (x + 1) >> 1 /* x = ceil(x / 2) */
		e++
	}
	return ((e + 1) << 3) | (int(x) - 8)
}

/* converts back */
func Fb2int(x int) int {
	if x < 8 {
		return x
	} else {
		return ((x & 7) + 8) << ((uint(x) >> 3) - 1)
	}
}
