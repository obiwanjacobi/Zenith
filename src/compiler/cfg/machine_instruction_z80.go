package cfg

import (
	"fmt"
	"strings"
)

// ============================================================================
// Z80-specific instruction representation
// ============================================================================

// machineInstructionZ80 represents a concrete Z80 instruction
type machineInstructionZ80 struct {
	opcode        Z80Opcode
	result        *VirtualRegister
	operands      []*VirtualRegister
	conditionCode ConditionCode
	branchTargets []*BasicBlock
	comment       string
}

// newInstruction creates a new Z80 instruction
func newInstruction(opcode Z80Opcode, result, operand *VirtualRegister) *machineInstructionZ80 {
	operands := []*VirtualRegister{}
	if operand != nil {
		operands = append(operands, operand)
	}
	return &machineInstructionZ80{
		opcode:   opcode,
		result:   result,
		operands: operands,
	}
}
func newInstructionResult(opcode Z80Opcode, result *VirtualRegister) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode: opcode,
		result: result,
	}
}
func newInstructionOperand(opcode Z80Opcode, operand *VirtualRegister) *machineInstructionZ80 {
	return newInstruction(opcode, nil, operand)
}
func newInstruction0(opcode Z80Opcode) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode: opcode,
	}
}

// newBranchInternal is used when no basic block is needed (e.g., JR)
// displacement is relative offset of machine instructions (not bytes)
func newBranchInternal(condition ConditionCode, displacement *VirtualRegister) *machineInstructionZ80 {
	machInstr := newInstruction(Z80_JR_CC_E, nil, displacement)
	machInstr.conditionCode = condition
	return machInstr
}

// newJumpWithCondition creates a conditional jump with explicit condition code
func newJumpWithCondition(condition ConditionCode, trueBlock, falseBlock *BasicBlock) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        Z80_JP_CC_NN,
		conditionCode: condition,
		branchTargets: []*BasicBlock{trueBlock, falseBlock},
	}
}

// newJump creates an unconditional jump
func newJump(opcode Z80Opcode, target *BasicBlock) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        opcode,
		branchTargets: []*BasicBlock{target},
	}
}

// TODO: target block? or do we resolve them seperately after instruction selection?
// newCall creates a function call
func newCall(functionName string) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:  Z80_CALL_NN,
		comment: functionName,
	}
}

// Implement MachineInstruction interface

func (z *machineInstructionZ80) GetResult() *VirtualRegister {
	return z.result
}

func (z *machineInstructionZ80) GetOperands() []*VirtualRegister {
	return z.operands
}

func (z *machineInstructionZ80) SetResult(vr *VirtualRegister) {
	z.result = vr
}

func (z *machineInstructionZ80) SetOperand(index int, vr *VirtualRegister) {
	if index < len(z.operands) {
		z.operands[index] = vr
	}
}

func (z *machineInstructionZ80) GetCategory() InstrCategory {
	// Lookup from descriptor table
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		return desc.Category
	}
	return CatOther
}

func (z *machineInstructionZ80) GetAddressingMode() AddressingMode {
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		return desc.AddressingMode
	}
	return 0
}

func (z *machineInstructionZ80) GetTargetBlocks() []*BasicBlock {
	if z.branchTargets == nil {
		return []*BasicBlock{}
	}
	return z.branchTargets
}

func (z *machineInstructionZ80) GetCost() InstructionCost {
	cycles := uint8(0)
	bytes := uint8(0)

	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		cycles += desc.Cycles
		bytes += desc.Size
	} else {
		cycles = 255
		bytes = 255
	}

	return InstructionCost{Cycles: cycles, Size: bytes}
}

func (z *machineInstructionZ80) String() string {

	operands := make([]string, 0)
	if z.result != nil {
		operands = append(operands, z.result.String())
	}
	for _, op := range z.operands {
		operands = append(operands, op.String())
	}

	var builder strings.Builder
	builder.WriteString(FormatInstruction(z.opcode, z.conditionCode, operands...))
	if z.comment != "" {
		builder.WriteString(" ")
		builder.WriteString(z.comment)
		builder.WriteString(" ")
	}
	if len(z.branchTargets) > 0 {
		for _, target := range z.branchTargets {
			if target != nil {
				fmt.Fprintf(&builder, "Block %d ", target.ID)
			}
		}
	}
	return builder.String()
}
