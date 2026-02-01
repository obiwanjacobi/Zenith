package cfg

import (
	"testing"
)

// Test basic interference graph construction
func TestInterference_SimpleLinearFlow(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	// Create virtual registers
	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("y", Z80RegistersR)
	vr3 := vrAlloc.AllocateNamed("z", Z80RegistersR)

	// Block 0: z = x + y (x and y are live together, z doesn't interfere with them)
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

	// x and y are used together, so they interfere
	if !ig.Interferes(vr1.ID, vr2.ID) {
		t.Error("vr1 (x) and vr2 (y) should interfere (both live at same time)")
	}

	// z is defined after x and y are used, so it should not interfere with them
	// (they're not live when z is defined)
	if ig.Interferes(vr3.ID, vr1.ID) {
		t.Error("vr3 (z) should not interfere with vr1 (x)")
	}
	if ig.Interferes(vr3.ID, vr2.ID) {
		t.Error("vr3 (z) should not interfere with vr2 (y)")
	}

	// Check degree
	if ig.GetDegree(vr1.ID) != 1 {
		t.Errorf("vr1 should have degree 1 (interferes with vr2), got %d", ig.GetDegree(vr1.ID))
	}
	if ig.GetDegree(vr2.ID) != 1 {
		t.Errorf("vr2 should have degree 1 (interferes with vr1), got %d", ig.GetDegree(vr2.ID))
	}
}

// Test interference with live ranges that overlap
func TestInterference_OverlappingLiveRanges(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("a", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("b", Z80RegistersR)
	vr3 := vrAlloc.AllocateNamed("c", Z80RegistersR)

	// Block 0:
	//   a = load 1
	//   b = load 2
	//   c = a + b
	// After c is defined, a and b are dead
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:         Z80_LD_R_N,
				result:         vr1,
				immediateValue: 1,
			},
			&machineInstructionZ80{
				opcode:         Z80_LD_R_N,
				result:         vr2,
				immediateValue: 2,
			},
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr3,
				operands: []*VirtualRegister{vr1, vr2},
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test_overlap",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	// a and b are both live between their definitions and use in ADD
	if !ig.Interferes(vr1.ID, vr2.ID) {
		t.Error("vr1 (a) and vr2 (b) should interfere (live ranges overlap)")
	}

	// c doesn't interfere with a or b (they're dead when c is defined)
	if ig.Interferes(vr3.ID, vr1.ID) {
		t.Error("vr3 (c) should not interfere with vr1 (a)")
	}
	if ig.Interferes(vr3.ID, vr2.ID) {
		t.Error("vr3 (c) should not interfere with vr2 (b)")
	}
}

// Test interference in branching control flow
func TestInterference_Branching(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("a", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("b", Z80RegistersR)
	vr3 := vrAlloc.AllocateNamed("c", Z80RegistersR)
	vr4 := vrAlloc.AllocateNamed("result", Z80RegistersR)

	// Block 0: a, b, c are all live
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_CP_R,
				operands: []*VirtualRegister{vr1, vr2},
			},
		},
	}

	// Block 1: result = a + c
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

	// Block 2: result = b + c
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

	// Block 3: return result
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

	// a, b, c are all live at block 0 exit, so they all interfere
	if !ig.Interferes(vr1.ID, vr2.ID) {
		t.Error("a and b should interfere (both live at block 0)")
	}
	if !ig.Interferes(vr1.ID, vr3.ID) {
		t.Error("a and c should interfere (both live at block 0)")
	}
	if !ig.Interferes(vr2.ID, vr3.ID) {
		t.Error("b and c should interfere (both live at block 0)")
	}

	// result doesn't interfere with a, b, c (defined after they're used)
	if ig.Interferes(vr4.ID, vr1.ID) {
		t.Error("result should not interfere with a")
	}
	if ig.Interferes(vr4.ID, vr2.ID) {
		t.Error("result should not interfere with b")
	}
	if ig.Interferes(vr4.ID, vr3.ID) {
		t.Error("result should not interfere with c")
	}
}

// Test interference in a loop (variables live across iterations)
func TestInterference_Loop(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("i", Z80RegistersR)   // loop counter
	vr2 := vrAlloc.AllocateNamed("sum", Z80RegistersR) // accumulator
	vr3 := vrAlloc.AllocateNamed("n", Z80RegistersR)   // loop bound

	// Block 0: Entry - initialize
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

	// Block 1: Loop header - check i < n
	block1 := &BasicBlock{
		ID: 1,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_CP_R,
				operands: []*VirtualRegister{vr1, vr3},
			},
		},
	}

	// Block 2: Loop body - sum = sum + i, i = i + 1
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

	// Block 3: Exit
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

	// i, sum, and n are all live in the loop header, so they interfere
	if !ig.Interferes(vr1.ID, vr2.ID) {
		t.Error("i and sum should interfere (both live in loop)")
	}
	if !ig.Interferes(vr1.ID, vr3.ID) {
		t.Error("i and n should interfere (both live in loop)")
	}
	if !ig.Interferes(vr2.ID, vr3.ID) {
		t.Error("sum and n should interfere (both live in loop)")
	}

	// Check degrees - each should interfere with the other two
	if ig.GetDegree(vr1.ID) != 2 {
		t.Errorf("i should have degree 2, got %d", ig.GetDegree(vr1.ID))
	}
	if ig.GetDegree(vr2.ID) != 2 {
		t.Errorf("sum should have degree 2, got %d", ig.GetDegree(vr2.ID))
	}
	if ig.GetDegree(vr3.ID) != 2 {
		t.Errorf("n should have degree 2, got %d", ig.GetDegree(vr3.ID))
	}
}

