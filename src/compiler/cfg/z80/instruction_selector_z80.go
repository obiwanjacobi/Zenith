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
		// LD A, (HL)
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_LD_R_HL, aVR, hlVR, nil)

		// LD dst, A  (peephole removes if dst also ends up in A)
		dst := s.alloc.Alloc(8, Z80Registers8)
		t.Dst.AllowedSet = dst.AllowedSet
		emitInstr(block, Z80_LD_R_R, t.Dst, aVR, nil)
	} else {
		// 16-bit: load low byte then high byte.
		// LD E, (HL) ; INC HL ; LD D, (HL)  → result in DE
		eVR := s.reg8(&RegE)
		emitInstr(block, Z80_LD_R_HL, eVR, hlVR, nil)

		incHL := s.reg16(&RegHL)
		emitInstr(block, Z80_INC_RR, incHL, hlVR, nil)

		dVR := s.reg8(&RegD)
		emitInstr(block, Z80_LD_R_HL, dVR, incHL, nil)

		deVR := s.reg16(&RegDE)
		emitInstr(block, Z80_LD_R_R, t.Dst, deVR, nil) // pseudo: move DE→dst
		t.Dst.AllowedSet = []*cfg.Register{&RegDE}
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
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_LD_R_HL, aVR, addHL, nil)
		t.Dst.AllowedSet = Z80Registers8
		emitInstr(block, Z80_LD_R_R, t.Dst, aVR, nil)
	} else {
		// 16-bit: LD E,(HL); INC HL; LD D,(HL) → DE
		eVR := s.reg8(&RegE)
		emitInstr(block, Z80_LD_R_HL, eVR, addHL, nil)
		incHL := s.reg16(&RegHL)
		emitInstr(block, Z80_INC_RR, incHL, addHL, nil)
		dVR := s.reg8(&RegD)
		emitInstr(block, Z80_LD_R_HL, dVR, incHL, nil)
		t.Dst.AllowedSet = []*cfg.Register{&RegDE}
		deResult := s.reg16(&RegDE)
		emitInstr(block, Z80_LD_R_R, t.Dst, deResult, nil)
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
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_LD_R_R, aVR, t.Value, nil)
		emitInstr(block, Z80_LD_HL_R, hlVR, aVR, nil)
	} else {
		// 16-bit: store low byte then high byte.
		// LD (HL), E ; INC HL ; LD (HL), D
		// Require value in DE.
		deVR := s.reg16(&RegDE)
		emitInstr(block, Z80_LD_R_R, deVR, t.Value, nil)
		eVR := s.reg8(&RegE)
		emitInstr(block, Z80_LD_HL_R, hlVR, eVR, nil)
		incHL := s.reg16(&RegHL)
		emitInstr(block, Z80_INC_RR, incHL, hlVR, nil)
		dVR := s.reg8(&RegD)
		emitInstr(block, Z80_LD_HL_R, incHL, dVR, nil)
	}
}

// SelectStoreIndexed: *(Base + Index*ElemSize) = Value
func (s *instructionSelectorZ80) SelectStoreIndexed(block *cfg.BasicBlock, t *cfg.TacStoreIndexed) {
	hlVR := s.moveToHL(block, t.Base)
	deVR := s.scaleIndex(block, t.Index, t.ElemSize)

	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlVR, deVR)

	if t.Size == 8 {
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_LD_R_R, aVR, t.Value, nil)
		emitInstr(block, Z80_LD_HL_R, addHL, aVR, nil)
	} else {
		deResult := s.reg16(&RegDE)
		emitInstr(block, Z80_LD_R_R, deResult, t.Value, nil)
		eVR := s.reg8(&RegE)
		emitInstr(block, Z80_LD_HL_R, addHL, eVR, nil)
		incHL := s.reg16(&RegHL)
		emitInstr(block, Z80_INC_RR, incHL, addHL, nil)
		dVR := s.reg8(&RegD)
		emitInstr(block, Z80_LD_HL_R, incHL, dVR, nil)
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
			// LD A, val ; LD (HL), A
			aVR := s.reg8(&RegA)
			emitInstr(block, Z80_LD_R_R, aVR, val, nil)
			emitInstr(block, Z80_LD_HL_R, hlVR, aVR, nil)
		} else {
			// 16-bit element: store L then H.
			// Require val in DE; store E then D.
			deVR := s.reg16(&RegDE)
			emitInstr(block, Z80_LD_R_R, deVR, val, nil)
			eVR := s.reg8(&RegE)
			emitInstr(block, Z80_LD_HL_R, hlVR, eVR, nil)
			incHL := s.reg16(&RegHL)
			emitInstr(block, Z80_INC_RR, incHL, hlVR, nil)
			hlVR = incHL
			dVR := s.reg8(&RegD)
			emitInstr(block, Z80_LD_HL_R, hlVR, dVR, nil)
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
	t.Dst.AllowedSet = []*cfg.Register{&RegHL}
	emitInstr(block, Z80_LD_R_R, t.Dst, addHL, nil)
}

