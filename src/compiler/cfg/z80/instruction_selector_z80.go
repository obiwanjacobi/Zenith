package z80

import (
	"zenith/compiler/cfg"
)

// ============================================================================
// Z80 Instruction Selector
// ============================================================================

// instructionSelectorZ80 implements cfg.InstructionSelector for the Z80 target.
// It holds a shared TempVRAllocator and the calling convention for the function
// being compiled. One selector instance is used per function.
type instructionSelectorZ80 struct {
	alloc *cfg.TempVRAllocator
	cc    CallingConvention
}

// NewInstructionSelectorZ80 returns a new Z80 instruction selector.
func NewInstructionSelectorZ80(alloc *cfg.TempVRAllocator) cfg.InstructionSelector {
	return &instructionSelectorZ80{alloc: alloc, cc: NewCallingConventionZ80()}
}

// ── Prologue / Epilogue ───────────────────────────────────────────────────────

// BindParameters emits a copy from each calling-convention register into its
// parameter TempVR. Called PRE-regalloc so that the AllowedSet constraint is
// visible to constraint propagation and the linear scan.
func (s *instructionSelectorZ80) BindParameters(entryBlock *cfg.BasicBlock, fnCFG *cfg.CFG) {
	params := fnCFG.FunctionDecl.Parameters
	for i, paramVR := range fnCFG.ParamVRs {
		if i >= len(params) {
			break
		}
		size := paramVR.Size()
		reg, _, onStack := s.cc.GetParameterLocation(i, size)
		if onStack {
			// Stack parameters are not in a register; leave the VR unconstrained
			// so regalloc handles it normally.
			continue
		}
		// Allocate a source VR pinned to the ABI register and emit LD paramVR, ccReg.
		ccVR := s.alloc.Alloc(size, []*cfg.Register{reg})
		paramVR.ConstrainTo([]*cfg.Register{reg})
		emitInstr(entryBlock, Z80_LD_R_R, paramVR, ccVR, nil)
	}
}

// SelectPrologue emits the stack frame setup into the entry block. Called
// POST-regalloc so StackFrame.Size() includes any spill slots. Operands are
// PhysVRs and do not go through register allocation.
//
//	LD HL, -frameSize
//	ADD HL, SP
//	LD SP, HL
func (s *instructionSelectorZ80) SelectPrologue(entryBlock *cfg.BasicBlock, fnCFG *cfg.CFG) {
	frameSize := fnCFG.StackFrame.Size()
	if frameSize == 0 {
		return
	}
	hlPhys := &cfg.PhysVR{Reg: &RegHL}
	spPhys := &cfg.PhysVR{Reg: &RegSP}
	setup := []cfg.MachineInstruction{
		&MachineInstrZ80{Opcode: Z80_LD_RR_NN, Result: hlPhys, Src1: cfg.NewImmVR(-int32(frameSize), 16)},
		&MachineInstrZ80{Opcode: Z80_ADD_HL_RR, Result: hlPhys, Src1: hlPhys, Src2: spPhys},
		&MachineInstrZ80{Opcode: Z80_LD_SP_HL, Result: spPhys, Src1: hlPhys},
	}
	entryBlock.MachineInstructions = append(setup, entryBlock.MachineInstructions...)
}

// SelectEpilogue emits the stack frame teardown into the exit block, prepended
// before the return-value move and RET. Called POST-regalloc. Operands are
// PhysVRs and do not go through register allocation.
//
//	LD HL, frameSize
//	ADD HL, SP
//	LD SP, HL
func (s *instructionSelectorZ80) SelectEpilogue(exitBlock *cfg.BasicBlock, fnCFG *cfg.CFG) {
	frameSize := fnCFG.StackFrame.Size()
	if frameSize == 0 {
		exitBlock.MachineInstructions = append(exitBlock.MachineInstructions,
			&MachineInstrZ80{Opcode: Z80_RET},
		)
		return
	}
	hlPhys := &cfg.PhysVR{Reg: &RegHL}
	spPhys := &cfg.PhysVR{Reg: &RegSP}
	teardown := []cfg.MachineInstruction{
		&MachineInstrZ80{Opcode: Z80_LD_RR_NN, Result: hlPhys, Src1: cfg.NewImmVR(int32(frameSize), 16)},
		&MachineInstrZ80{Opcode: Z80_ADD_HL_RR, Result: hlPhys, Src1: hlPhys, Src2: spPhys},
		&MachineInstrZ80{Opcode: Z80_LD_SP_HL, Result: spPhys, Src1: hlPhys},
		&MachineInstrZ80{Opcode: Z80_RET},
	}
	exitBlock.MachineInstructions = append(teardown, exitBlock.MachineInstructions...)
}

// ── Memory: Load ─────────────────────────────────────────────────────────────

