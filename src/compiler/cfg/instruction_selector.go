package cfg

import (
	"zenith/compiler/zsm"
)

// ============================================================================
// Expression Evaluation Context
// ============================================================================

// EvalMode specifies how an expression should be evaluated
type EvalMode uint8

const (
	// ValueMode: expression must produce a VirtualRegister result
	ValueMode EvalMode = iota
	// BranchMode: expression should emit conditional branch (preferred for comparisons)
	BranchMode
)

// ExprContext provides context for expression evaluation
// Enables short-circuit evaluation and direct flag-to-branch conversion
type ExprContext struct {
	// Evaluation mode
	Mode EvalMode

	// For BranchMode: target blocks for conditional jumps
	TrueBlock  *BasicBlock
	FalseBlock *BasicBlock

	// For future: target VR for assignments (Phase 2)
	// TargetVR *VirtualRegister
}

// NewValueContext creates a context for value-producing expressions
// func NewValueContext() *ExprContext {
// 	return &ExprContext{
// 		Mode: ValueMode,
// 	}
// }

// NewExprContextBranch creates a context for conditional branch expressions
func NewExprContextBranch(trueBlock, falseBlock *BasicBlock) *ExprContext {
	return &ExprContext{
		Mode:       BranchMode,
		TrueBlock:  trueBlock,
		FalseBlock: falseBlock,
	}
}

// ============================================================================
// Instruction Categories
// ============================================================================

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

