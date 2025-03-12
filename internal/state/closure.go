package state

import "github.com/uganh16/golua/internal/binary"

type lClosure struct {
	proto *binary.Proto
}

func newLuaClosure(proto *binary.Proto) *lClosure {
	return &lClosure{proto: proto}
}