// SelectLoad: Dst = *(Base + Offset)
//
// Z80 sequence:
//
//	LD HL, base      ; base address into HL
//	(if offset > 0)
//	  LD DE, offset  ; offset into DE
//	  ADD HL, DE
//	LD A, (HL)       ; load byte  (8-bit)  — or LD r,(HL); INC HL; LD H,(HL); LD L,r  (16-bit)
//	LD dst, A
func (s *instructionSelectorZ80) SelectLoad(block *cfg.BasicBlock, t *cfg.TacLoad) {
	hlVR := s.addressWithOffset(block, t.Base, t.Offset)

	if t.Size == 8 {
		// LD r, (HL)
		aVR := s.any8()
		emitInstr(block, Z80_LD_R_HL, aVR, hlVR, nil)

		// LD dst, r  (peephole removes if dst ends up in same register)
		t.Dst.ConstrainTo(Z80Registers8)
		emitInstr(block, Z80_LD_R_R, t.Dst, aVR, nil)
	} else {
		s.emitLoadHL16(block, hlVR, t.Dst)
	}
}

// SelectLoadIndexed: Dst = *(Base + Index*ElemSize)
//
// Z80 sequence:
//
//	LD HL, base
//	LD DE, index     ; (ElemSize scaling handled by multiplying index in TAC lowering
//	                 ;  or by shifting here; for ElemSize==1 or 2 we emit directly)
//	(if ElemSize==2: ADD HL,DE once more, i.e. SLA DE equivalent via ADD HL,HL)
//	ADD HL, DE
//	LD A, (HL)       ; 8-bit load
//	LD dst, A
func (s *instructionSelectorZ80) SelectLoadIndexed(block *cfg.BasicBlock, t *cfg.TacLoadIndexed) {
	hlVR := s.moveToHL(block, t.Base)
	deVR := s.scaleIndex(block, t.Index, t.ElemSize)

	// ADD HL, DE
	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlVR, deVR)

	if t.Size == 8 {
		aVR := s.any8()
		emitInstr(block, Z80_LD_R_HL, aVR, addHL, nil)
		t.Dst.ConstrainTo(Z80Registers8)
		emitInstr(block, Z80_LD_R_R, t.Dst, aVR, nil)
	} else {
		s.emitLoadHL16(block, addHL, t.Dst)
	}
}

// ── Memory: Store ────────────────────────────────────────────────────────────

// SelectStore: *(Base + Offset) = Value
//
// Z80 sequence:
//
//	LD HL, base
//	(offset adjustment)
//	LD A, value      ; 8-bit
//	LD (HL), A
func (s *instructionSelectorZ80) SelectStore(block *cfg.BasicBlock, t *cfg.TacStore) {
	hlVR := s.addressWithOffset(block, t.Base, t.Offset)

	if t.Size == 8 {
		rVR := s.any8()
		emitInstr(block, Z80_LD_R_R, rVR, t.Value, nil)
		emitInstr(block, Z80_LD_HL_R, hlVR, rVR, nil)
	} else {
		// 16-bit: store lo byte then hi byte using any PP register (BC or DE).
		s.emitStoreHL16(block, hlVR, t.Value)
	}
}

// SelectStoreIndexed: *(Base + Index*ElemSize) = Value
func (s *instructionSelectorZ80) SelectStoreIndexed(block *cfg.BasicBlock, t *cfg.TacStoreIndexed) {
	hlVR := s.moveToHL(block, t.Base)
	deVR := s.scaleIndex(block, t.Index, t.ElemSize)

	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlVR, deVR)

	if t.Size == 8 {
		rVR := s.any8()
		emitInstr(block, Z80_LD_R_R, rVR, t.Value, nil)
		emitInstr(block, Z80_LD_HL_R, addHL, rVR, nil)
	} else {
		// 16-bit: store lo byte then hi byte using any PP register (BC or DE).
		s.emitStoreHL16(block, addHL, t.Value)
	}
}

// SelectInitSeq: Base[0]=v0, Base[1]=v1, ...
//
// Z80 sequence:
//
//	LD HL, base
//	for each value:
//	  LD A, value   (or LD (HL), n for immediate)
//	  LD (HL), A
//	  INC HL        (omit after last element)
func (s *instructionSelectorZ80) SelectInitSeq(block *cfg.BasicBlock, t *cfg.TacInitSeq) {
	hlVR := s.moveToHL(block, t.Base)

	for i, val := range t.Values {
		if t.ElemSize == 1 {
			// LD r, val ; LD (HL), r
			rVR := s.any8()
			emitInstr(block, Z80_LD_R_R, rVR, val, nil)
			emitInstr(block, Z80_LD_HL_R, hlVR, rVR, nil)
		} else {
			// 16-bit element: store lo byte then hi byte using any PP register (BC or DE).
			hlVR = s.emitStoreHL16(block, hlVR, val)
		}

		// INC HL between elements (but not after the last one).
		if i < len(t.Values)-1 {
			incHL := s.reg16(&RegHL)
			emitInstr(block, Z80_INC_RR, incHL, hlVR, nil)
			hlVR = incHL
		}
	}
}

// ── Stack address ─────────────────────────────────────────────────────────────

