package cfg

// Register represents a physical register
type Register struct {
	Name        string
	Size        int         // 8 or 16 bits
	Composition []*Register // For multi-byte registers (typical Intel and Zilog)
	RegisterId  int         // the register id for encoding
}

// InstrCategory categorizes instructions for scheduling and optimization
type InstrCategory uint8

const (
	CatLoad       InstrCategory = iota // Load from memory or immediate
	CatStore                           // Store to memory
	CatMove                            // Register-to-register transfers
	CatArithmetic                      // Arithmetic operations (add, subtract, multiply, divide)
	CatBitwise                         // Bitwise and logical operations (and, or, xor, shift, rotate, bit test/set/clear)
	CatBranch                          // Conditional and unconditional branches/jumps
	CatSubroutine                      // Subroutine call and return
	CatIO                              // Input/output operations
	CatStack                           // Stack operations (push, pop)
	CatInterrupt                       // Interrupt control (enable, disable, return from interrupt)
	CatOther                           // Other CPU-specific instructions (nop, halt, etc.)
)

// ============================================================================
// Addressing Modes
// ============================================================================

type AddressingMode uint8

const (
	AddrImmediate AddressingMode = 1 << 0 // Immediate/literal operand
	AddrDirect    AddressingMode = 1 << 1 // Direct memory address
	AddrIndirect  AddressingMode = 1 << 2 // Register indirect (memory through register)
	AddrIndexed   AddressingMode = 1 << 3 // Indexed addressing (base register + offset)
	AddrRelative  AddressingMode = 1 << 4 // PC-relative addressing
	AddrImplicit  AddressingMode = 1 << 5 // No explicit operands
)