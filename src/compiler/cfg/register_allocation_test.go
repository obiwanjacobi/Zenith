package cfg

import (
	"testing"
)

// Test simple register allocation with no interference
func TestRegisterAllocation_NoInterference(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	// Two VRs that don't interfere (sequential use)
	vr1 := vrAlloc.AllocateNamed("temp1", Z80Registers8)
	vr2 := vrAlloc.AllocateNamed("temp2", Z80Registers8)

	// Block: temp1 = load 5, temp2 = temp1 + 10
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode: Z80_LD_R_N,
				result: vr1,
			},
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr2,
				operands: []*VirtualRegister{vr1},
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// Both should be allocated (they can share the same register)
	if vr1.Type != AllocatedRegister {
		t.Errorf("vr1 should be allocated, got %v", vr1.Type)
	}
	if vr2.Type != AllocatedRegister {
		t.Errorf("vr2 should be allocated, got %v", vr2.Type)
	}

	// Both should have physical registers assigned
	if vr1.PhysicalReg == nil {
		t.Error("vr1 should have physical register")
	}
	if vr2.PhysicalReg == nil {
		t.Error("vr2 should have physical register")
	}

	// They could share the same register (no interference)
	// or use different registers - both are valid
}

// Test register allocation with interference (must use different registers)
func TestRegisterAllocation_WithInterference(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("x", Z80Registers8)
	vr2 := vrAlloc.AllocateNamed("y", Z80Registers8)
	vr3 := vrAlloc.AllocateNamed("z", Z80Registers8)

	// Block: z = x + y (x and y are live together)
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr3,
				operands: []*VirtualRegister{vr1, vr2},
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// All should be allocated (Z80 has enough registers for 3 VRs)
	if vr1.Type != AllocatedRegister || vr2.Type != AllocatedRegister || vr3.Type != AllocatedRegister {
		t.Errorf("All VRs should be allocated: vr1=%v, vr2=%v, vr3=%v", vr1.Type, vr2.Type, vr3.Type)
	}

	// x and y must have different registers (they interfere)
	if vr1.PhysicalReg == vr2.PhysicalReg {
		t.Error("vr1 and vr2 interfere, they must use different registers")
	}
}

// Test allocation with constrained register (AllowedSet)
func TestRegisterAllocation_ConstrainedRegister(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	// VR constrained to specific register (e.g., accumulator for Z80 ADD)
	vr1 := vrAlloc.AllocateNamed("x", Z80Registers8)
	vr2 := vrAlloc.Allocate(Z80RegA) // Must use A register
	vr2.Name = "result"

	// Block: result = x (move to accumulator)
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr2,
				operands: []*VirtualRegister{vr1},
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// vr2 must be assigned to A register
	if vr2.PhysicalReg != &RegA {
		t.Errorf("vr2 should be assigned to A register, got %v", vr2.PhysicalReg)
	}

	// vr1 should be allocated (but may or may not be A, depending on allocation order)
	if vr1.Type != AllocatedRegister {
		t.Error("vr1 should be allocated")
	}
}

// Test allocation with multiple interfering VRs
func TestRegisterAllocation_MultipleInterference(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("a", Z80Registers8)
	vr2 := vrAlloc.AllocateNamed("b", Z80Registers8)
	vr3 := vrAlloc.AllocateNamed("c", Z80Registers8)
	vr4 := vrAlloc.AllocateNamed("d", Z80Registers8)

	// All four are live at the same time
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr4,
				operands: []*VirtualRegister{vr1, vr2},
			},
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr4,
				operands: []*VirtualRegister{vr4, vr3},
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// All should be allocated (Z80 has enough registers)
	allocatedRegs := make(map[*Register]bool)
	for _, vr := range []*VirtualRegister{vr1, vr2, vr3} {
		if vr.Type != AllocatedRegister {
			t.Errorf("VR %s should be allocated, got %v", vr.Name, vr.Type)
		}
		if vr.PhysicalReg == nil {
			t.Errorf("VR %s should have physical register", vr.Name)
		} else {
			// Check for duplicates among interfering VRs
			if allocatedRegs[vr.PhysicalReg] {
				t.Errorf("VR %s shares register %s with another interfering VR", vr.Name, vr.PhysicalReg.Name)
			}
			allocatedRegs[vr.PhysicalReg] = true
		}
	}
}

// Test allocation in a loop (VRs live across iterations)
func TestRegisterAllocation_Loop(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("i", Z80Registers8)
	vr2 := vrAlloc.AllocateNamed("sum", Z80Registers8)
	vr3 := vrAlloc.AllocateNamed("n", Z80Registers8)

	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode: Z80_LD_R_N,
				result: vr2,
			},
			&machineInstructionZ80{
				opcode: Z80_LD_R_N,
				result: vr1,
			},
		},
	}

	block1 := &BasicBlock{
		ID: 1,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_CP_R,
				operands: []*VirtualRegister{vr1, vr3},
			},
		},
	}

	block2 := &BasicBlock{
		ID: 2,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr2,
				operands: []*VirtualRegister{vr2, vr1},
			},
			&machineInstructionZ80{
				opcode:   Z80_INC_R,
				result:   vr1,
				operands: []*VirtualRegister{vr1},
			},
		},
	}

	block3 := &BasicBlock{
		ID: 3,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_RET,
				operands: []*VirtualRegister{vr2},
			},
		},
	}

	block0.Successors = []*BasicBlock{block1}
	block1.Successors = []*BasicBlock{block2, block3}
	block2.Successors = []*BasicBlock{block1}
	block3.Successors = []*BasicBlock{}

	cfg := &CFG{
		FunctionName: "test_loop",
		Blocks:       []*BasicBlock{block0, block1, block2, block3},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// All three should be allocated (Z80 has enough registers)
	if vr1.Type != AllocatedRegister || vr2.Type != AllocatedRegister || vr3.Type != AllocatedRegister {
		t.Errorf("All loop VRs should be allocated: vr1=%v, vr2=%v, vr3=%v", vr1.Type, vr2.Type, vr3.Type)
	}

	// All three interfere, so they must use different registers
	if vr1.PhysicalReg == vr2.PhysicalReg {
		t.Error("i and sum must use different registers")
	}
	if vr1.PhysicalReg == vr3.PhysicalReg {
		t.Error("i and n must use different registers")
	}
	if vr2.PhysicalReg == vr3.PhysicalReg {
		t.Error("sum and n must use different registers")
	}
}

