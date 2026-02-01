package cfg

import "zenith/compiler/zsm"

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
	SelectAdd(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectSubtract generates instructions for subtraction (a - b)
	SelectSubtract(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectMultiply generates instructions for multiplication (a * b)
	SelectMultiply(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectDivide generates instructions for division (a / b)
	SelectDivide(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectNegate generates instructions for negation (-a)
	SelectNegate(operand *VirtualRegister, size RegisterSize) (*VirtualRegister, error)
	// ============================================================================
	// Bitwise Operations
	// ============================================================================

	// SelectBitwiseAnd generates instructions for bitwise AND (a & b)
	SelectBitwiseAnd(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectBitwiseOr generates instructions for bitwise OR (a | b)
	SelectBitwiseOr(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)
	// SelectBitwiseXor generates instructions for bitwise XOR (a ^ b)
	SelectBitwiseXor(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectBitwiseNot generates instructions for bitwise NOT (~a)
	SelectBitwiseNot(operand *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectShiftLeft generates instructions for left shift (a << b)
	SelectShiftLeft(value, amount *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectShiftRight generates instructions for right shift (a >> b)
	SelectShiftRight(value, amount *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectLogicalAnd generates instructions for logical AND (a && b)
	// Includes short-circuit evaluation
	SelectLogicalAnd(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectLogicalOr generates instructions for logical OR (a || b)
	// Includes short-circuit evaluation
	SelectLogicalOr(left, right *VirtualRegister) (*VirtualRegister, error)

	// SelectLogicalNot generates instructions for logical NOT (!a)
	SelectLogicalNot(operand *VirtualRegister) (*VirtualRegister, error)

	// ============================================================================
	// Comparison Operations
	// ============================================================================

	// SelectEqual generates instructions for equality comparison (a == b)
	// Returns a virtual register containing boolean result (0 or 1)
	SelectEqual(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectNotEqual generates instructions for inequality comparison (a != b)
	SelectNotEqual(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectLessThan generates instructions for less-than comparison (a < b)
	SelectLessThan(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectLessEqual generates instructions for less-or-equal comparison (a <= b)
	SelectLessEqual(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)
	// SelectGreaterThan generates instructions for greater-than comparison (a > b)
	SelectGreaterThan(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// SelectGreaterEqual generates instructions for greater-or-equal comparison (a >= b)
	SelectGreaterEqual(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error)

	// ============================================================================
	// Memory Operations
	// ============================================================================

	// SelectLoad generates instructions to load from memory
	// address is the base address, offset is optional byte offset
	SelectLoad(address *VirtualRegister, offset int, size RegisterSize) (*VirtualRegister, error)

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

	// SelectBranch evaluates a conditional expression and branches
	// Handles comparison operations and logical operators (&& ||) with short-circuit evaluation
	// evaluateExpr: callback to evaluate sub-expressions to VirtualRegisters
	// expr: the boolean/comparison expression to evaluate
	// trueBlock: jumped to if expression is true
	// falseBlock: jumped to if expression is false
	SelectBranch(evaluateExpr func(zsm.SemExpression) (*VirtualRegister, error), expr zsm.SemExpression, trueBlock, falseBlock *BasicBlock) error

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
	// Returns n blocks for multi-way branches (select/case/else - in order, else always last)
	GetTargetBlocks() []*BasicBlock

	// returns the cost metrics for this instruction
	GetCost() InstructionCost
}

type RegisterSize uint8

const (
	Bits8  RegisterSize = 8
	Bits16 RegisterSize = 16
)

type VirtualRegisterType uint8

const (
	CandidateRegister VirtualRegisterType = iota // General-purpose virtual register
	StackLocation                                // Stack location (for parameters/locals)
	ImmediateValue                               // Immediate/literal value
	AllocatedRegister                            // Physical register assigned after allocation
)

// VirtualRegister represents a register before physical allocation
type VirtualRegister struct {
	// ID uniquely identifies this virtual register
	ID int

	// Size in bits - determines which physical registers are compatible
	Size RegisterSize

	// Type of virtual register
	Type VirtualRegisterType

	// AllowedSet restricts allocation to specific registers (e.g., [A] for Z80 ADD result)
	// If nil or empty, any register of the correct size and class can be used
	AllowedSet []*Register

	// PhysicalReg is set after register allocation
	PhysicalReg *Register

	// Name for debugging (optional, e.g., variable name)
	Name string

	// Value holds the value when Type is not CandidateRegister or AllocatedRegister
	Value uint32
}

// VirtualRegisterAllocator manages virtual register creation
type VirtualRegisterAllocator struct {
	nextID   int
	virtRegs map[int]*VirtualRegister
}

// NewVirtualRegisterAllocator creates a new allocator
func NewVirtualRegisterAllocator() *VirtualRegisterAllocator {
	return &VirtualRegisterAllocator{
		nextID:   0,
		virtRegs: make(map[int]*VirtualRegister),
	}
}

// Allocate creates a new virtual register
func (vra *VirtualRegisterAllocator) Allocate(size RegisterSize) *VirtualRegister {
	vr := &VirtualRegister{
		ID:   vra.nextID,
		Size: size,
	}
	vra.virtRegs[vra.nextID] = vr
	vra.nextID++
	return vr
}

// AllocateConstrained creates a virtual register with specific constraints
func (vra *VirtualRegisterAllocator) AllocateConstrained(size RegisterSize, allowedSet []*Register) *VirtualRegister {
	vr := &VirtualRegister{
		ID:         vra.nextID,
		Size:       size,
		AllowedSet: allowedSet,
	}
	vra.virtRegs[vra.nextID] = vr
	vra.nextID++
	return vr
}

// AllocateNamed creates a named virtual register (for debugging)
func (vra *VirtualRegisterAllocator) AllocateNamed(name string, size RegisterSize) *VirtualRegister {
	vr := vra.Allocate(size)
	vr.Name = name
	return vr
}

// AllocateWithStackHome creates a virtual register backed by a stack location
// This is used for parameters and locals that have a permanent stack home
func (vra *VirtualRegisterAllocator) AllocateWithStackHome(name string, size RegisterSize, stackOffset uint8) *VirtualRegister {
	vr := &VirtualRegister{
		ID:    vra.nextID,
		Size:  size,
		Type:  StackLocation,
		Name:  name,
		Value: uint32(stackOffset),
	}
	vra.virtRegs[vra.nextID] = vr
	vra.nextID++
	return vr
}

// AllocateImmediate creates a virtual register representing a constant immediate value
// This is used for constant values that don't need physical register allocation
func (vra *VirtualRegisterAllocator) AllocateImmediate(value int32, size RegisterSize) *VirtualRegister {
	vr := &VirtualRegister{
		ID:    vra.nextID,
		Size:  size,
		Type:  ImmediateValue,
		Value: uint32(value),
	}
	vra.virtRegs[vra.nextID] = vr
	vra.nextID++
	return vr
}

// GetAll returns all allocated virtual registers
func (vra *VirtualRegisterAllocator) GetAll() []*VirtualRegister {
	result := make([]*VirtualRegister, 0, len(vra.virtRegs))
	for _, vr := range vra.virtRegs {
		result = append(result, vr)
	}
	return result
}