// SelectStackAddr: Dst = SP + Offset
//
// Z80 sequence:
//
//	LD HL, offset
//	ADD HL, SP
//	LD dst, HL
func (s *instructionSelectorZ80) SelectStackAddr(block *cfg.BasicBlock, t *cfg.TacStackAddr) {
	// LD HL, offset (immediate)
	offsetVR := cfg.NewImmVR(int32(t.Offset), 16)
	hlImm := s.reg16(&RegHL)
	emitInstr(block, Z80_LD_RR_NN, hlImm, offsetVR, nil)

	// ADD HL, SP
	spVR := s.reg16(&RegSP)
	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlImm, spVR)

	// LD dst, HL  (dst is any 16-bit — peephole removes self-move)
	t.Dst.ConstrainTo([]*cfg.Register{&RegHL})
	emitInstr(block, Z80_LD_R_R, t.Dst, addHL, nil)
}

// ── Copy / move ───────────────────────────────────────────────────────────────

// SelectCopy: Dst = Src
//
// 8-bit: Z80_LD_R_R (single instruction).
// 16-bit: emitLD16 — two Z80_LD_R_R on the lo/hi sub-registers.
func (s *instructionSelectorZ80) SelectCopy(block *cfg.BasicBlock, t *cfg.TacCopy) {
	if t.Size == 8 {
		emitInstr(block, Z80_LD_R_R, t.Dst, t.Src, nil)
	} else {
		s.emitLD16(block, t.Dst, t.Src)
	}
}

// ── Arithmetic / logical ──────────────────────────────────────────────────────

// SelectBinOp: Dst = Left Op Right
func (s *instructionSelectorZ80) SelectBinOp(block *cfg.BasicBlock, t *cfg.TacBinOp) {
	switch t.Op {
	case cfg.TacAdd:
		if t.Size == 8 {
			s.selectAdd8(block, t)
		} else {
			s.selectAdd16(block, t)
		}
	case cfg.TacSub:
		if t.Size == 8 {
			s.selectSub8(block, t)
		} else {
			s.selectSub16(block, t)
		}
	case cfg.TacMul:
		if t.Size == 8 {
			s.selectMulDiv8(block, t, "__mul8")
		} else {
			s.selectMulDiv16(block, t, "__mul16")
		}
	case cfg.TacDiv:
		if t.Size == 8 {
			s.selectMulDiv8(block, t, "__div8")
		} else {
			s.selectMulDiv16(block, t, "__div16")
		}
	case cfg.TacAnd:
		if t.Size == 8 {
			s.selectBitwise8(block, Z80_AND_R, Z80_AND_N, t.Left, t.Right, t.Dst)
		} else {
			s.selectBitwise16(block, Z80_AND_R, t.Left, t.Right, t.Dst)
		}
	case cfg.TacOr:
		if t.Size == 8 {
			s.selectBitwise8(block, Z80_OR_R, Z80_OR_N, t.Left, t.Right, t.Dst)
		} else {
			s.selectBitwise16(block, Z80_OR_R, t.Left, t.Right, t.Dst)
		}
	case cfg.TacXor:
		if t.Size == 8 {
			s.selectBitwise8(block, Z80_XOR_R, Z80_XOR_N, t.Left, t.Right, t.Dst)
		} else {
			s.selectBitwise16(block, Z80_XOR_R, t.Left, t.Right, t.Dst)
		}
	default:
		panic("SelectBinOp: unhandled TacOp")
	}
}

// selectAdd8: A = left + right  →  Dst
//
//	LD A, left
//	ADD A, right   (ADD A, n if right is immediate)
//	LD dst, A
func (s *instructionSelectorZ80) selectAdd8(block *cfg.BasicBlock, t *cfg.TacBinOp) {
	aVR := s.reg8(&RegA)
	emitInstr(block, Z80_LD_R_R, aVR, t.Left, nil)

	addVR := s.reg8(&RegA)
	if _, isImm := t.Right.(*cfg.ImmVR); isImm {
		emitInstr(block, Z80_ADD_A_N, addVR, aVR, t.Right)
	} else {
		emitInstr(block, Z80_ADD_A_R, addVR, aVR, t.Right)
	}

	t.Dst.ConstrainTo([]*cfg.Register{&RegA})
	emitInstr(block, Z80_LD_R_R, t.Dst, addVR, nil)
}

// selectSub8: A = left - right  →  Dst
//
//	LD A, left
//	SUB right      (SUB n if right is immediate)
//	LD dst, A
func (s *instructionSelectorZ80) selectSub8(block *cfg.BasicBlock, t *cfg.TacBinOp) {
	aVR := s.reg8(&RegA)
	emitInstr(block, Z80_LD_R_R, aVR, t.Left, nil)

	subVR := s.reg8(&RegA)
	if _, isImm := t.Right.(*cfg.ImmVR); isImm {
		emitInstr(block, Z80_SUB_N, subVR, aVR, t.Right)
	} else {
		emitInstr(block, Z80_SUB_R, subVR, aVR, t.Right)
	}

	t.Dst.ConstrainTo([]*cfg.Register{&RegA})
	emitInstr(block, Z80_LD_R_R, t.Dst, subVR, nil)
}

// selectAdd16: HL = left + right  →  Dst
//
//	LD HL, left
//	LD pp, right   (pp = BC or DE)
//	ADD HL, pp
//	LD dst, HL
func (s *instructionSelectorZ80) selectAdd16(block *cfg.BasicBlock, t *cfg.TacBinOp) {
	hlVR := s.reg16(&RegHL)
	s.emitLD16(block, hlVR, t.Left)

	ppVR := s.alloc.Alloc(16, Z80RegistersPP)
	s.emitLD16(block, ppVR, t.Right)

	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlVR, ppVR)

	t.Dst.ConstrainTo([]*cfg.Register{&RegHL})
	s.emitLD16(block, t.Dst, addHL)
}