// Test 16-bit register allocation
func TestRegisterAllocation_16Bit(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("x", Z80Registers16)
	vr2 := vrAlloc.AllocateNamed("y", Z80Registers16)
	vr3 := vrAlloc.AllocateNamed("z", Z80Registers16)

	// Block: z = x + y (both x and y are live together, then z)
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_HL_RR,
				result:   vr3,
				operands: []*VirtualRegister{vr1, vr2},
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test_16bit",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// All should be allocated to 16-bit registers
	if vr1.PhysicalReg == nil || vr1.PhysicalReg.Size != 16 {
		t.Error("vr1 should be allocated to 16-bit register")
	}
	if vr2.PhysicalReg == nil || vr2.PhysicalReg.Size != 16 {
		t.Error("vr2 should be allocated to 16-bit register")
	}
	if vr3.PhysicalReg == nil || vr3.PhysicalReg.Size != 16 {
		t.Error("vr3 should be allocated to 16-bit register")
	}

	// x and y should use different 16-bit registers (they interfere - both live at same time)
	if vr1.PhysicalReg == vr2.PhysicalReg {
		t.Error("Interfering 16-bit VRs (x and y) must use different registers")
	}
}

// Test that stack locations and immediates are not allocated
func TestRegisterAllocation_SkipsNonCandidates(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("x", Z80Registers8)
	vrStack := vrAlloc.AllocateWithStackHome("param", Bits8, 4)
	vrImm := vrAlloc.AllocateImmediate(42, Bits8)

	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr1,
				operands: []*VirtualRegister{vrStack, vrImm},
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// vr1 should be allocated
	if vr1.Type != AllocatedRegister {
		t.Error("vr1 should be allocated")
	}

	// Stack and immediate should remain unchanged
	if vrStack.Type != StackLocation {
		t.Error("Stack VR should remain StackLocation")
	}
	if vrStack.PhysicalReg != nil {
		t.Error("Stack VR should not have PhysicalReg assigned")
	}

	if vrImm.Type != ImmediateValue {
		t.Error("Immediate VR should remain ImmediateValue")
	}
	if vrImm.PhysicalReg != nil {
		t.Error("Immediate VR should not have PhysicalReg assigned")
	}
}

// Test allocation with branching
func TestRegisterAllocation_Branching(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("a", Z80Registers8)
	vr2 := vrAlloc.AllocateNamed("b", Z80Registers8)
	vr3 := vrAlloc.AllocateNamed("c", Z80Registers8)
	vr4 := vrAlloc.AllocateNamed("result", Z80Registers8)

	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_CP_R,
				operands: []*VirtualRegister{vr1, vr2},
			},
		},
	}

	block1 := &BasicBlock{
		ID: 1,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr4,
				operands: []*VirtualRegister{vr1, vr3},
			},
		},
	}

	block2 := &BasicBlock{
		ID: 2,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr4,
				operands: []*VirtualRegister{vr2, vr3},
			},
		},
	}

	block3 := &BasicBlock{
		ID: 3,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_RET,
				operands: []*VirtualRegister{vr4},
			},
		},
	}

	block0.Successors = []*BasicBlock{block1, block2}
	block1.Successors = []*BasicBlock{block3}
	block2.Successors = []*BasicBlock{block3}
	block3.Successors = []*BasicBlock{}

	cfg := &CFG{
		FunctionName: "test_branch",
		Blocks:       []*BasicBlock{block0, block1, block2, block3},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	// All should be allocated
	for _, vr := range []*VirtualRegister{vr1, vr2, vr3, vr4} {
		if vr.Type != AllocatedRegister {
			t.Errorf("VR %s should be allocated", vr.Name)
		}
	}

	// a, b, c all interfere (live at block 0 exit), must use different registers
	if vr1.PhysicalReg == vr2.PhysicalReg || vr1.PhysicalReg == vr3.PhysicalReg || vr2.PhysicalReg == vr3.PhysicalReg {
		t.Error("a, b, c all interfere and must use different registers")
	}
}

// Test empty CFG (no VRs to allocate)
func TestRegisterAllocation_EmptyCFG(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	block0 := &BasicBlock{
		ID:                  0,
		MachineInstructions: []MachineInstruction{},
		Successors:          []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test_empty",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	allocator := NewRegisterAllocator(Z80Registers)
	err := allocator.Allocate(cfg, ig, vrAlloc.GetAll())

	if err != nil {
		t.Fatalf("Allocation of empty CFG should not fail: %v", err)
	}
}
