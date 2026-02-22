package cfg

import "fmt"

type VirtualRegisterType uint8

const (
	Unused            VirtualRegisterType = iota // Unused VR
	CandidateRegister                            // General-purpose virtual register
	StackLocation                                // Stack location (for parameters/locals)
	StackAddress                                 // Stack address (SP + offset) that needs recomputation
	ImmediateValue                               // Immediate/literal value
	AllocatedRegister                            // Physical register assigned after allocation
)

// VirtualRegister represents a register before physical allocation
type VirtualRegister struct {
	// ID uniquely identifies this virtual register
	ID int

	// Size in bits - determines which physical registers are compatible
	Size uint8

	// Type of virtual register
	Type VirtualRegisterType

	// AllowedSet restricts allocation to specific registers (e.g., [A] for Z80 ADD result)
	// If nil or empty, any register of the correct size and class can be used
	AllowedSet []*Register

	// PhysicalReg is set after register allocation
	PhysicalReg *Register

	// Name for debugging (optional, e.g., variable name)
	Name string

	// Value has different meanings based on Type:
	// - ImmediateValue: the immediate/literal value
	// - StackLocation: the stack offset for stack-based variables
	// - StackAddress: the stack offset (SP + Value) to recompute
	Value int32

	// ParentVR links component VRs (e.g., E, D) to their parent 16-bit VR (e.g., DE)
	// This allows the allocator to coordinate allocation of related VRs
	ParentVR *VirtualRegister

	// ComponentVRs links a parent VR to its component VRs [low, high]
	// e.g., DE parent would have [VR_for_E, VR_for_D]
	ComponentVRs []*VirtualRegister

	// ComponentIndex indicates which component this VR represents: 0=low byte, 1=high byte
	// Only used when ParentVR is set
	ComponentIndex int
}

func (vr *VirtualRegister) Unused() {
	if vr.PhysicalReg == nil {
		vr.Type = Unused
	}
}

func (vr *VirtualRegister) Assign(register *Register) {
	vr.PhysicalReg = register
	vr.Type = AllocatedRegister
}

func (vr *VirtualRegister) IsRegister(register *Register) bool {
	switch vr.Type {
	case AllocatedRegister:
		return vr.PhysicalReg == register
	case CandidateRegister:
		return len(vr.AllowedSet) == 1 && vr.AllowedSet[0] == register
	default:
		return false
	}
}

func (vr *VirtualRegister) HasRegister(register *Register) bool {
	if vr.Type != CandidateRegister && vr.Type != AllocatedRegister {
		return false
	}

	for _, allowed := range vr.AllowedSet {
		if register == allowed {
			return true
		}
	}

	return false
}

func (vr *VirtualRegister) MatchRegisters(registers []*Register) bool {
	if vr.Type != CandidateRegister && vr.Type != AllocatedRegister {
		return false
	}

	for _, reg := range registers {
		for _, allowed := range vr.AllowedSet {
			if reg != allowed {
				return false
			}
		}
	}

	return true
}

func (vr *VirtualRegister) MatchAnyRegisters(registers []*Register) bool {
	if vr.Type != CandidateRegister && vr.Type != AllocatedRegister {
		return false
	}

	for _, reg := range registers {
		for _, allowed := range vr.AllowedSet {
			if reg == allowed {
				return true
			}
		}
	}

	return false
}

func (vr *VirtualRegister) String() string {
	name := vr.Name
	if name == "" {
		name = fmt.Sprintf("VR%d", vr.ID)
	} else {
		name = fmt.Sprintf("'%s' VR%d", name, vr.ID)
	}
	candidates := ""
	for i, reg := range vr.AllowedSet {
		if i > 0 {
			candidates += "|"
		}
		candidates += reg.Name
	}

	switch vr.Type {
	case AllocatedRegister:
		return fmt.Sprintf("%s = %s {%s}", name, vr.PhysicalReg.Name, candidates)
	case CandidateRegister:
		return fmt.Sprintf("%s = {%s}", name, candidates)
	case ImmediateValue:
		return fmt.Sprintf("%s = #%d", name, vr.Value)
	case StackLocation:
		return fmt.Sprintf("%s = [SP+%d]", name, vr.Value)
	case StackAddress:
		return fmt.Sprintf("%s = SP+%d {%s}", name, vr.Value, candidates)
	}

	return name
}

// ============================================================================
// Virtual Register Allocator
// ============================================================================

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

// AllocateConstrained creates a virtual register with specific constraints
func (vra *VirtualRegisterAllocator) Allocate(allowedSet []*Register) *VirtualRegister {
	// TODO: check all allowed registers have the same size
	size := uint8(allowedSet[0].Size)
	vr := &VirtualRegister{
		ID:         vra.nextID,
		Size:       size,
		Type:       CandidateRegister,
		AllowedSet: allowedSet,
	}
	vra.virtRegs[vra.nextID] = vr
	vra.nextID++
	return vr
}

