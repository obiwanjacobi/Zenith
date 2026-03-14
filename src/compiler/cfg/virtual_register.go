package cfg

import "fmt"

// VROperand is the common interface for all virtual register kinds.
// Implemented by TempVR, ImmVR, StackVR, and PhysVR.
type VROperand interface {
	Size() uint8
	String() string
}

// ============================================================================
// TempVR — unallocated virtual register
// ============================================================================

// TempVR is an unallocated virtual register produced by each TAC expression.
// allowedSet constrains which physical register the allocator may assign.
// It is always non-empty: unconstrained VRs use a full register-class slice
// (e.g. Z80Registers8 or Z80Registers16) set at allocation time.
//
// All fields are private; use the getters to read them. To narrow the
// constraint at instruction selection time, call ConstrainTo — the explicit
// escape hatch that makes deliberate mutations visible at the call site.
type TempVR struct {
	id         int
	name       string      // optional, for debugging (e.g. source variable name)
	allowedSet []*Register // architectural constraint; always non-empty
	size       uint8       // width in bits (8 or 16)
}

func (v *TempVR) ID() int                { return v.id }
func (v *TempVR) Name() string           { return v.name }
func (v *TempVR) AllowedSet() []*Register { return v.allowedSet }
func (v *TempVR) Size() uint8            { return v.size }
func (v *TempVR) String() string {
	if v.name != "" {
		return fmt.Sprintf("'%s' t%d", v.name, v.id)
	}
	return fmt.Sprintf("t%d", v.id)
}

// ConstrainTo narrows the AllowedSet of this TempVR to regs. This is the
// deliberate escape hatch for instruction selection and register allocation,
// where constraining a pre-existing VR is the correct approach.
func (v *TempVR) ConstrainTo(regs []*Register) { v.allowedSet = regs }

// ============================================================================
// ImmVR — compile-time constant
// ============================================================================

// ImmVR represents a compile-time constant value. Never register-allocated.
type ImmVR struct {
	Value int32
	size  uint8
}

func (v *ImmVR) Size() uint8    { return v.size }
func (v *ImmVR) String() string { return fmt.Sprintf("#%d", v.Value) }

// NewImmVR creates a compile-time constant operand with the given bit width.
func NewImmVR(value int32, bits uint8) *ImmVR {
	return &ImmVR{Value: value, size: bits}
}

// ============================================================================
// StackVR — stack-allocated slot
// ============================================================================

// StackVR represents a value permanently backed by a named stack frame slot.
// Used for stack-allocated arrays, structs, and stack-homed parameters.
// Offset is relative to SP at function entry.
type StackVR struct {
	Name   string
	Offset uint16
	size   uint8
}

func (v *StackVR) Size() uint8    { return v.size }
func (v *StackVR) String() string { return fmt.Sprintf("[SP+%d]", v.Offset) }

// NewStackVR creates a StackVR for a spilled virtual register.
func NewStackVR(name string, offset uint16, size uint8) *StackVR {
	return &StackVR{Name: name, Offset: offset, size: size}
}

// ============================================================================
// PhysVR — post-allocation physical register
// ============================================================================

// PhysVR is produced by the register allocator when a TempVR is assigned
// to a concrete physical register. All TempVRs in machine instructions are
// replaced by PhysVRs before assembly emission.
type PhysVR struct {
	ID  int // original TempVR ID, for traceability
	Reg *Register
}

func (v *PhysVR) Size() uint8    { return uint8(v.Reg.Size) }
func (v *PhysVR) String() string { return v.Reg.Name }

// ============================================================================
// TempVRAllocator
// ============================================================================

// TempVRAllocator creates sequentially-numbered TempVRs for a single function.
// Use one allocator per function; do not share across functions.
type TempVRAllocator struct {
	nextID int
}

// Alloc creates a new TempVR with the given size and register constraint.
func (a *TempVRAllocator) Alloc(size uint8, allowed []*Register) *TempVR {
	vr := &TempVR{id: a.nextID, size: size, allowedSet: allowed}
	a.nextID++
	return vr
}

// AllocNamed creates a named TempVR, used for source-level variables.
func (a *TempVRAllocator) AllocNamed(name string, size uint8, allowed []*Register) *TempVR {
	vr := a.Alloc(size, allowed)
	vr.name = name
	return vr
}

// NextAnonName returns a unique anonymous name for compiler-generated temporaries.
func (a *TempVRAllocator) NextAnonName() string {
	return fmt.Sprintf("$anon%d", a.nextID)
}