// selectSub16: HL = left - right  →  Dst
//
//	LD HL, left
//	LD pp, right   (pp = BC or DE)
//	AND A          ; clear carry flag
//	SBC HL, pp
//	LD dst, HL
func (s *instructionSelectorZ80) selectSub16(block *cfg.BasicBlock, t *cfg.TacBinOp) {
	hlVR := s.reg16(&RegHL)
	s.emitLD16(block, hlVR, t.Left)

	ppVR := s.alloc.Alloc(16, Z80RegistersPP)
	s.emitLD16(block, ppVR, t.Right)

	// AND A clears the carry flag without altering HL or the second operand.
	aVR := s.reg8(&RegA)
	emitInstr(block, Z80_AND_R, aVR, aVR, nil)

	sbcHL := s.reg16(&RegHL)
	emitInstr(block, Z80_SBC_HL_RR, sbcHL, hlVR, ppVR)

	t.Dst.ConstrainTo([]*cfg.Register{&RegHL})
	s.emitLD16(block, t.Dst, sbcHL)
}

// selectMulDiv8: delegate 8-bit mul/div to a runtime helper.
//
// Calling convention: param0→E, param1→C; return→A (but helper returns u16 in DE for mul).
//
//	LD E, left
//	LD C, right
//	CALL __mul8 / __div8
//	LD dst, A   (div) or LD dst, DE  (mul, result is u16)
func (s *instructionSelectorZ80) selectMulDiv8(block *cfg.BasicBlock, t *cfg.TacBinOp, helper string) {
	eVR := s.reg8(&RegE)
	emitInstr(block, Z80_LD_R_R, eVR, t.Left, nil)

	cVR := s.reg8(&RegC)
	emitInstr(block, Z80_LD_R_R, cVR, t.Right, nil)

	if helper == "__mul8" {
		// mul8 widens: u8 × u8 → u16 result in DE
		resultVR := s.reg16(&RegDE)
		emitCall(block, helper, resultVR)
		t.Dst.ConstrainTo([]*cfg.Register{&RegDE})
		s.emitLD16(block, t.Dst, resultVR)
	} else {
		// div8 quotient in A
		resultVR := s.reg8(&RegA)
		emitCall(block, helper, resultVR)
		t.Dst.ConstrainTo([]*cfg.Register{&RegA})
		emitInstr(block, Z80_LD_R_R, t.Dst, resultVR, nil)
	}
}

// selectMulDiv16: delegate 16-bit mul/div to a runtime helper.
//
// Calling convention: param0→DE, param1→BC; return→DE.
//
//	LD DE, left
//	LD BC, right
//	CALL __mul16 / __div16
//	LD dst, DE
func (s *instructionSelectorZ80) selectMulDiv16(block *cfg.BasicBlock, t *cfg.TacBinOp, helper string) {
	deVR := s.reg16(&RegDE)
	s.emitLD16(block, deVR, t.Left)

	bcVR := s.reg16(&RegBC)
	s.emitLD16(block, bcVR, t.Right)

	resultVR := s.reg16(&RegDE)
	emitCall(block, helper, resultVR)

	t.Dst.ConstrainTo([]*cfg.Register{&RegDE})
	s.emitLD16(block, t.Dst, resultVR)
}

// ── Comparisons and control flow ──────────────────────────────────────────────

// SelectBranchCond: fused compare + conditional branch.
//
// 8-bit:  LD A, left ; CP right ; JP cc, then ; JP else
// 16-bit: LD HL, left ; LD DE, right ; AND A ; SBC HL, DE ; JP cc, then ; JP else
//
// GT and LE swap operands so the carry-based condition code works correctly.
func (s *instructionSelectorZ80) SelectBranchCond(block *cfg.BasicBlock, t *cfg.TacBranchCond) {
	cc, swap := cmpOpToCondAndSwap(t.Op)
	left, right := t.Left, t.Right
	if swap {
		left, right = right, left
	}

	if t.Size == 8 {
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_LD_R_R, aVR, left, nil)
		if _, isImm := right.(*cfg.ImmVR); isImm {
			emitInstr(block, Z80_CP_N, nil, aVR, right)
		} else {
			emitInstr(block, Z80_CP_R, nil, aVR, right)
		}
	} else {
		hlVR := s.reg16(&RegHL)
		s.emitLD16(block, hlVR, left)
		ppVR := s.alloc.Alloc(16, Z80RegistersPP)
		s.emitLD16(block, ppVR, right)
		// AND A clears carry before SBC HL, pp.
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_AND_R, aVR, aVR, nil)
		sbcHL := s.reg16(&RegHL)
		emitInstr(block, Z80_SBC_HL_RR, sbcHL, hlVR, ppVR)
	}

	emitBranch(block, Z80_JP_CC_NN, cc, t.Then)
	emitBranch(block, Z80_JP_NN, Cond_None, t.Else)
}

