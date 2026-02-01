package cfg

import (
	"testing"
)

// Test basic liveness analysis with a simple linear flow
func TestLiveness_SimpleLinearFlow(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	// Create virtual registers
	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("y", Z80RegistersR)
	vr3 := vrAlloc.AllocateNamed("z", Z80RegistersR)

	// Block 0: z = x + y
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			newInstructionZ80(Z80_ADD_A_R, vr3, vr1), // z = x + y (uses vr1, vr2, defines vr3)
		},
		Successors: []*BasicBlock{},
	}

	// Manually set operands for the instruction (simulating what instruction selection would do)
	block0.MachineInstructions[0].SetOperand(0, vr1)
	block0.MachineInstructions[0].(*machineInstructionZ80).operands = append(
		block0.MachineInstructions[0].(*machineInstructionZ80).operands, vr2)

	// Create CFG
	cfg := &CFG{
		FunctionName: "test",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	// Compute liveness
	liveness := ComputeLiveness(cfg)

	// Verify Use and Def sets for block 0
	// Use[0] should contain vr1 and vr2 (used before defined)
	if !liveness.Use[0][vr1.ID] {
		t.Errorf("Expected vr1 (ID=%d) in Use[0]", vr1.ID)
	}
	if !liveness.Use[0][vr2.ID] {
		t.Errorf("Expected vr2 (ID=%d) in Use[0]", vr2.ID)
	}

	// Def[0] should contain vr3 (defined in block)
	if !liveness.Def[0][vr3.ID] {
		t.Errorf("Expected vr3 (ID=%d) in Def[0]", vr3.ID)
	}

	// LiveIn[0] should contain vr1 and vr2 (needed as inputs)
	if !liveness.LiveIn[0][vr1.ID] {
		t.Errorf("Expected vr1 (ID=%d) in LiveIn[0]", vr1.ID)
	}
	if !liveness.LiveIn[0][vr2.ID] {
		t.Errorf("Expected vr2 (ID=%d) in LiveIn[0]", vr2.ID)
	}

	// LiveOut[0] should be empty (no successors)
	if len(liveness.LiveOut[0]) != 0 {
		t.Errorf("Expected LiveOut[0] to be empty, got %v", liveness.LiveOut[0])
	}
}

// Test liveness with branching
func TestLiveness_ConditionalBranch(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("a", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("b", Z80RegistersR)
	vr3 := vrAlloc.AllocateNamed("c", Z80RegistersR)
	vr4 := vrAlloc.AllocateNamed("result", Z80RegistersR)

	// Block 0: condition = a < b
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_CP_R,
				operands: []*VirtualRegister{vr1, vr2},
			},
		},
	}

	// Block 1: result = a + c (true branch)
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

	// Block 2: result = b + c (false branch)
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

	// Block 3: return result (merge point)
	block3 := &BasicBlock{
		ID: 3,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_RET,
				operands: []*VirtualRegister{vr4},
			},
		},
	}

	// Set up CFG structure
	block0.Successors = []*BasicBlock{block1, block2}
	block1.Successors = []*BasicBlock{block3}
	block2.Successors = []*BasicBlock{block3}
	block3.Successors = []*BasicBlock{}

	cfg := &CFG{
		FunctionName: "test_branch",
		Blocks:       []*BasicBlock{block0, block1, block2, block3},
		Entry:        block0,
	}

	// Compute liveness
	liveness := ComputeLiveness(cfg)

	// Block 0: vr1 and vr2 are used
	if !liveness.Use[0][vr1.ID] || !liveness.Use[0][vr2.ID] {
		t.Error("Block 0 should use vr1 and vr2")
	}

	// Block 0: vr1, vr2, vr3 should be live-out (needed by successors)
	if !liveness.LiveOut[0][vr1.ID] {
		t.Error("vr1 should be live-out of block 0 (needed by block 1)")
	}
	if !liveness.LiveOut[0][vr2.ID] {
		t.Error("vr2 should be live-out of block 0 (needed by block 2)")
	}
	if !liveness.LiveOut[0][vr3.ID] {
		t.Error("vr3 should be live-out of block 0 (needed by both branches)")
	}

	// Block 1: vr1 and vr3 should be live-in
	if !liveness.LiveIn[1][vr1.ID] || !liveness.LiveIn[1][vr3.ID] {
		t.Error("Block 1 should have vr1 and vr3 live-in")
	}

	// Block 2: vr2 and vr3 should be live-in
	if !liveness.LiveIn[2][vr2.ID] || !liveness.LiveIn[2][vr3.ID] {
		t.Error("Block 2 should have vr2 and vr3 live-in")
	}

	// Block 3: vr4 should be live-in (defined in predecessors, used here)
	if !liveness.LiveIn[3][vr4.ID] {
		t.Error("Block 3 should have vr4 live-in")
	}

	// Block 1 and 2 should define vr4
	if !liveness.Def[1][vr4.ID] {
		t.Error("Block 1 should define vr4")
	}
	if !liveness.Def[2][vr4.ID] {
		t.Error("Block 2 should define vr4")
	}
}

