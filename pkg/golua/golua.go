package golua

import (
	"github.com/uganh16/golua/internal/state"
	"github.com/uganh16/golua/pkg/lua"
)

func NewState() lua.State {
	return state.New()
}