// ── Copy / move ───────────────────────────────────────────────────────────────

// SelectCopy: Dst = Src
//
// Z80: LD dst, src  (8 or 16-bit; peephole removes if same register)
func (s *instructionSelectorZ80) SelectCopy(block *cfg.BasicBlock, t *cfg.TacCopy) {
	if t.Size == 8 {
		emitInstr(block, Z80_LD_R_R, t.Dst, t.Src, nil)
	} else {
		emitInstr(block, Z80_LD_R_R, t.Dst, t.Src, nil)
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

	t.Dst.AllowedSet = []*cfg.Register{&RegA}
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

	t.Dst.AllowedSet = []*cfg.Register{&RegA}
	emitInstr(block, Z80_LD_R_R, t.Dst, subVR, nil)
}

// selectAdd16: HL = left + right  →  Dst
//
//	LD HL, left
//	LD DE, right
//	ADD HL, DE
//	LD dst, HL
func (s *instructionSelectorZ80) selectAdd16(block *cfg.BasicBlock, t *cfg.TacBinOp) {
	hlVR := s.reg16(&RegHL)
	emitInstr(block, Z80_LD_RR_NN, hlVR, t.Left, nil)

	deVR := s.reg16(&RegDE)
	emitInstr(block, Z80_LD_RR_NN, deVR, t.Right, nil)

	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlVR, deVR)

	t.Dst.AllowedSet = []*cfg.Register{&RegHL}
	emitInstr(block, Z80_LD_R_R, t.Dst, addHL, nil)
}

