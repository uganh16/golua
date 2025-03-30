package state

import (
	"math"

	"github.com/uganh16/golua/internal/number"
	"github.com/uganh16/golua/pkg/lua"
)

type luaTable struct {
	__mt *luaTable
	_arr []luaValue
	_map map[luaValue]luaValue
}

func newLuaTable(nArr, nRec int) *luaTable {
	t := &luaTable{}
	if nArr > 0 {
		t._arr = make([]luaValue, 0, nArr)
	}
	if nRec > 0 {
		t._map = make(map[luaValue]luaValue, nRec)
	}
	return t
}

func (t *luaTable) len() int {
	return len(t._arr)
}

func (t *luaTable) get(key luaValue) luaValue {
	key = _normalizeKey(key)
	if idx, ok := key.(lua.Integer); ok {
		if 1 <= idx && idx <= lua.Integer(len(t._arr)) {
			return t._arr[idx-1]
		}
	}
	return t._map[key]
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
		nArr := lua.Integer(len(t._arr))
		if idx <= nArr {
			t._arr[idx-1] = val
			if idx == nArr && val == nil {
				t._shrinkArr()
			}
			return
		}
		if idx == nArr+1 {
			delete(t._map, key)
			if val != nil {
				t._arr = append(t._arr, val)
				t._expandArr()
			}
			return
		}
	}

	if val != nil {
		if t._map == nil {
			t._map = make(map[luaValue]luaValue, 8)
		}
		t._map[key] = val
	} else {
		delete(t._map, key)
	}
}

func (t *luaTable) _shrinkArr() {
	var nArr int
	for nArr = len(t._arr); nArr > 0; nArr-- {
		if t._arr[nArr-1] != nil {
			break
		}
	}
	t._arr = t._arr[:nArr]
}

func (t *luaTable) _expandArr() {
	for idx := len(t._arr); true; idx++ {
		if val, found := t._map[idx]; found {
			delete(t._map, idx)
			t._arr = append(t._arr, val)
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
