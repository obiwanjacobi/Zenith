package cfg

import (
	"fmt"
	"strings"
)

// ============================================================================
// MachineInstruction interface (implemented in cfg/<target>)
// ============================================================================

// MachineInstruction is a target-specific instruction produced by the backend
// instruction selector. Concrete types live in cfg/<target>/ (e.g. cfg/z80/).
type MachineInstruction interface {
	GetResult() VROperand     // the single VROperand written by this instruction (nil if none)
	GetOperands() []VROperand // all VROperands read by this instruction
	String() string
}

// ============================================================================
// TAC operation codes
// ============================================================================

// TacOp identifies the binary arithmetic or logical operation in a TacBinOp.
type TacOp uint8

const (
	TacAdd TacOp = iota // dst = left + right
	TacSub              // dst = left - right
	TacMul              // dst = left * right  (result is 2× the operand size)
	TacDiv              // dst = left / right
	TacAnd              // dst = left & right
	TacOr               // dst = left | right
	TacXor              // dst = left ^ right
)

func (op TacOp) String() string {
	switch op {
	case TacAdd:
		return "ADD"
	case TacSub:
		return "SUB"
	case TacMul:
		return "MUL"
	case TacDiv:
		return "DIV"
	case TacAnd:
		return "AND"
	case TacOr:
		return "OR"
	case TacXor:
		return "XOR"
	default:
		return "?"
	}
}

// TacUnaryOp identifies the unary operation in a TacUnary.
type TacUnaryOp uint8

const (
	TacNegate     TacUnaryOp = iota // dst = -operand
	TacBitwiseNot                   // dst = ^operand
	TacIncrement                    // dst = operand + 1
	TacDecrement                    // dst = operand - 1
)

func (op TacUnaryOp) String() string {
	switch op {
	case TacNegate:
		return "NEG"
	case TacBitwiseNot:
		return "BNOT"
	case TacIncrement:
		return "INC"
	case TacDecrement:
		return "DEC"
	default:
		return "?"
	}
}

// TacCmpOp identifies the comparison in TacCompare and TacBranchCond.
type TacCmpOp uint8

const (
	TacCmpEqual     TacCmpOp = iota // =
	TacCmpNotEqual                  // <>
	TacCmpLess                      // <
	TacCmpLessEq                    // <=
	TacCmpGreater                   // >
	TacCmpGreaterEq                 // >=
)

func (op TacCmpOp) String() string {
	switch op {
	case TacCmpEqual:
		return "EQ"
	case TacCmpNotEqual:
		return "NEQ"
	case TacCmpLess:
		return "LT"
	case TacCmpLessEq:
		return "LE"
	case TacCmpGreater:
		return "GT"
	case TacCmpGreaterEq:
		return "GE"
	default:
		return "?"
	}
}

// ============================================================================
// TacInstruction interface
// ============================================================================

// TacInstruction is implemented by all TAC node types.
type TacInstruction interface {
	tacNode()          // unexported marker
	GetDst() VROperand // result VR; nil for instructions with no result
	String() string
}

// ============================================================================
// Binary and unary operations
// ============================================================================

// TacBinOp is a binary arithmetic or logical operation: Dst = Left Op Right.
// Size is the *operand* width in bits (8 or 16).
// For TacMul the result is 2× wider than Size; the Dst TempVR carries the
// correct result size.
type TacBinOp struct {
	Op    TacOp
	Dst   *TempVR
	Left  VROperand
	Right VROperand
	Size  uint8
}

func (t *TacBinOp) tacNode()          {}
func (t *TacBinOp) GetDst() VROperand { return t.Dst }
func (t *TacBinOp) String() string {
	return fmt.Sprintf("%s%d %s, %s, %s", t.Op, t.Size, t.Dst, t.Left, t.Right)
}

// TacUnary is a unary operation: Dst = Op Operand.
type TacUnary struct {
	Op      TacUnaryOp
	Dst     *TempVR
	Operand VROperand
	Size    uint8
}

func (t *TacUnary) tacNode()          {}
func (t *TacUnary) GetDst() VROperand { return t.Dst }
func (t *TacUnary) String() string {
	return fmt.Sprintf("%s%d %s, %s", t.Op, t.Size, t.Dst, t.Operand)
}