// SelectCompare: Dst = (Left Op Right), materialising a bit value into a register.
//
// Pattern (8-bit):
//
//	LD A, left
//	CP right       ; (CP n for immediates)
//	LD A, 0
//	JR inverse_cc, +1   ; skip INC A when condition is false
//	INC A               ; A = 1 when condition is true
//	LD dst, A
//
// The JR offset is always +1 because INC r is exactly 1 byte.
// 16-bit uses AND A; SBC HL, DE to set the same flags, then the same tail.
func (s *instructionSelectorZ80) SelectCompare(block *cfg.BasicBlock, t *cfg.TacCompare) {
	cc, swap := cmpOpToCondAndSwap(t.Op)
	left, right := t.Left, t.Right
	if swap {
		left, right = right, left
	}

	if t.Size == 8 {
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_LD_R_R, aVR, left, nil)
		if _, isImm := right.(*cfg.ImmVR); isImm {
			emitInstr(block, Z80_CP_N, nil, aVR, right)
		} else {
			emitInstr(block, Z80_CP_R, nil, aVR, right)
		}
	} else {
		hlVR := s.reg16(&RegHL)
		s.emitLD16(block, hlVR, left)
		ppVR := s.alloc.Alloc(16, Z80RegistersPP)
		s.emitLD16(block, ppVR, right)
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_AND_R, aVR, aVR, nil) // clear carry
		sbcHL := s.reg16(&RegHL)
		emitInstr(block, Z80_SBC_HL_RR, sbcHL, hlVR, ppVR)
	}

	// Materialise: LD A, 0 ; JR inverse_cc, +1 ; INC A
	zeroA := s.reg8(&RegA)
	emitInstr(block, Z80_LD_R_N, zeroA, cfg.NewImmVR(0, 8), nil)
	emitRelJump(block, invertCond(cc), 1) // skip INC A if condition false
	oneA := s.reg8(&RegA)
	emitInstr(block, Z80_INC_R, oneA, zeroA, nil)

	t.Dst.ConstrainTo([]*cfg.Register{&RegA})
	emitInstr(block, Z80_LD_R_R, t.Dst, oneA, nil)
}

// SelectJump: unconditional branch to target.
func (s *instructionSelectorZ80) SelectJump(block *cfg.BasicBlock, t *cfg.TacJump) {
	emitBranch(block, Z80_JP_NN, Cond_None, t.Target)
}

// SelectBranchIf: branch on a pre-computed boolean (bit value in a register).
//
//	LD A, cond
//	AND A          ; set Z if zero, NZ if non-zero
//	JP NZ, then    ; jump to then if true
//	JP else
func (s *instructionSelectorZ80) SelectBranchIf(block *cfg.BasicBlock, t *cfg.TacBranchIf) {
	aVR := s.reg8(&RegA)
	emitInstr(block, Z80_LD_R_R, aVR, t.Cond, nil)
	emitInstr(block, Z80_AND_R, aVR, aVR, nil)
	emitBranch(block, Z80_JP_CC_NN, Cond_NZ, t.Then)
	emitBranch(block, Z80_JP_NN, Cond_None, t.Else)
}

// ── Calls and returns ─────────────────────────────────────────────────────────

// SelectCall: Dst = Fn(Args...)
//
// Arguments are placed into registers using the calling convention:
//
//	param 0 (8-bit) → E,  param 0 (16-bit) → DE
//	param 1 (8-bit) → C,  param 1 (16-bit) → BC
//	further params  → stack (not yet implemented)
//
// Return value is moved into Dst if non-nil:
//
//	8-bit return  → A
//	16-bit return → DE
func (s *instructionSelectorZ80) SelectCall(block *cfg.BasicBlock, t *cfg.TacCall) {
	for i, arg := range t.Args {
		reg, _, onStack := s.cc.GetParameterLocation(i, arg.Size())
		if onStack {
			panic("SelectCall: stack-passed arguments not yet implemented")
		}
		argVR := s.alloc.Alloc(arg.Size(), []*cfg.Register{reg})
		if arg.Size() == 8 {
			emitInstr(block, Z80_LD_R_R, argVR, arg, nil)
		} else {
			s.emitLD16(block, argVR, arg)
		}
	}

	var resultVR *cfg.TempVR
	if t.Dst != nil {
		retReg := s.cc.GetReturnValueRegister(t.RetSize)
		resultVR = s.alloc.Alloc(t.RetSize, []*cfg.Register{retReg})
	}
	emitCall(block, t.Fn, resultVR)

	if t.Dst != nil {
		retReg := s.cc.GetReturnValueRegister(t.RetSize)
		t.Dst.ConstrainTo([]*cfg.Register{retReg})
		if t.RetSize == 8 {
			emitInstr(block, Z80_LD_R_R, t.Dst, resultVR, nil)
		} else {
			s.emitLD16(block, t.Dst, resultVR)
		}
	}
}