// InstructionSelector converts IR to target-specific machine instructions
// This interface defines low-level operations that must be implemented per target
type InstructionSelector interface {
	// ============================================================================
	// Arithmetic Operations
	// ============================================================================

	// SelectAdd generates instructions for addition (a + b)
	SelectAdd(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectSubtract generates instructions for subtraction (a - b)
	SelectSubtract(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectMultiply generates instructions for multiplication (a * b)
	SelectMultiply(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectDivide generates instructions for division (a / b)
	SelectDivide(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectNegate generates instructions for negation (-a)
	SelectNegate(operand *VirtualRegister) (*VirtualRegister, error)
	// ============================================================================
	// Bitwise Operations
	// ============================================================================

	// SelectBitwiseAnd generates instructions for bitwise AND (a & b)
	SelectBitwiseAnd(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectBitwiseOr generates instructions for bitwise OR (a | b)
	SelectBitwiseOr(left, right *VirtualRegister) (*VirtualRegister, error)
	// SelectBitwiseXor generates instructions for bitwise XOR (a ^ b)
	SelectBitwiseXor(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectBitwiseNot generates instructions for bitwise NOT (~a)
	SelectBitwiseNot(operand *VirtualRegister) (*VirtualRegister, error)

	// SelectShiftLeft generates instructions for left shift (a << b)
	SelectShiftLeft(value, amount *VirtualRegister) (*VirtualRegister, error)

	// SelectShiftRight generates instructions for right shift (a >> b)
	SelectShiftRight(value, amount *VirtualRegister) (*VirtualRegister, error)

	// SelectLogicalAnd generates instructions for logical AND (a && b)
	// ctx: evaluation context (enables short-circuit evaluation in BranchMode)
	// left, right: the operand expressions (not yet evaluated)
	// evaluateExpr: callback to evaluate sub-expressions with context
	SelectLogicalAnd(ctx *ExprContext, left, right zsm.SemExpression, evaluateExpr func(*ExprContext, zsm.SemExpression) (*VirtualRegister, error)) (*VirtualRegister, error)

	// SelectLogicalOr generates instructions for logical OR (a || b)
	// ctx: evaluation context (enables short-circuit evaluation in BranchMode)
	// left, right: the operand expressions (not yet evaluated)
	// evaluateExpr: callback to evaluate sub-expressions with context
	SelectLogicalOr(ctx *ExprContext, left, right zsm.SemExpression, evaluateExpr func(*ExprContext, zsm.SemExpression) (*VirtualRegister, error)) (*VirtualRegister, error)

	// SelectLogicalNot generates instructions for logical NOT (!a)
	// ctx: evaluation context (inverts branch targets in BranchMode)
	// operand: the expression to negate (not yet evaluated)
	// evaluateExpr: callback to evaluate sub-expressions with context
	SelectLogicalNot(ctx *ExprContext, operand zsm.SemExpression, evaluateExpr func(*ExprContext, zsm.SemExpression) (*VirtualRegister, error)) (*VirtualRegister, error)

	// ============================================================================
	// Comparison Operations
	// ============================================================================

	// SelectEqual generates instructions for equality comparison (a == b)
	// ctx: evaluation context (BranchMode or ValueMode)
	// Returns a virtual register containing boolean result (0 or 1) in ValueMode
	// Returns nil in BranchMode (emits conditional branch instead)
	SelectEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectNotEqual generates instructions for inequality comparison (a != b)
	SelectNotEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectLessThan generates instructions for less-than comparison (a < b)
	SelectLessThan(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectLessEqual generates instructions for less-or-equal comparison (a <= b)
	SelectLessEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error)
	// SelectGreaterThan generates instructions for greater-than comparison (a > b)
	SelectGreaterThan(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectGreaterEqual generates instructions for greater-or-equal comparison (a >= b)
	SelectGreaterEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error)

	// ============================================================================
	// Memory Operations
	// ============================================================================

	// SelectLoad generates instructions to load from memory
	// address is the base address, offset is optional byte offset
	SelectLoad(address *VirtualRegister, offset int, size RegisterSize) (*VirtualRegister, error)

	// SelectLoadIndexed generates instructions to load from memory with a dynamic index
	// address is the base address, index is the index register, elementSize is bytes per element
	SelectLoadIndexed(address *VirtualRegister, index *VirtualRegister, elementSize int, size RegisterSize) (*VirtualRegister, error)

	// SelectStore generates instructions to store to memory
	SelectStore(address *VirtualRegister, value *VirtualRegister, offset int, size RegisterSize) error

	// SelectLoadConstant generates instructions to load an immediate value
	SelectLoadConstant(value interface{}, size RegisterSize) (*VirtualRegister, error)

	// SelectLoadVariable generates instructions to load a variable's value
	SelectLoadVariable(symbol *zsm.Symbol) (*VirtualRegister, error)

	// SelectStoreVariable generates instructions to store to a variable
	SelectStoreVariable(symbol *zsm.Symbol, value *VirtualRegister) error

	// Move register value -of size- from source to target
	SelectMove(target *VirtualRegister, source *VirtualRegister, size RegisterSize) error
	// ============================================================================
	// Control Flow
	// ============================================================================

	// SelectJump generates an unconditional jump to a basic block
	SelectJump(target *BasicBlock) error

	// SelectCall generates a function call
	// returnSize is the size of the return value in bits (0 for void functions)
	// Returns the virtual register containing the return value (nil if void)
	SelectCall(functionName string, args []*VirtualRegister, returnSize RegisterSize) (*VirtualRegister, error)

	// SelectReturn generates a return statement
	// value is nil for void functions
	SelectReturn(value *VirtualRegister) error

	// ============================================================================
	// Function Management
	// ============================================================================

	// SelectFunctionPrologue generates function entry code (stack frame setup)
	SelectFunctionPrologue(fn *zsm.SemFunctionDecl) error

	// SelectFunctionEpilogue generates function exit code (stack frame teardown)
	SelectFunctionEpilogue(fn *zsm.SemFunctionDecl) error

	// ============================================================================
	// Utility
	// ============================================================================

	// SetCurrentBlock sets the active block for instruction emission
	SetCurrentBlock(block *BasicBlock)

	// GetCallingConvention returns the calling convention used by this selector
	GetCallingConvention() CallingConvention

	// GetTargetRegisters returns the set of physical registers available on the target
	GetTargetRegisters() []*Register
}

type InstructionCost struct {
	Cycles uint8 // Estimated execution cycles
	Size   uint8 // Instruction size in bytes
}

// MachineInstruction represents a single target-specific instruction
// This interface exposes only what optimizers and register allocators need
type MachineInstruction interface {
	// GetResult returns the virtual register that receives the result (if any)
	GetResult() *VirtualRegister

	// GetOperands returns the virtual registers used as inputs
	GetOperands() []*VirtualRegister

	// SetResult updates the result virtual register (used during register allocation)
	SetResult(vr *VirtualRegister)

	// SetOperand updates an operand virtual register (used during register allocation)
	SetOperand(index int, vr *VirtualRegister)

	// GetCategory returns the instruction category (load, arithmetic, branch, etc.)
	GetCategory() InstrCategory

	// GetAddressingMode returns instruction addressing mode flags
	GetAddressingMode() AddressingMode

	// GetTargetBlocks returns the basic block targets for control flow instructions
	// Returns nil if this instruction doesn't transfer control
	// Returns 1 block for unconditional jumps/gotos
	// Returns 2 blocks for conditional branches ([0]=true target, [1]=false target)
	//	- if [0] or [1] is nil, falls through to next instruction
	//  - CONSTRAINT: The last branch in a block must have non-nil targets
	//                All non-terminal branches must have nil targets
	// Returns n blocks for multi-way branches (select/case/else - in order, else always last)
	GetTargetBlocks() []*BasicBlock

	// returns the cost metrics for this instruction
	GetCost() InstructionCost

	// String returns a human-readable representation of the instruction
	String() string
}

type RegisterSize uint8

const (
	Bits8  RegisterSize = 8
	Bits16 RegisterSize = 16
)