// Test liveness with a loop
func TestLiveness_Loop(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("i", Z80RegistersR)   // loop counter
	vr2 := vrAlloc.AllocateNamed("sum", Z80RegistersR) // accumulator
	vr3 := vrAlloc.AllocateNamed("n", Z80RegistersR)   // loop bound

	// Block 0: Entry - initialize sum = 0, i = 0
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

	// Block 3: Exit - return sum
	block3 := &BasicBlock{
		ID: 3,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_RET,
				operands: []*VirtualRegister{vr2},
			},
		},
	}

	// Set up CFG structure (loop: 0 -> 1 -> 2 -> 1 -> 3)
	block0.Successors = []*BasicBlock{block1}
	block1.Successors = []*BasicBlock{block2, block3} // true: continue loop, false: exit
	block2.Successors = []*BasicBlock{block1}         // back edge
	block3.Successors = []*BasicBlock{}

	cfg := &CFG{
		FunctionName: "test_loop",
		Blocks:       []*BasicBlock{block0, block1, block2, block3},
		Entry:        block0,
	}

	// Compute liveness
	liveness := ComputeLiveness(cfg)

	// Block 0 should define vr1 and vr2
	if !liveness.Def[0][vr1.ID] || !liveness.Def[0][vr2.ID] {
		t.Error("Block 0 should define vr1 and vr2")
	}

	// Block 1 (loop header): vr1 and vr3 are used, vr1 and vr2 should be live-in
	// (vr1 and vr2 are modified in the loop and flow back)
	if !liveness.Use[1][vr1.ID] || !liveness.Use[1][vr3.ID] {
		t.Error("Block 1 should use vr1 and vr3")
	}

	// Block 1: vr1, vr2, vr3 should be live-in (needed for loop)
	if !liveness.LiveIn[1][vr1.ID] {
		t.Error("vr1 should be live-in at loop header (modified in loop)")
	}
	if !liveness.LiveIn[1][vr2.ID] {
		t.Error("vr2 should be live-in at loop header (modified in loop)")
	}
	if !liveness.LiveIn[1][vr3.ID] {
		t.Error("vr3 should be live-in at loop header (used for comparison)")
	}

	// Block 2 (loop body): should use and define both vr1 and vr2
	if !liveness.Use[2][vr1.ID] || !liveness.Use[2][vr2.ID] {
		t.Error("Block 2 should use vr1 and vr2")
	}
	if !liveness.Def[2][vr1.ID] || !liveness.Def[2][vr2.ID] {
		t.Error("Block 2 should define vr1 and vr2")
	}

	// Block 3 (exit): vr2 should be live-in (used for return)
	if !liveness.LiveIn[3][vr2.ID] {
		t.Error("vr2 should be live-in at exit (return value)")
	}
}

// Test GetLiveRanges helper function
func TestLiveness_GetLiveRanges(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)
	vr2 := vrAlloc.AllocateNamed("y", Z80RegistersR)

	// Block 0: defines vr1
	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode: Z80_LD_R_N,
				result: vr1,
			},
		},
	}

	// Block 1: uses vr1, defines vr2
	block1 := &BasicBlock{
		ID: 1,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr2,
				operands: []*VirtualRegister{vr1},
			},
		},
	}

	// Block 2: uses vr2
	block2 := &BasicBlock{
		ID: 2,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_RET,
				operands: []*VirtualRegister{vr2},
			},
		},
	}

	block0.Successors = []*BasicBlock{block1}
	block1.Successors = []*BasicBlock{block2}
	block2.Successors = []*BasicBlock{}

	cfg := &CFG{
		FunctionName: "test_ranges",
		Blocks:       []*BasicBlock{block0, block1, block2},
		Entry:        block0,
	}

	// Compute liveness
	liveness := ComputeLiveness(cfg)

	// Get live ranges
	ranges := liveness.GetLiveRanges()

	// vr1 should be live in blocks 0 and 1
	vr1Ranges, ok := ranges[vr1.ID]
	if !ok {
		t.Fatal("vr1 should have live ranges")
	}
	if len(vr1Ranges) < 1 {
		t.Error("vr1 should be live in at least one block")
	}

	// vr2 should be live in blocks 1 and 2
	vr2Ranges, ok := ranges[vr2.ID]
	if !ok {
		t.Fatal("vr2 should have live ranges")
	}
	if len(vr2Ranges) < 1 {
		t.Error("vr2 should be live in at least one block")
	}
}

// Test IsLiveAt helper function
func TestLiveness_IsLiveAt(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()
	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)

	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode: Z80_LD_R_N,
				result: vr1,
			},
		},
		Successors: []*BasicBlock{},
	}

	cfg := &CFG{
		FunctionName: "test_is_live",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)

	// vr1 is defined in block 0, so it should be live-out if used in successor
	// Since there's no successor, it's not live-out
	if liveness.IsLiveAt(vr1.ID, 0) {
		t.Error("vr1 should not be live-in at block 0 (it's defined there)")
	}
}

// Test that immediate values are not tracked in liveness
func TestLiveness_IgnoresImmediates(t *testing.T) {
	vrAlloc := NewVirtualRegisterAllocator()

	vr1 := vrAlloc.AllocateNamed("x", Z80RegistersR)
	vrImm := vrAlloc.AllocateImmediate(42, Bits8)

	block0 := &BasicBlock{
		ID: 0,
		MachineInstructions: []MachineInstruction{
			&machineInstructionZ80{
				opcode:   Z80_ADD_A_R,
				result:   vr1,
				operands: []*VirtualRegister{vrImm},
			},
		},
	}

	cfg := &CFG{
		FunctionName: "test_immediates",
		Blocks:       []*BasicBlock{block0},
		Entry:        block0,
	}

	liveness := ComputeLiveness(cfg)

	// vrImm (immediate) should not appear in Use set
	if liveness.Use[0][vrImm.ID] {
		t.Error("Immediate values should not be tracked in Use sets")
	}

	// Only vr1 should be in Def set
	if !liveness.Def[0][vr1.ID] {
		t.Error("vr1 should be in Def set")
	}
	if liveness.Def[0][vrImm.ID] {
		t.Error("Immediate values should not be tracked in Def sets")
	}
}