// SelectReturn: return [Value]
//
// Move Value into the return register (A for 8-bit, DE for 16-bit) then jump
// to the exit block. The exit block's epilogue (SelectEpilogue) emits RET.
func (s *instructionSelectorZ80) SelectReturn(block *cfg.BasicBlock, exitBlock *cfg.BasicBlock, t *cfg.TacReturn) {
	if t.Value != nil {
		retReg := s.cc.GetReturnValueRegister(t.Value.Size())
		retVR := s.alloc.Alloc(t.Value.Size(), []*cfg.Register{retReg})
		if t.Value.Size() == 8 {
			emitInstr(block, Z80_LD_R_R, retVR, t.Value, nil)
		} else {
			s.emitLD16(block, retVR, t.Value)
		}
	}
	emitBranch(block, Z80_JP_NN, Cond_None, exitBlock)
}

// ── Unary operations ──────────────────────────────────────────────────────────

// SelectUnary: Dst = Op Operand
//
//	INC / DEC: INC r / DEC r (8-bit); INC rr / DEC rr (16-bit)
//	NEG:       LD A, operand ; NEG ; LD dst, A
//	BNOT:      LD A, operand ; CPL ; LD dst, A  (8-bit only; 16-bit: byte-wise)
func (s *instructionSelectorZ80) SelectUnary(block *cfg.BasicBlock, t *cfg.TacUnary) {
	switch t.Op {
	case cfg.TacIncrement:
		if t.Size == 8 {
			incVR := s.any8()
			emitInstr(block, Z80_LD_R_R, incVR, t.Operand, nil)
			resVR := s.any8()
			emitInstr(block, Z80_INC_R, resVR, incVR, nil)
			t.Dst.ConstrainTo(Z80Registers8)
			emitInstr(block, Z80_LD_R_R, t.Dst, resVR, nil)
		} else {
			hlVR := s.reg16(&RegHL)
			s.emitLD16(block, hlVR, t.Operand)
			resVR := s.reg16(&RegHL)
			emitInstr(block, Z80_INC_RR, resVR, hlVR, nil)
			t.Dst.ConstrainTo([]*cfg.Register{&RegHL})
			s.emitLD16(block, t.Dst, resVR)
		}
	case cfg.TacDecrement:
		if t.Size == 8 {
			decVR := s.any8()
			emitInstr(block, Z80_LD_R_R, decVR, t.Operand, nil)
			resVR := s.any8()
			emitInstr(block, Z80_DEC_R, resVR, decVR, nil)
			t.Dst.ConstrainTo(Z80Registers8)
			emitInstr(block, Z80_LD_R_R, t.Dst, resVR, nil)
		} else {
			hlVR := s.reg16(&RegHL)
			s.emitLD16(block, hlVR, t.Operand)
			resVR := s.reg16(&RegHL)
			emitInstr(block, Z80_DEC_RR, resVR, hlVR, nil)
			t.Dst.ConstrainTo([]*cfg.Register{&RegHL})
			s.emitLD16(block, t.Dst, resVR)
		}
	case cfg.TacNegate:
		// NEG only operates on A.
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_LD_R_R, aVR, t.Operand, nil)
		resVR := s.reg8(&RegA)
		emitInstr(block, Z80_NEG, resVR, aVR, nil)
		t.Dst.ConstrainTo([]*cfg.Register{&RegA})
		emitInstr(block, Z80_LD_R_R, t.Dst, resVR, nil)
	case cfg.TacBitwiseNot:
		panic("SelectUnary: TacBitwiseNot not yet implemented (Z80_CPL opcode not defined)")
	}
}

// selectBitwise8: A = left op right  →  Dst
//
//	LD A, left
//	AND/OR/XOR right   (immediate variant if right is ImmVR)
//	LD dst, A
func (s *instructionSelectorZ80) selectBitwise8(block *cfg.BasicBlock, regOp, immOp Z80Opcode, left, right cfg.VROperand, dst *cfg.TempVR) {
	aVR := s.reg8(&RegA)
	emitInstr(block, Z80_LD_R_R, aVR, left, nil)

	resVR := s.reg8(&RegA)
	if _, isImm := right.(*cfg.ImmVR); isImm {
		emitInstr(block, immOp, resVR, aVR, right)
	} else {
		emitInstr(block, regOp, resVR, aVR, right)
	}

	dst.ConstrainTo([]*cfg.Register{&RegA})
	emitInstr(block, Z80_LD_R_R, dst, resVR, nil)
}

// selectBitwise16: HL = left op right  →  Dst
//
// Z80 has no 16-bit AND/OR/XOR, so operate byte-wise on H/L and D/E.
//
//	LD HL, left ; LD DE, right
//	LD A, L ; AND/OR/XOR E ; LD L, A
//	LD A, H ; AND/OR/XOR D ; LD H, A
//	LD dst, HL
func (s *instructionSelectorZ80) selectBitwise16(block *cfg.BasicBlock, regOp Z80Opcode, left, right cfg.VROperand, dst *cfg.TempVR) {
	hlVR := s.reg16(&RegHL)
	s.emitLD16(block, hlVR, left)
	deVR := s.reg16(&RegDE)
	s.emitLD16(block, deVR, right)

	// Low byte: L op E → L
	aVR := s.reg8(&RegA)
	lVR := s.reg8(&RegL)
	emitInstr(block, Z80_LD_R_R, aVR, lVR, nil)
	eVR := s.reg8(&RegE)
	resL := s.reg8(&RegA)
	emitInstr(block, regOp, resL, aVR, eVR)
	newL := s.reg8(&RegL)
	emitInstr(block, Z80_LD_R_R, newL, resL, nil)

	// High byte: H op D → H
	a2VR := s.reg8(&RegA)
	hVR := s.reg8(&RegH)
	emitInstr(block, Z80_LD_R_R, a2VR, hVR, nil)
	dVR := s.reg8(&RegD)
	resH := s.reg8(&RegA)
	emitInstr(block, regOp, resH, a2VR, dVR)
	newH := s.reg8(&RegH)
	emitInstr(block, Z80_LD_R_R, newH, resH, nil)

	dst.ConstrainTo([]*cfg.Register{&RegHL})
	s.emitLD16(block, dst, hlVR)
}