// selectSub16: HL = left - right  →  Dst
//
//	LD HL, left
//	LD DE, right
//	AND A          ; clear carry flag
//	SBC HL, DE
//	LD dst, HL
func (s *instructionSelectorZ80) selectSub16(block *cfg.BasicBlock, t *cfg.TacBinOp) {
	hlVR := s.reg16(&RegHL)
	emitInstr(block, Z80_LD_RR_NN, hlVR, t.Left, nil)

	deVR := s.reg16(&RegDE)
	emitInstr(block, Z80_LD_RR_NN, deVR, t.Right, nil)

	// AND A clears the carry flag without altering HL or DE.
	aVR := s.reg8(&RegA)
	emitInstr(block, Z80_AND_R, aVR, aVR, nil)

	sbcHL := s.reg16(&RegHL)
	emitInstr(block, Z80_SBC_HL_RR, sbcHL, hlVR, deVR)

	t.Dst.AllowedSet = []*cfg.Register{&RegHL}
	emitInstr(block, Z80_LD_R_R, t.Dst, sbcHL, nil)
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
		t.Dst.AllowedSet = []*cfg.Register{&RegDE}
		emitInstr(block, Z80_LD_R_R, t.Dst, resultVR, nil)
	} else {
		// div8 quotient in A
		resultVR := s.reg8(&RegA)
		emitCall(block, helper, resultVR)
		t.Dst.AllowedSet = []*cfg.Register{&RegA}
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
	emitInstr(block, Z80_LD_RR_NN, deVR, t.Left, nil)

	bcVR := s.reg16(&RegBC)
	emitInstr(block, Z80_LD_RR_NN, bcVR, t.Right, nil)

	resultVR := s.reg16(&RegDE)
	emitCall(block, helper, resultVR)

	t.Dst.AllowedSet = []*cfg.Register{&RegDE}
	emitInstr(block, Z80_LD_R_R, t.Dst, resultVR, nil)
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
		emitInstr(block, Z80_LD_RR_NN, hlVR, left, nil)
		deVR := s.reg16(&RegDE)
		emitInstr(block, Z80_LD_RR_NN, deVR, right, nil)
		// AND A clears carry before SBC HL, DE.
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_AND_R, aVR, aVR, nil)
		sbcHL := s.reg16(&RegHL)
		emitInstr(block, Z80_SBC_HL_RR, sbcHL, hlVR, deVR)
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
		emitInstr(block, Z80_LD_RR_NN, hlVR, left, nil)
		deVR := s.reg16(&RegDE)
		emitInstr(block, Z80_LD_RR_NN, deVR, right, nil)
		aVR := s.reg8(&RegA)
		emitInstr(block, Z80_AND_R, aVR, aVR, nil) // clear carry
		sbcHL := s.reg16(&RegHL)
		emitInstr(block, Z80_SBC_HL_RR, sbcHL, hlVR, deVR)
	}

	// Materialise: LD A, 0 ; JR inverse_cc, +1 ; INC A
	zeroA := s.reg8(&RegA)
	emitInstr(block, Z80_LD_R_N, zeroA, cfg.NewImmVR(0, 8), nil)
	emitRelJump(block, invertCond(cc), 1) // skip INC A if condition false
	oneA := s.reg8(&RegA)
	emitInstr(block, Z80_INC_R, oneA, zeroA, nil)

	t.Dst.AllowedSet = []*cfg.Register{&RegA}
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

	dst.AllowedSet = []*cfg.Register{&RegA}
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
	emitInstr(block, Z80_LD_RR_NN, hlVR, left, nil)
	deVR := s.reg16(&RegDE)
	emitInstr(block, Z80_LD_RR_NN, deVR, right, nil)

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

	dst.AllowedSet = []*cfg.Register{&RegHL}
	emitInstr(block, Z80_LD_R_R, dst, hlVR, nil)
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

// moveToHL emits a LD HL, src and returns the constrained HL TempVR.
func (s *instructionSelectorZ80) moveToHL(block *cfg.BasicBlock, src cfg.VROperand) *cfg.TempVR {
	hlVR := s.reg16(&RegHL)
	emitInstr(block, Z80_LD_RR_NN, hlVR, src, nil)
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

	// LD DE, offset
	offsetImm := cfg.NewImmVR(int32(offset), 16)
	deVR := s.reg16(&RegDE)
	emitInstr(block, Z80_LD_RR_NN, deVR, offsetImm, nil)

	// ADD HL, DE
	addHL := s.reg16(&RegHL)
	emitInstr(block, Z80_ADD_HL_RR, addHL, hlVR, deVR)
	return addHL
}

// scaleIndex returns a TempVR in DE holding Index scaled by ElemSize.
// ElemSize 1: DE = Index (zero-extended from 8-bit if needed).
// ElemSize 2: DE = Index * 2 (via ADD HL,HL on a scratch HL, result moved to DE).
func (s *instructionSelectorZ80) scaleIndex(block *cfg.BasicBlock, index cfg.VROperand, elemSize uint8) *cfg.TempVR {
	deVR := s.reg16(&RegDE)

	if index.Size() == 8 {
		// Zero-extend 8-bit index into DE: LD E, index; LD D, 0
		eVR := s.reg8(&RegE)
		emitInstr(block, Z80_LD_R_R, eVR, index, nil)
		dVR := s.reg8(&RegD)
		zeroImm := cfg.NewImmVR(0, 8)
		emitInstr(block, Z80_LD_R_N, dVR, zeroImm, nil)
	} else {
		// 16-bit index already fits in DE.
		emitInstr(block, Z80_LD_R_R, deVR, index, nil)
	}

	if elemSize == 2 {
		// DE *= 2: move DE to HL, ADD HL,HL, move back to DE.
		hlVR := s.reg16(&RegHL)
		emitInstr(block, Z80_LD_R_R, hlVR, deVR, nil)
		emitInstr(block, Z80_ADD_HL_RR, hlVR, hlVR, nil)
		emitInstr(block, Z80_LD_R_R, deVR, hlVR, nil)
	}

	return deVR
}