// AllocateNamed creates a named virtual register (for debugging)
func (vra *VirtualRegisterAllocator) AllocateNamed(name string, allowedSet []*Register) *VirtualRegister {
	vr := vra.Allocate(allowedSet)
	vr.Name = name
	return vr
}

// AllocateOnStack creates a virtual register backed by a stack location
// This is used for parameters and locals that have a permanent stack home
func (vra *VirtualRegisterAllocator) AllocateOnStack(name string, size uint8, stackOffset uint8) *VirtualRegister {
	vr := &VirtualRegister{
		ID:    vra.nextID,
		Size:  size,
		Type:  StackLocation,
		Name:  name,
		Value: int32(stackOffset),
	}
	vra.virtRegs[vra.nextID] = vr
	vra.nextID++
	return vr
}

// AllocateImmediate creates a virtual register representing a constant immediate value
// This is used for constant values that don't need physical register allocation
func (vra *VirtualRegisterAllocator) AllocateImmediate(value int32, size uint8) *VirtualRegister {
	vr := &VirtualRegister{
		ID:    vra.nextID,
		Size:  size,
		Type:  ImmediateValue,
		Value: value,
	}
	vra.virtRegs[vra.nextID] = vr
	vra.nextID++
	return vr
}

// AllocateComponents creates component VRs (lo, hi) linked to a parent 16-bit VR
// This maintains the relationship so the register allocator can coordinate their allocation
// Returns (lowVR, highVR)
func (vra *VirtualRegisterAllocator) AllocateComponents(parentVR *VirtualRegister) (loVR, hiVR *VirtualRegister) {
	if parentVR.Size != 16 {
		panic("AllocateComponents requires a 16-bit parent VR")
	}

	// Derive component registers from parent's AllowedSet
	loRegs, hiRegs := ToPairs(parentVR.AllowedSet)

	// Create low byte component
	loVR = &VirtualRegister{
		ID:             vra.nextID,
		Size:           8,
		Type:           CandidateRegister,
		AllowedSet:     loRegs,
		ParentVR:       parentVR,
		ComponentIndex: 0,
	}
	vra.virtRegs[vra.nextID] = loVR
	vra.nextID++

	// Create high byte component
	hiVR = &VirtualRegister{
		ID:             vra.nextID,
		Size:           8,
		Type:           CandidateRegister,
		AllowedSet:     hiRegs,
		ParentVR:       parentVR,
		ComponentIndex: 1,
	}
	vra.virtRegs[vra.nextID] = hiVR
	vra.nextID++

	// Link components to parent
	parentVR.ComponentVRs = []*VirtualRegister{loVR, hiVR}

	return loVR, hiVR
}

// AllocateStackAddress creates a VR marker representing a stack address (SP + offset)
// This VR must be materialized via SelectLoadStackAddress before use.
func (vra *VirtualRegisterAllocator) AllocateStackAddress(stackOffset uint16) *VirtualRegister {
	vr := &VirtualRegister{
		ID:    vra.nextID,
		Size:  16,
		Type:  StackAddress,
		Value: int32(stackOffset),
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

func DumpAllocation(vrAlloc *VirtualRegisterAllocator) {
	fmt.Println("========== REGISTER ALLOCATION ==========")

	// Collect VRs by type
	unused := []*VirtualRegister{}
	allocated := []*VirtualRegister{}
	spilled := []*VirtualRegister{}
	immediates := []*VirtualRegister{}
	candidates := []*VirtualRegister{}

	for _, vr := range vrAlloc.GetAll() {
		switch vr.Type {
		case Unused:
			unused = append(unused, vr)
		case AllocatedRegister:
			allocated = append(allocated, vr)
		case StackLocation:
			spilled = append(spilled, vr)
		case ImmediateValue:
			immediates = append(immediates, vr)
		case CandidateRegister:
			candidates = append(candidates, vr)
		}
	}

	if len(allocated) > 0 {
		fmt.Printf("Allocated (%d):\n", len(allocated))
		for _, vr := range allocated {
			fmt.Println(vr.String())
		}
	}

	if len(spilled) > 0 {
		fmt.Printf("\nSpilled to stack (%d):\n", len(spilled))
		for _, vr := range spilled {
			fmt.Println(vr.String())
		}
	}

	if len(immediates) > 0 {
		fmt.Printf("\nImmediates (%d):\n", len(immediates))
		for _, vr := range immediates {
			fmt.Println(vr.String())
		}
	}

	if len(candidates) > 0 {
		fmt.Printf("\nUnallocated candidates (%d):\n", len(candidates))
		for _, vr := range candidates {
			fmt.Println(vr.String())
		}
	}

	if len(unused) > 0 {
		fmt.Printf("\nUnused (%d):\n", len(unused))
		for _, vr := range unused {
			fmt.Println(vr.String())
		}
	}

	fmt.Printf("\nTotal: %d VRs (%d allocated, %d spilled, %d immediate, %d unallocated, unused %d)\n\n",
		len(vrAlloc.GetAll()), len(allocated), len(spilled), len(immediates), len(candidates), len(unused))
}