// invertCond returns the logical inverse of a condition code.
func invertCond(cc ConditionCode) ConditionCode {
	switch cc {
	case Cond_Z:
		return Cond_NZ
	case Cond_NZ:
		return Cond_Z
	case Cond_C:
		return Cond_NC
	case Cond_NC:
		return Cond_C
	default:
		panic("invertCond: unsupported ConditionCode")
	}
}

// cmpOpToCondAndSwap returns the Z80 condition code for a TacCmpOp and whether// the left/right operands should be swapped before emitting the comparison.
// All mappings assume unsigned semantics (CP / SBC HL,rr carry flag).
func cmpOpToCondAndSwap(op cfg.TacCmpOp) (ConditionCode, bool) {
	switch op {
	case cfg.TacCmpEqual:
		return Cond_Z, false
	case cfg.TacCmpNotEqual:
		return Cond_NZ, false
	case cfg.TacCmpLess:
		return Cond_C, false
	case cfg.TacCmpGreaterEq:
		return Cond_NC, false
	case cfg.TacCmpGreater:
		return Cond_C, true // emit CP left with right in A → carry if right < left
	case cfg.TacCmpLessEq:
		return Cond_NC, true // emit CP left with right in A → no-carry if right >= left
	default:
		panic("cmpOpToCondAndSwap: unknown TacCmpOp")
	}
}

// ── Private helpers ─────────────────────────────────────────────────────────────

// moveToHL loads src into HL and returns the constrained HL TempVR.
func (s *instructionSelectorZ80) moveToHL(block *cfg.BasicBlock, src cfg.VROperand) *cfg.TempVR {
	hlVR := s.reg16(&RegHL)
	s.emitLD16(block, hlVR, src)
	return hlVR
}

// addressWithOffset returns a TempVR in HL pointing to Base+Offset.
// If offset is 0 it simply moves Base into HL.
// If offset > 0 it additionally loads offset into DE and adds.
func (s *instructionSelectorZ80) addressWithOffset(block *cfg.BasicBlock, base cfg.VROperand, offset uint16) *cfg.TempVR {
	hlVR := s.moveToHL(block, base)
	if offset == 0 {
		return hlVR
	}

	// LD pp, offset  (pp = BC or DE)
	offsetImm := cfg.NewImmVR(int32(offset), 16)
	ppVR := s.alloc.Alloc(16, Z80RegistersPP)
	emitInstr(block, Z80_LD_RR_NN, ppVR, offsetImm, nil)

	// ADD HL, pp
	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlVR, ppVR)
	return addHL
}

// scaleIndex returns a TempVR in a PP register (BC or DE) holding Index scaled
// by ElemSize. ElemSize 1: pp = Index (zero-extended from 8-bit if needed).
// ElemSize 2: pp = Index * 2 (via ADD HL,HL on a scratch HL, result moved to pp).
func (s *instructionSelectorZ80) scaleIndex(block *cfg.BasicBlock, index cfg.VROperand, elemSize uint8) *cfg.TempVR {
	ppVR := s.alloc.Alloc(16, Z80RegistersPP)

	if index.Size() == 8 {
		// Zero-extend 8-bit index: lo = index, hi = 0
		ppLos, ppHis := cfg.ToPairs(ppVR.AllowedSet())
		loVR := s.alloc.Alloc(8, ppLos)
		emitInstr(block, Z80_LD_R_R, loVR, index, nil)
		hiVR := s.alloc.Alloc(8, ppHis)
		emitInstr(block, Z80_LD_R_N, hiVR, cfg.NewImmVR(0, 8), nil)
	} else {
		// 16-bit index: load directly into pp.
		s.emitLD16(block, ppVR, index)
	}

	if elemSize == 2 {
		// pp *= 2: move to HL, ADD HL,HL, move back to pp.
		hlVR := s.reg16(&RegHL)
		s.emitLD16(block, hlVR, ppVR)
		emitInstr(block, Z80_ADD_HL_RR, hlVR, hlVR, hlVR)
		s.emitLD16(block, ppVR, hlVR)
	}

	return ppVR
}

