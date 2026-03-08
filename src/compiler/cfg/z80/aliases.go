package z80

import "zenith/compiler/cfg"

// Type aliases: make cfg types available in this package without a cfg. prefix.
// This lets the Z80 source files reference Register, InstrCategory, etc.
// directly, keeping them readable and free of cross-package noise.
type (
	Register          = cfg.Register
	InstrCategory     = cfg.InstrCategory
	AddressingMode    = cfg.AddressingMode
	CallingConvention = cfg.CallingConvention
)

// InstrCategory constants re-exported from cfg.
const (
	CatLoad       = cfg.CatLoad
	CatStore      = cfg.CatStore
	CatMove       = cfg.CatMove
	CatArithmetic = cfg.CatArithmetic
	CatBitwise    = cfg.CatBitwise
	CatBranch     = cfg.CatBranch
	CatSubroutine = cfg.CatSubroutine
	CatIO         = cfg.CatIO
	CatStack      = cfg.CatStack
	CatInterrupt  = cfg.CatInterrupt
	CatOther      = cfg.CatOther
)

// AddressingMode constants re-exported from cfg.
const (
	AddrImmediate = cfg.AddrImmediate
	AddrDirect    = cfg.AddrDirect
	AddrIndirect  = cfg.AddrIndirect
	AddrIndexed   = cfg.AddrIndexed
	AddrRelative  = cfg.AddrRelative
	AddrImplicit  = cfg.AddrImplicit
)