// ============================================================================
// Comparisons and branches
// ============================================================================

// TacCompare produces a boolean TempVR: Dst = (Left Op Right).
// Used when the comparison result is stored in a bit variable.
// Size is the operand width; the result is always 8-bit (bit type).
type TacCompare struct {
	Op    TacCmpOp
	Dst   *TempVR
	Left  VROperand
	Right VROperand
	Size  uint8 // operand size, not result size
}

func (t *TacCompare) tacNode()          {}
func (t *TacCompare) GetDst() VROperand { return t.Dst }
func (t *TacCompare) String() string {
	return fmt.Sprintf("CMP%d %s, %s %s %s", t.Size, t.Dst, t.Left, t.Op, t.Right)
}

// TacBranchCond is a comparison + conditional branch fused into one TAC node.
// Used when the comparison result feeds directly into a branch (BranchMode).
// The instruction selector emits a conditional jump directly from the flags;
// no boolean value is materialised into a register.
type TacBranchCond struct {
	Op    TacCmpOp
	Left  VROperand
	Right VROperand
	Size  uint8
	Then  *BasicBlock
	Else  *BasicBlock
}

func (t *TacBranchCond) tacNode()          {}
func (t *TacBranchCond) GetDst() VROperand { return nil }
func (t *TacBranchCond) String() string {
	return fmt.Sprintf("BRANCH%d %s %s %s ? Block%d : Block%d",
		t.Size, t.Left, t.Op, t.Right, t.Then.ID, t.Else.ID)
}

// TacBranchIf branches on a pre-computed boolean operand.
// Used when a bit variable is evaluated as an if-condition.
type TacBranchIf struct {
	Cond VROperand
	Then *BasicBlock
	Else *BasicBlock
}

func (t *TacBranchIf) tacNode()          {}
func (t *TacBranchIf) GetDst() VROperand { return nil }
func (t *TacBranchIf) String() string {
	return fmt.Sprintf("BRANCH_IF %s ? Block%d : Block%d", t.Cond, t.Then.ID, t.Else.ID)
}

// TacJump is an unconditional branch.
type TacJump struct {
	Target *BasicBlock
}

func (t *TacJump) tacNode()          {}
func (t *TacJump) GetDst() VROperand { return nil }
func (t *TacJump) String() string {
	return fmt.Sprintf("JUMP Block%d", t.Target.ID)
}

// TacCopy copies a value to a new TempVR: Dst = Src.
// Emitted at control-flow merge points to unify values from different paths,
// replacing φ nodes. The peephole pass removes self-copies after allocation.
type TacCopy struct {
	Dst  *TempVR
	Src  VROperand
	Size uint8
}

func (t *TacCopy) tacNode()          {}
func (t *TacCopy) GetDst() VROperand { return t.Dst }
func (t *TacCopy) String() string {
	return fmt.Sprintf("COPY%d %s, %s", t.Size, t.Dst, t.Src)
}

// ============================================================================
// Memory operations
// ============================================================================

// TacLoad loads from memory: Dst = *(Base + Offset).
type TacLoad struct {
	Dst    *TempVR
	Base   VROperand
	Offset uint16
	Size   uint8
}

func (t *TacLoad) tacNode()          {}
func (t *TacLoad) GetDst() VROperand { return t.Dst }
func (t *TacLoad) String() string {
	return fmt.Sprintf("LOAD%d %s, [%s+%d]", t.Size, t.Dst, t.Base, t.Offset)
}

// TacStore stores to memory: *(Base + Offset) = Value.
type TacStore struct {
	Base   VROperand
	Offset uint16
	Value  VROperand
	Size   uint8
}

func (t *TacStore) tacNode()          {}
func (t *TacStore) GetDst() VROperand { return nil }
func (t *TacStore) String() string {
	return fmt.Sprintf("STORE%d [%s+%d], %s", t.Size, t.Base, t.Offset, t.Value)
}