// emitLoadHL16 loads a 16-bit value from (HL):(HL+1) into dst.
// lo byte constrained to {C,E}, hi byte to {B,D} so constraint propagation
// can unify them with dst's sub-registers and eliminate the copy instructions.
func (s *instructionSelectorZ80) emitLoadHL16(block *cfg.BasicBlock, hlVR *cfg.TempVR, dst *cfg.TempVR) {
	loVR := s.alloc.Alloc(8, []*cfg.Register{&RegC, &RegE})
	emitInstr(block, Z80_LD_R_HL, loVR, hlVR, nil)

	incHL := s.reg16(&RegHL)
	emitInstr(block, Z80_INC_RR, incHL, hlVR, nil)

	hiVR := s.alloc.Alloc(8, []*cfg.Register{&RegB, &RegD})
	emitInstr(block, Z80_LD_R_HL, hiVR, incHL, nil)

	dst.ConstrainTo(Z80RegistersPP)
	dstLos, dstHis := cfg.ToPairs(dst.AllowedSet())
	emitInstr(block, Z80_LD_R_R, s.alloc.Alloc(8, dstLos), loVR, nil)
	emitInstr(block, Z80_LD_R_R, s.alloc.Alloc(8, dstHis), hiVR, nil)
}

// emitStoreHL16 stores a 16-bit value into (HL) [lo byte] and (HL+1) [hi byte]
// using any PP register pair (BC or DE) as scratch. Returns an HL TempVR
// pointing at the hi-byte address (after the internal INC HL), so the caller
// can INC HL once more to advance past the full 16-bit element.
func (s *instructionSelectorZ80) emitStoreHL16(block *cfg.BasicBlock, hlVR *cfg.TempVR, val cfg.VROperand) *cfg.TempVR {
	loVR := s.alloc.Alloc(8, []*cfg.Register{&RegC, &RegE})
	hiVR := s.alloc.Alloc(8, []*cfg.Register{&RegB, &RegD})

	switch v := val.(type) {
	case *cfg.ImmVR:
		emitInstr(block, Z80_LD_R_N, loVR, cfg.NewImmVR(v.Value&0xFF, 8), nil)
		emitInstr(block, Z80_LD_R_N, hiVR, cfg.NewImmVR((v.Value>>8)&0xFF, 8), nil)
	case *cfg.TempVR:
		srcLos, srcHis := cfg.ToPairs(v.AllowedSet())
		emitInstr(block, Z80_LD_R_R, loVR, s.alloc.Alloc(8, srcLos), nil)
		emitInstr(block, Z80_LD_R_R, hiVR, s.alloc.Alloc(8, srcHis), nil)
	default:
		panic("emitStoreHL16: unexpected VROperand type")
	}

	emitInstr(block, Z80_LD_HL_R, hlVR, loVR, nil)
	incHL := s.reg16(&RegHL)
	emitInstr(block, Z80_INC_RR, incHL, hlVR, nil)
	emitInstr(block, Z80_LD_HL_R, incHL, hiVR, nil)
	return incHL
}

// emitLD16 emits a 16-bit load of src into dst using real Z80 instructions.
// ImmVR source  → Z80_LD_RR_NN (the real "LD rr, nn" instruction).
// TempVR source → two Z80_LD_R_R on the lo/hi byte sub-registers derived
//
//	via ToPairs on each operand's AllowedSet: LD dstLo, srcLo ; LD dstHi, srcHi.
//
// dst does not appear as GetResult in either emitted instruction; its physical
// register pair is set by the two sub-LDs and enters liveness at its first use.
func (s *instructionSelectorZ80) emitLD16(block *cfg.BasicBlock, dst *cfg.TempVR, src cfg.VROperand) {
	switch v := src.(type) {
	case *cfg.ImmVR:
		emitInstr(block, Z80_LD_RR_NN, dst, v, nil)
	case *cfg.TempVR:
		srcLos, srcHis := cfg.ToPairs(v.AllowedSet())
		dstLos, dstHis := cfg.ToPairs(dst.AllowedSet())
		emitInstr(block, Z80_LD_R_R, s.alloc.Alloc(8, dstLos), s.alloc.Alloc(8, srcLos), nil)
		emitInstr(block, Z80_LD_R_R, s.alloc.Alloc(8, dstHis), s.alloc.Alloc(8, srcHis), nil)
	default:
		panic("emitLD16: unexpected VROperand type at selection time")
	}
}

// ── Allocator helpers ─────────────────────────────────────────────────────────

// reg8 allocates a TempVR constrained to a single 8-bit register.
func (s *instructionSelectorZ80) reg8(r *Register) *cfg.TempVR {
	return s.alloc.Alloc(8, []*cfg.Register{r})
}

// reg16 allocates a TempVR constrained to a single 16-bit register pair.
func (s *instructionSelectorZ80) reg16(r *Register) *cfg.TempVR {
	return s.alloc.Alloc(16, []*cfg.Register{r})
}

// any8 allocates an unconstrained 8-bit TempVR (any general-purpose 8-bit reg).
func (s *instructionSelectorZ80) any8() *cfg.TempVR {
	return s.alloc.Alloc(8, Z80Registers8)
}

// any16 allocates an unconstrained 16-bit TempVR (any general-purpose 16-bit reg).
func (s *instructionSelectorZ80) any16() *cfg.TempVR {
	return s.alloc.Alloc(16, Z80Registers16)
}
