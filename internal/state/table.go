package state

import (
	"math"

	"github.com/uganh16/golua/internal/number"
	"github.com/uganh16/golua/pkg/lua"
)

type luaTable struct {
	a []luaValue
	m map[luaValue]luaValue
}

func newLuaTable(nArr, nRec int) *luaTable {
	t := &luaTable{}
	if nArr > 0 {
		t.a = make([]luaValue, 0, nArr)
	}
	if nRec > 0 {
		t.m = make(map[luaValue]luaValue, nRec)
	}
	return t
}

func (t *luaTable) len() int {
	return len(t.a)
}

func (t *luaTable) get(key luaValue) luaValue {
	key = _normalizeKey(key)
	if idx, ok := key.(lua.Integer); ok {
		if 1 <= idx && idx <= lua.Integer(len(t.a)) {
			return t.a[idx-1]
		}
	}
	return t.m[key]
}

func (t *luaTable) set(key, val luaValue) {
	if key == nil {
		panic(runtimeError("table index is nil"))
	}

	if f, ok := key.(lua.Number); ok && math.IsNaN(f) {
		panic(runtimeError("table index is NaN"))
	}

	key = _normalizeKey(key)

	if idx, ok := key.(lua.Integer); ok && idx >= 1 {
		nArr := lua.Integer(len(t.a))
		if idx <= nArr {
			t.a[idx-1] = val
			if idx == nArr && val == nil {
				t._shrinkArr()
			}
			return
		}
		if idx == nArr+1 {
			delete(t.m, key)
			if val != nil {
				t.a = append(t.a, val)
				t._expandArr()
			}
			return
		}
	}

	if val != nil {
		if t.m == nil {
			t.m = make(map[luaValue]luaValue, 8)
		}
		t.m[key] = val
	} else {
		delete(t.m, key)
	}
}

func (t *luaTable) _shrinkArr() {
	var nArr int
	for nArr = len(t.a); nArr > 0; nArr-- {
		if t.a[nArr-1] != nil {
			break
		}
	}
	t.a = t.a[:nArr]
}

func (t *luaTable) _expandArr() {
	for idx := len(t.a); true; idx++ {
		if val, found := t.m[idx]; found {
			delete(t.m, idx)
			t.a = append(t.a, val)
		} else {
			break
		}
	}
}

func _normalizeKey(key luaValue) luaValue {
	if f, ok := key.(lua.Number); ok {
		if i, ok := number.FloatToInteger(f); ok {
			return i
		}
	}
	return key
}
