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
// AllowedSet constrains which physical register the allocator may assign.
// It is always non-empty: unconstrained VRs use a full register-class slice
// (e.g. Z80Registers8 or Z80Registers16) set at allocation time.
//
// VRs are immutable once created: the allocator produces PhysVRs from TempVRs
// rather than mutating TempVR fields.
type TempVR struct {
	ID         int
	Name       string      // optional, for debugging (e.g. source variable name)
	AllowedSet []*Register // architectural constraint; always non-empty
	size       uint8       // width in bits (8 or 16)
}

func (v *TempVR) Size() uint8 { return v.size }
func (v *TempVR) String() string {
	if v.Name != "" {
		return fmt.Sprintf("'%s' t%d", v.Name, v.ID)
	}
	return fmt.Sprintf("t%d", v.ID)
}

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
	vr := &TempVR{ID: a.nextID, size: size, AllowedSet: allowed}
	a.nextID++
	return vr
}

// AllocNamed creates a named TempVR, used for source-level variables.
func (a *TempVRAllocator) AllocNamed(name string, size uint8, allowed []*Register) *TempVR {
	vr := a.Alloc(size, allowed)
	vr.Name = name
	return vr
}

// NextAnonName returns a unique anonymous name for compiler-generated temporaries.
func (a *TempVRAllocator) NextAnonName() string {
	return fmt.Sprintf("$anon%d", a.nextID)
}