// Test that immediately reused registers interfere
func TestInterference_ImmediateReuse(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("temp1", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("temp2", Z80RegistersR)

	// Block 0:
	//   temp1 = load 5
	//   temp2 = temp1 + 10
	// temp1 is dead after being used for temp2, so they could share a register
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
		FunctionName: "test_reuse",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	// temp1 and temp2 should NOT interfere (temp1 is dead when temp2 is defined)
	if ig.Interferes(vr1.ID, vr2.ID) {
		t.Error("temp1 and temp2 should not interfere (disjoint live ranges)")
	}

	// Both should be in the graph though
	nodes := ig.GetNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes in graph, got %d", len(nodes))
	}
}

// Test GetNeighbors functionality
func TestInterference_GetNeighbors(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("a", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("b", Z80RegistersR)
	vr3 := vrAlloc.AllocateNamed("c", Z80RegistersR)

	// Create a scenario where all three are live at the same time
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr1,
				operands: []*VirtualRegister{vr2, vr3}, // All three live here
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test_neighbors",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	// vr2 and vr3 should interfere with each other (both live when used)
	if !ig.Interferes(vr2.ID, vr3.ID) {
		t.Error("vr2 and vr3 should interfere (both live at same time)")
	}

	// Check neighbors of vr2
	neighbors2 := ig.GetNeighbors(vr2.ID)
	if len(neighbors2) != 1 || neighbors2[0] != vr3.ID {
		t.Errorf("vr2 should have vr3 as neighbor, got %v", neighbors2)
	}

	// Check neighbors of vr3
	neighbors3 := ig.GetNeighbors(vr3.ID)
	if len(neighbors3) != 1 || neighbors3[0] != vr2.ID {
		t.Errorf("vr3 should have vr2 as neighbor, got %v", neighbors3)
	}

	// Test symmetry: if a interferes with b, then b interferes with a
	for _, neighborID := range neighbors2 {
		if !ig.Interferes(neighborID, vr2.ID) {
			t.Errorf("Interference should be symmetric: vr2 interferes with VR%d, but not vice versa", neighborID)
		}
	}
}

// Test that the graph correctly handles self-loops (should not add them)
func TestInterference_NoSelfLoops(t *testing.T) {
	ig := NewInterferenceGraph()

	vrAlloc := NewVirtualRegisterAllocator()
	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)

	ig.AddNode(vr1.ID)
	ig.AddEdge(vr1.ID, vr1.ID) // Try to add self-loop

	// Should not interfere with itself
	if ig.Interferes(vr1.ID, vr1.ID) {
		t.Error("VR should not interfere with itself")
	}

	// Degree should be 0 (no actual edges)
	if ig.GetDegree(vr1.ID) != 0 {
		t.Errorf("Expected degree 0 for self-loop attempt, got %d", ig.GetDegree(vr1.ID))
	}
}

// Test empty interference graph
func TestInterference_EmptyGraph(t *testing.T) {
	ig := NewInterferenceGraph()

	nodes := ig.GetNodes()
	if len(nodes) != 0 {
		t.Errorf("Empty graph should have 0 nodes, got %d", len(nodes))
	}

	vrAlloc := NewVirtualRegisterAllocator()
	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("y", Z80RegistersR)

	if ig.Interferes(vr1.ID, vr2.ID) {
		t.Error("Empty graph should not report any interferences")
	}

	if ig.GetDegree(vr1.ID) != 0 {
		t.Error("Non-existent node should have degree 0")
	}
}

// Test that immediate values are not added to interference graph
func TestInterference_IgnoresImmediates(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)
	vrImm := vrAlloc.AllocateImmediate(42, Bits8)

	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_LD_R_N,
				result:   vr1,
				operands: []*VirtualRegister{vrImm},
			},
		},
	}

	cfg := &CFG{
		FunctionName: "test_imm",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)
	ig := BuildInterferenceGraph(cfg, liveness)

	// Immediate should not be in the graph
	nodes := ig.GetNodes()
	for _, nodeID := range nodes {
		if nodeID == vrImm.ID {
			t.Error("Immediate value should not be in interference graph")
		}
	}
}