// TacLoadIndexed loads from memory with a dynamic index:
// Dst = *(Base + Index * ElemSize).
type TacLoadIndexed struct {
	Dst      *TempVR
	Base     VROperand
	Index    VROperand
	ElemSize uint8
	Size     uint8
}

func (t *TacLoadIndexed) tacNode()          {}
func (t *TacLoadIndexed) GetDst() VROperand { return t.Dst }
func (t *TacLoadIndexed) String() string {
	return fmt.Sprintf("LOAD_IDX%d %s, [%s + %s*%d]", t.Size, t.Dst, t.Base, t.Index, t.ElemSize)
}

// TacStoreIndexed stores to memory with a dynamic index:
// *(Base + Index * ElemSize) = Value.
type TacStoreIndexed struct {
	Base     VROperand
	Index    VROperand
	Value    VROperand
	ElemSize uint8
	Size     uint8
}

func (t *TacStoreIndexed) tacNode()          {}
func (t *TacStoreIndexed) GetDst() VROperand { return nil }
func (t *TacStoreIndexed) String() string {
	return fmt.Sprintf("STORE_IDX%d [%s + %s*%d], %s", t.Size, t.Base, t.Index, t.ElemSize, t.Value)
}

// TacInitSeq stores a sequence of literal values into contiguous memory:
//
//	Base[0] = Values[0], Base[1] = Values[1], ...
//
// ElemSize is the width of each element in bytes.
// Backends must not flatten this into independent TacStore instructions;
// they must compute the base address once and increment for each element.
type TacInitSeq struct {
	Base     VROperand
	ElemSize uint8
	Values   []VROperand
}

func (t *TacInitSeq) tacNode()          {}
func (t *TacInitSeq) GetDst() VROperand { return nil }
func (t *TacInitSeq) String() string {
	parts := make([]string, len(t.Values))
	for i, v := range t.Values {
		parts[i] = v.String()
	}
	return fmt.Sprintf("INIT_SEQ [%s], elem=%d, [%s]", t.Base, t.ElemSize, strings.Join(parts, ", "))
}

// TacStackAddr computes the address of a stack slot: Dst = SP + Offset.
type TacStackAddr struct {
	Dst    *TempVR
	Offset uint16
}

func (t *TacStackAddr) tacNode()          {}
func (t *TacStackAddr) GetDst() VROperand { return t.Dst }
func (t *TacStackAddr) String() string {
	return fmt.Sprintf("STACK_ADDR %s, SP+%d", t.Dst, t.Offset)
}

// ============================================================================
// Function call and return
// ============================================================================

// TacCall calls a function: Dst = Fn(Args...).
// Dst is nil for void calls.
type TacCall struct {
	Dst     *TempVR // nil for void calls
	Fn      string
	Args    []VROperand
	RetSize uint8
}

func (t *TacCall) tacNode()          {}
func (t *TacCall) GetDst() VROperand { return t.Dst }
func (t *TacCall) String() string {
	args := make([]string, len(t.Args))
	for i, a := range t.Args {
		args[i] = a.String()
	}
	if t.Dst != nil {
		return fmt.Sprintf("CALL %s = %s(%s)", t.Dst, t.Fn, strings.Join(args, ", "))
	}
	return fmt.Sprintf("CALL %s(%s)", t.Fn, strings.Join(args, ", "))
}

// TacReturn returns from the current function.
// Value is nil for void returns.
type TacReturn struct {
	Value VROperand // nil for void return
}

func (t *TacReturn) tacNode()          {}
func (t *TacReturn) GetDst() VROperand { return nil }
func (t *TacReturn) String() string {
	if t.Value != nil {
		return fmt.Sprintf("RETURN %s", t.Value)
	}
	return "RETURN"
}

// ===========================================================================

// DumpTAC prints the TAC instructions for every block in fnCFG to stdout.
func DumpTAC(fnName string, fnCFG *CFG) {
	fmt.Printf("========== TAC: %s ==========\n", fnName)
	for _, block := range fnCFG.Blocks {
		fmt.Printf("  Block %d [%s]:\n", block.ID, block.Label)
		for _, instr := range block.TAC {
			fmt.Printf("    %s\n", instr)
		}
	}
	fmt.Println()
}
