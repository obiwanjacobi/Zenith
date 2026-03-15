package cfg

import (
	"strings"

	"zenith/compiler/lexer"
	"zenith/compiler/parser"
	"zenith/compiler/zsm"
)

// ============================================================================
// Register set descriptors (target-independent handle)
// ============================================================================

// RegisterSets carries the full register-class slices for 8-bit and 16-bit
// temporaries. They are passed in from the target so that TAC lowering can
// set AllowedSet on unconstrained TempVRs without any target-specific imports.
type RegisterSets struct {
	Regs8  []*Register // full 8-bit general-purpose class
	Regs16 []*Register // full 16-bit general-purpose class
}

// ============================================================================
// TAC lowering context
// ============================================================================

// tacLoweringCtx holds state for lowering one function's SemInstructions to TAC.
type tacLoweringCtx struct {
	cfg     *CFG
	alloc   *TempVRAllocator
	regsets RegisterSets
	symToVR map[*zsm.Symbol]VROperand // current live VR for each source symbol
}

func newTacLoweringCtx(cfg *CFG, alloc *TempVRAllocator, regsets RegisterSets) *tacLoweringCtx {
	return &tacLoweringCtx{
		cfg:     cfg,
		alloc:   alloc,
		regsets: regsets,
		symToVR: make(map[*zsm.Symbol]VROperand),
	}
}

// temp allocates an unconstrained TempVR of the given byte size.
func (ctx *tacLoweringCtx) temp(byteSize uint16) *TempVR {
	var allowed []*Register
	if byteSize <= 1 {
		allowed = ctx.regsets.Regs8
	} else {
		allowed = ctx.regsets.Regs16
	}
	return ctx.alloc.Alloc(uint8(byteSize)*8, allowed)
}

// tempFor allocates an unconstrained TempVR sized for the given type.
func (ctx *tacLoweringCtx) tempFor(t zsm.Type) *TempVR {
	return ctx.temp(t.Size())
}

// emit appends a TAC instruction to the given block.
func emit(block *BasicBlock, instr TacInstruction) {
	block.TAC = append(block.TAC, instr)
}

// ============================================================================
// Public entry point
// ============================================================================

// LowerTAC translates the SemInstructions in every BasicBlock of cfg into TAC,
// populating each block's TAC slice. The allocator and register-set descriptors
// are supplied by the caller so this function stays target-independent.
func LowerTAC(cfg *CFG, alloc *TempVRAllocator, regsets RegisterSets) error {
	ctx := newTacLoweringCtx(cfg, alloc, regsets)

	// Seed symbolToVR from function parameters: arrays/structs get StackVRs;
	// scalars get TempVRs (the selector will constrain them to calling-convention
	// registers when it sees the function entry).
	for _, param := range cfg.FunctionDecl.Parameters {
		ctx.seedParameter(param)
	}

	// Walk blocks in CFG order (they are in structural order from the builder).
	for _, block := range cfg.Blocks {
		ctx.lowerBlock(block)
	}
	return nil
}

// seedParameter establishes the initial VR for a function parameter and
// records it in cfg.ParamVRs so SelectPrologue can constrain it to the
// calling-convention register.
func (ctx *tacLoweringCtx) seedParameter(sym *zsm.Symbol) {
	var vr *TempVR
	switch sym.Type.(type) {
	case *zsm.ArrayType, *zsm.StructType:
		// Aggregate params are always passed as a pointer. Give them a 16-bit VR.
		vr = ctx.alloc.AllocNamed(sym.Name, 16, ctx.regsets.Regs16)
	default:
		vr = ctx.alloc.AllocNamed(sym.Name, uint8(sym.Type.Size()*8), ctx.regClass(sym.Type.Size()))
	}
	ctx.symToVR[sym] = vr
	ctx.cfg.ParamVRs = append(ctx.cfg.ParamVRs, vr)
}

// regClass returns the full register class for a type of the given byte size.
func (ctx *tacLoweringCtx) regClass(byteSize uint16) []*Register {
	if byteSize <= 1 {
		return ctx.regsets.Regs8
	}
	return ctx.regsets.Regs16
}

// ============================================================================
// Block lowering
// ============================================================================

func (ctx *tacLoweringCtx) lowerBlock(block *BasicBlock) {
	for _, stmt := range block.SemInstructions {
		ctx.lowerStmt(block, stmt)
	}
	// Emit an unconditional jump for blocks with a single successor that have
	// not already terminated with a branch (e.g. for.inc → for.cond, for.body
	// → for.inc). Conditional-branch blocks (for.cond, if-cond, select) emit
	// their own TacBranchCond/TacJump via lowerStmt, so we guard against
	// double-emission by checking the last TAC instruction.
	if len(block.Successors) == 1 {
		alreadyTerminated := false
		if n := len(block.TAC); n > 0 {
			switch block.TAC[n-1].(type) {
			case *TacJump, *TacBranchCond, *TacBranchIf, *TacReturn:
				alreadyTerminated = true
			}
		}
		if !alreadyTerminated {
			emit(block, &TacJump{Target: block.Successors[0]})
		}
	}
}

// ============================================================================
// Statement lowering
// ============================================================================

// nodeSourceText reconstructs a readable single-line string from the tokens
// of any parser node. Whitespace and EOL tokens are collapsed to a single
// space so the result is always one line regardless of source formatting.
// Returns an empty string if node is nil.
func nodeSourceText(node parser.ParserNode) string {
	if node == nil {
		return ""
	}
	var sb strings.Builder
	for _, tok := range node.Tokens() {
		id := tok.Id()
		if id == lexer.TokenWhitespace || id == lexer.TokenEOL {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(tok.Text())
	}
	return sb.String()
}

// commentNodeFor selects the most informative parser node to use as the
// source comment for a statement. The default is stmt.ASTNode(), but some
// statement kinds are better represented by a sub-node:
//
//   - SemFor: the for.cond block only executes the condition, so show the
//     condition expression rather than the entire for-statement header.
//   - SemExpressionStmt with nil astNode: synthetic node (e.g. the for-loop
//     increment created by the CFG builder); fall back to the wrapped
//     expression's node.
func commentNodeFor(stmt zsm.SemStatement) parser.ParserNode {
	switch s := stmt.(type) {
	case *zsm.SemFor:
		if s.Condition != nil {
			return s.Condition.ASTNode()
		}
	case *zsm.SemExpressionStmt:
		if s.ASTNode() == nil && s.Expression != nil {
			return s.Expression.ASTNode()
		}
	}
	return stmt.ASTNode()
}

func (ctx *tacLoweringCtx) lowerStmt(block *BasicBlock, stmt zsm.SemStatement) {
	emit(block, &TacComment{Text: nodeSourceText(commentNodeFor(stmt))})

	switch s := stmt.(type) {
	case *zsm.SemVariableDecl:
		ctx.lowerVarDecl(block, s)

	case *zsm.SemAssignment:
		ctx.lowerAssignment(block, s)

	case *zsm.SemExpressionStmt:
		// Evaluate for side effects; discard the result VR.
		ctx.lowerExpr(block, s.Expression)

	case *zsm.SemReturn:
		ctx.lowerReturn(block, s)

	case *zsm.SemIf:
		// The CFG builder stores *SemIf in the condition block. The then/else
		// successors are already wired in the CFG. Emit a branch on the condition.
		ctx.lowerIfBranch(block, s)

	case *zsm.SemElsif:
		// Elsif condition block: same treatment as SemIf.
		ctx.lowerElsifBranch(block, s)

	case *zsm.SemFor:
		// For-condition block: emit conditional branch.
		ctx.lowerForBranch(block, s)

	case *zsm.SemSelect:
		// Select: emit a chain of equality comparisons + branches.
		ctx.lowerSelectBranch(block, s)
	}
}

// lowerVarDecl handles "var x T" and "var x T := expr".
func (ctx *tacLoweringCtx) lowerVarDecl(block *BasicBlock, decl *zsm.SemVariableDecl) {
	sym := decl.Symbol

	switch sym.Type.(type) {
	case *zsm.ArrayType, *zsm.StructType:
		// Arrays and structs live on the stack. Allocate a frame slot and compute
		// a STACK_ADDR into a 16-bit TempVR that becomes the symbol's VR.
		frameOffset := ctx.cfg.StackFrame.AddSlot(sym, sym.Type.Size())
		dst := ctx.alloc.AllocNamed(sym.Name, 16, ctx.regsets.Regs16)
		ctx.symToVR[sym] = dst
		emit(block, &TacStackAddr{Dst: dst, Offset: frameOffset})

		// Handle array literal initialiser: emit INIT_SEQ.
		if decl.Initializer != nil {
			ctx.lowerArrayInit(block, dst, decl.Initializer)
		}

	default:
		// Scalar: lower the initialiser (if any) and bind the symbol.
		if decl.Initializer != nil {
			src := ctx.lowerExpr(block, decl.Initializer)
			// Introduce a named copy so the variable has its own VR.
			dst := ctx.alloc.AllocNamed(sym.Name, uint8(sym.Type.Size()*8), ctx.regClass(sym.Type.Size()))
			emit(block, &TacCopy{Dst: dst, Src: src, Size: uint8(sym.Type.Size() * 8)})
			ctx.symToVR[sym] = dst
		} else {
			// No initialiser: allocate a VR; its value is undefined until assigned.
			dst := ctx.alloc.AllocNamed(sym.Name, uint8(sym.Type.Size()*8), ctx.regClass(sym.Type.Size()))
			ctx.symToVR[sym] = dst
		}
	}
}

// lowerArrayInit emits an INIT_SEQ for a SemArrayInitializer, or falls back
// to a single TacStore for any other initialiser expression.
// base is the TempVR holding the base address of the array's stack slot.
func (ctx *tacLoweringCtx) lowerArrayInit(block *BasicBlock, base *TempVR, init zsm.SemExpression) {
	arrInit, ok := init.(*zsm.SemArrayInitializer)
	if !ok {
		// Non-literal initialiser (e.g. a function call returning a pointer):
		// emit a single store of the evaluated result at offset 0.
		src := ctx.lowerExpr(block, init)
		arrType := init.Type().(*zsm.ArrayType)
		emit(block, &TacStore{Base: base, Offset: 0, Value: src, Size: uint8(arrType.ElementType().Size())})
		return
	}

	// Lower each element expression to a VROperand.
	values := make([]VROperand, len(arrInit.Elements))
	for i, elem := range arrInit.Elements {
		values[i] = ctx.lowerExpr(block, elem)
	}

	arrType := arrInit.TypeInfo.(*zsm.ArrayType)
	emit(block, &TacInitSeq{
		Base:     base,
		ElemSize: uint8(arrType.ElementType().Size()),
		Values:   values,
	})
}

// lowerAssignment handles "x := expr" and "arr[i] := expr".
func (ctx *tacLoweringCtx) lowerAssignment(block *BasicBlock, assign *zsm.SemAssignment) {
	if assign.TargetIndex != nil {
		// Indexed assignment: arr[i] := value
		arrayVR := ctx.lookupSym(block, assign.Target)
		indexVR := ctx.lowerExpr(block, assign.TargetIndex)
		valueVR := ctx.lowerExpr(block, assign.Value)
		elemSize := assign.Target.Type.(*zsm.ArrayType).ElementType().Size()
		emit(block, &TacStoreIndexed{
			Base:     arrayVR,
			Index:    indexVR,
			Value:    valueVR,
			ElemSize: uint8(elemSize),
			Size:     uint8(elemSize),
		})
		return
	}

	// Plain assignment or struct-field assignment.
	sym := assign.Target
	switch sym.Type.(type) {
	case *zsm.StructType:
		// Struct assignment is a struct-pointer copy (structs never in a VR).
		// Lower the RHS to a pointer VR and store it as the new symbol binding.
		rhs := ctx.lowerExpr(block, assign.Value)
		dst := ctx.alloc.AllocNamed(sym.Name, 16, ctx.regsets.Regs16)
		emit(block, &TacCopy{Dst: dst, Src: rhs, Size: 16})
		ctx.symToVR[sym] = dst

	default:
		// Scalar assignment: new VR for SSA, copy from RHS.
		rhs := ctx.lowerExpr(block, assign.Value)
		dst := ctx.alloc.AllocNamed(sym.Name, uint8(sym.Type.Size()*8), ctx.regClass(sym.Type.Size()))
		emit(block, &TacCopy{Dst: dst, Src: rhs, Size: uint8(sym.Type.Size() * 8)})
		ctx.symToVR[sym] = dst
	}
}

// lowerReturn emits a TacReturn.
func (ctx *tacLoweringCtx) lowerReturn(block *BasicBlock, ret *zsm.SemReturn) {
	if ret.Value != nil {
		val := ctx.lowerExpr(block, ret.Value)
		emit(block, &TacReturn{Value: val})
	} else {
		emit(block, &TacReturn{})
	}
}

// ============================================================================
// Control-flow branch emission (for sentinel SemIf/SemFor/SemSelect nodes)
// ============================================================================

// lowerIfBranch emits a conditional branch at the end of an if-condition block.
// The CFG already has the then/else successors wired; we just emit the TAC branch
// pointing at the first and second successor respectively.
func (ctx *tacLoweringCtx) lowerIfBranch(block *BasicBlock, s *zsm.SemIf) {
	// successors[0] = then, successors[1] = elsif-cond or else or merge
	if len(block.Successors) < 2 {
		return // degenerate — should not happen
	}
	thenBlock := block.Successors[0]
	elseBlock := block.Successors[1]
	ctx.emitCondBranch(block, s.Condition, thenBlock, elseBlock)
}

// lowerElsifBranch emits a conditional branch for an elsif-condition block.
func (ctx *tacLoweringCtx) lowerElsifBranch(block *BasicBlock, s *zsm.SemElsif) {
	if len(block.Successors) < 2 {
		return
	}
	thenBlock := block.Successors[0]
	elseBlock := block.Successors[1]
	ctx.emitCondBranch(block, s.Condition, thenBlock, elseBlock)
}

// lowerForBranch emits a conditional branch at the for-condition block.
// successors[0] = body, successors[1] = exit.
func (ctx *tacLoweringCtx) lowerForBranch(block *BasicBlock, s *zsm.SemFor) {
	if s.Condition == nil {
		// Unconditional loop: jump to body (successors[0]).
		if len(block.Successors) > 0 {
			emit(block, &TacJump{Target: block.Successors[0]})
		}
		return
	}
	if len(block.Successors) < 2 {
		return
	}
	bodyBlock := block.Successors[0]
	exitBlock := block.Successors[1]
	ctx.emitCondBranch(block, s.Condition, bodyBlock, exitBlock)
}

// lowerSelectBranch emits equality comparisons + branches for a select statement.
// The CFG wires: successors[0..n-1] = case blocks, successors[n] = else/merge.
func (ctx *tacLoweringCtx) lowerSelectBranch(block *BasicBlock, s *zsm.SemSelect) {
	exprVR := ctx.lowerExpr(block, s.Expression)

	succIdx := 0
	for _, cas := range s.Cases {
		if succIdx >= len(block.Successors) {
			break
		}
		caseBlock := block.Successors[succIdx]
		succIdx++

		// The "next" target is the following case (or else/merge).
		var nextBlock *BasicBlock
		if succIdx < len(block.Successors) {
			nextBlock = block.Successors[succIdx]
		} else if len(block.Successors) > 0 {
			nextBlock = block.Successors[len(block.Successors)-1]
		}
		if nextBlock == nil {
			nextBlock = caseBlock // degenerate
		}

		caseValVR := ctx.lowerExpr(block, cas.Value)
		opSize := uint8(s.Expression.Type().Size() * 8)
		emit(block, &TacBranchCond{
			Op:    TacCmpEqual,
			Left:  exprVR,
			Right: caseValVR,
			Size:  opSize,
			Then:  caseBlock,
			Else:  nextBlock,
		})
	}

	// Unconditional jump to else/merge (last successor) if no case matches.
	if len(block.Successors) > 0 {
		last := block.Successors[len(block.Successors)-1]
		emit(block, &TacJump{Target: last})
	}
}

// emitCondBranch emits either a fused TacBranchCond (BranchMode) for a comparison
// expression, or a TacBranchIf for a pre-computed boolean symbol ref.
func (ctx *tacLoweringCtx) emitCondBranch(block *BasicBlock, cond zsm.SemExpression, then, els *BasicBlock) {
	// BranchMode: binary comparison fed directly to branch.
	if binOp, ok := cond.(*zsm.SemBinaryOp); ok {
		if cmpOp, isCmp := binaryToCmpOp(binOp.Op); isCmp {
			left := ctx.lowerExpr(block, binOp.Left)
			right := ctx.lowerExpr(block, binOp.Right)
			opSize := uint8(binOp.Left.Type().Size() * 8)
			emit(block, &TacBranchCond{
				Op:    cmpOp,
				Left:  left,
				Right: right,
				Size:  opSize,
				Then:  then,
				Else:  els,
			})
			return
		}
	}

	// Logical-not: invert the branch directions.
	if unOp, ok := cond.(*zsm.SemUnaryOp); ok && unOp.Op == zsm.OpLogicalNot {
		ctx.emitCondBranch(block, unOp.Operand, els, then) // swap then/else
		return
	}

	// Logical-and: short-circuit — must be true on both sides.
	if binOp, ok := cond.(*zsm.SemBinaryOp); ok && binOp.Op == zsm.OpLogicalAnd {
		// Evaluate left; if false, jump to else immediately.
		// If true, evaluate right; branch on right.
		midBlock := &BasicBlock{ID: -1} // synthetic; lowering only, no machine instrs
		ctx.emitCondBranch(block, binOp.Left, midBlock, els)
		ctx.emitCondBranch(midBlock, binOp.Right, then, els)
		return
	}

	// Logical-or: short-circuit — true if either side is true.
	if binOp, ok := cond.(*zsm.SemBinaryOp); ok && binOp.Op == zsm.OpLogicalOr {
		midBlock := &BasicBlock{ID: -1}
		ctx.emitCondBranch(block, binOp.Left, then, midBlock)
		ctx.emitCondBranch(midBlock, binOp.Right, then, els)
		return
	}

	// Fallback: materialise the boolean into a VR and branch on it.
	condVR := ctx.lowerExpr(block, cond)
	emit(block, &TacBranchIf{Cond: condVR, Then: then, Else: els})
}

// ============================================================================
// Expression lowering
// ============================================================================

// lowerExpr lowers a SemExpression to a VROperand, appending TAC to block.
func (ctx *tacLoweringCtx) lowerExpr(block *BasicBlock, expr zsm.SemExpression) VROperand {
	switch e := expr.(type) {
	case *zsm.SemConstant:
		return ctx.lowerConstant(e)

	case *zsm.SemSymbolRef:
		return ctx.lookupSym(block, e.Symbol)

	case *zsm.SemBinaryOp:
		return ctx.lowerBinaryOp(block, e)

	case *zsm.SemUnaryOp:
		return ctx.lowerUnaryOp(block, e)

	case *zsm.SemFunctionCall:
		return ctx.lowerFunctionCall(block, e)

	case *zsm.SemSubscript:
		return ctx.lowerSubscript(block, e)

	case *zsm.SemMemberAccess:
		return ctx.lowerMemberAccess(block, e)

	case *zsm.SemArrayInitializer:
		return ctx.lowerArrayInitExpr(block, e)
	}

	// Unknown expression — return a zero-sized ImmVR as a safe fallback so
	// compilation can continue and report errors elsewhere.
	return &ImmVR{Value: 0, size: 8}
}

// lowerConstant converts a SemConstant to an ImmVR.
func (ctx *tacLoweringCtx) lowerConstant(c *zsm.SemConstant) VROperand {
	var val int32
	switch v := c.Value.(type) {
	case int:
		val = int32(v)
	case bool:
		if v {
			val = 1
		}
	case string:
		// String literals are not representable as ImmVR; emit a zero for now.
		// The frontend should lower string literals to array constants before TAC.
		val = 0
	}
	return &ImmVR{Value: val, size: uint8(c.TypeInfo.Size() * 8)}
}

// lookupSym returns the current VR for a symbol. For stack-backed symbols that
// have been declared (arrays/structs), returns their StackVR pointer VR.
// For scalars that have not yet been seen (forward reference — should not occur
// after semantic analysis), allocates a fresh VR as a recovery measure.
func (ctx *tacLoweringCtx) lookupSym(block *BasicBlock, sym *zsm.Symbol) VROperand {
	if vr, ok := ctx.symToVR[sym]; ok {
		return vr
	}
	// Symbol not yet seen — allocate a fresh (unnamed) VR.
	vr := ctx.alloc.AllocNamed(sym.Name, uint8(sym.Type.Size()*8), ctx.regClass(sym.Type.Size()))
	ctx.symToVR[sym] = vr
	return vr
}

// lowerBinaryOp lowers a SemBinaryOp expression.
// Comparison operators in value-context produce a TacCompare (bit result).
// Arithmetic/bitwise ops produce a TacBinOp.
// Logical ops short-circuit; in value-context they materialise as 0/1.
func (ctx *tacLoweringCtx) lowerBinaryOp(block *BasicBlock, e *zsm.SemBinaryOp) VROperand {
	// Comparison operators — materialise result as a bit VR (value-context).
	if cmpOp, isCmp := binaryToCmpOp(e.Op); isCmp {
		left := ctx.lowerExpr(block, e.Left)
		right := ctx.lowerExpr(block, e.Right)
		dst := ctx.temp(1)
		opSize := uint8(e.Left.Type().Size() * 8)
		emit(block, &TacCompare{
			Op:    cmpOp,
			Dst:   dst,
			Left:  left,
			Right: right,
			Size:  opSize,
		})
		return dst
	}

	// Logical operators — short-circuit evaluation in value-mode.
	// We materialise the result into a bit TempVR by emitting a branch sequence
	// and joining with copies. This keeps the backend simple (it never sees
	// OpLogicalAnd/Or as TAC ops).
	if e.Op == zsm.OpLogicalAnd || e.Op == zsm.OpLogicalOr {
		return ctx.lowerLogicalOp(block, e)
	}

	// Arithmetic / bitwise operators.
	tacOp, ok := binaryToTacOp(e.Op)
	if !ok {
		return &ImmVR{Value: 0, size: 8}
	}

	left := ctx.lowerExpr(block, e.Left)
	right := ctx.lowerExpr(block, e.Right)

	// Result size: multiply doubles the width; all others use the result type.
	resultSize := e.TypeInfo.Size()
	dst := ctx.temp(resultSize)
	opSize := uint8(e.Left.Type().Size() * 8)
	emit(block, &TacBinOp{
		Op:    tacOp,
		Dst:   dst,
		Left:  left,
		Right: right,
		Size:  opSize,
	})
	return dst
}

// lowerLogicalOp materialises a short-circuit logical-and/or as a bit VR.
// It emits a minimal branch-and-copy sequence into the current block list.
// Note: this produces inline TAC within the same block for value-mode use;
// in branch-mode the caller uses emitCondBranch instead, which is preferred.
func (ctx *tacLoweringCtx) lowerLogicalOp(block *BasicBlock, e *zsm.SemBinaryOp) VROperand {
	// Materialise left side.
	leftVR := ctx.lowerExpr(block, e.Left)

	// Allocate the result bit VR.
	result := ctx.alloc.Alloc(8, ctx.regsets.Regs8)

	// Materialise right side.
	rightVR := ctx.lowerExpr(block, e.Right)

	// Emit a compare-and-select pattern:
	// For AND: result = (leftVR != 0) & (rightVR != 0)
	// For OR:  result = (leftVR != 0) | (rightVR != 0)
	//
	// In practice the instruction selector will use flags for these;
	// we emit TacCompare nodes that the selector maps to TEST/AND/OR sequences.
	zero := &ImmVR{Value: 0, size: 8}
	leftBit := ctx.alloc.Alloc(8, ctx.regsets.Regs8)
	rightBit := ctx.alloc.Alloc(8, ctx.regsets.Regs8)
	emit(block, &TacCompare{Op: TacCmpNotEqual, Dst: leftBit, Left: leftVR, Right: zero, Size: uint8(e.Left.Type().Size() * 8)})
	emit(block, &TacCompare{Op: TacCmpNotEqual, Dst: rightBit, Left: rightVR, Right: zero, Size: uint8(e.Right.Type().Size() * 8)})

	var tacOp TacOp
	if e.Op == zsm.OpLogicalAnd {
		tacOp = TacAnd
	} else {
		tacOp = TacOr
	}
	emit(block, &TacBinOp{Op: tacOp, Dst: result, Left: leftBit, Right: rightBit, Size: 8})
	return result
}

// lowerUnaryOp lowers a SemUnaryOp.
func (ctx *tacLoweringCtx) lowerUnaryOp(block *BasicBlock, e *zsm.SemUnaryOp) VROperand {
	operand := ctx.lowerExpr(block, e.Operand)
	dst := ctx.tempFor(e.TypeInfo)
	byteSize := e.TypeInfo.Size()

	var tacOp TacUnaryOp
	switch e.Op {
	case zsm.OpNegate:
		tacOp = TacNegate
	case zsm.OpBitwiseNot:
		tacOp = TacBitwiseNot
	case zsm.OpIncrement:
		tacOp = TacIncrement
	case zsm.OpDecrement:
		tacOp = TacDecrement
	case zsm.OpLogicalNot:
		// Logical-not: compare operand != 0, then flip with BNOT.
		// Emit: tmp = (operand != 0); dst = ~tmp (which gives 0 or 1 for bit).
		// Simpler: CMP != 0 already gives a boolean; logical-not inverts it.
		// Emit a CMP EQ 0, which directly gives the logical-not bit.
		zero := &ImmVR{Value: 0, size: uint8(e.Operand.Type().Size() * 8)}
		cmpDst := ctx.alloc.Alloc(8, ctx.regsets.Regs8)
		emit(block, &TacCompare{Op: TacCmpEqual, Dst: cmpDst, Left: operand, Right: zero, Size: uint8(e.Operand.Type().Size() * 8)})
		return cmpDst
	default:
		// Unknown op — safe fallback.
		return operand
	}

	emit(block, &TacUnary{Op: tacOp, Dst: dst, Operand: operand, Size: uint8(byteSize * 8)})
	return dst
}

// lowerFunctionCall lowers a SemFunctionCall.
func (ctx *tacLoweringCtx) lowerFunctionCall(block *BasicBlock, e *zsm.SemFunctionCall) VROperand {
	args := make([]VROperand, len(e.Arguments))
	for i, arg := range e.Arguments {
		args[i] = ctx.lowerExpr(block, arg)
	}

	var dst *TempVR
	var retSize uint8
	if e.TypeInfo != nil && e.TypeInfo.Size() > 0 {
		retSize = uint8(e.TypeInfo.Size() * 8)
		dst = ctx.alloc.Alloc(retSize, ctx.regClass(e.TypeInfo.Size()))
	}

	emit(block, &TacCall{
		Dst:     dst,
		Fn:      e.Function.Name,
		Args:    args,
		RetSize: retSize,
	})

	if dst != nil {
		return dst
	}
	return &ImmVR{Value: 0, size: 0}
}

// lowerSubscript lowers arr[index] → TacLoadIndexed.
func (ctx *tacLoweringCtx) lowerSubscript(block *BasicBlock, e *zsm.SemSubscript) VROperand {
	arrayVR := ctx.lowerExpr(block, e.Array)
	indexVR := ctx.lowerExpr(block, e.Index)

	// Element type size.
	arrType, ok := e.Array.Type().(*zsm.ArrayType)
	if !ok {
		return &ImmVR{Value: 0, size: 8}
	}
	elemSize := uint8(arrType.ElementType().Size())
	dst := ctx.temp(uint16(elemSize))

	emit(block, &TacLoadIndexed{
		Dst:      dst,
		Base:     arrayVR,
		Index:    indexVR,
		ElemSize: elemSize,
		Size:     elemSize,
	})
	return dst
}

// lowerArrayInitExpr handles a SemArrayInitializer used as a sub-expression.
// It allocates an anonymous stack slot, emits TacStackAddr + TacInitSeq, and
// returns the base-address TempVR so the result can be used as a pointer arg.
func (ctx *tacLoweringCtx) lowerArrayInitExpr(block *BasicBlock, e *zsm.SemArrayInitializer) VROperand {
	arrType := e.TypeInfo.(*zsm.ArrayType)

	// Allocate an anonymous stack slot for the array data.
	anonSym := &zsm.Symbol{Name: ctx.alloc.NextAnonName(), Type: e.TypeInfo}
	frameOffset := ctx.cfg.StackFrame.AddSlot(anonSym, e.TypeInfo.Size())

	base := ctx.alloc.Alloc(16, ctx.regsets.Regs16)
	emit(block, &TacStackAddr{Dst: base, Offset: frameOffset})

	values := make([]VROperand, len(e.Elements))
	for i, elem := range e.Elements {
		values[i] = ctx.lowerExpr(block, elem)
	}
	emit(block, &TacInitSeq{
		Base:     base,
		ElemSize: uint8(arrType.ElementType().Size()),
		Values:   values,
	})
	return base
}

// lowerMemberAccess lowers s.field → TacLoad with the field's byte offset.
func (ctx *tacLoweringCtx) lowerMemberAccess(block *BasicBlock, e *zsm.SemMemberAccess) VROperand {
	// Object is stored as *SemExpression pointer in the semantic model.
	var objVR VROperand
	if e.Object != nil {
		objVR = ctx.lowerExpr(block, *e.Object)
	} else {
		return &ImmVR{Value: 0, size: 8}
	}

	field := e.Field
	dst := ctx.temp(field.Type.Size())
	emit(block, &TacLoad{
		Dst:    dst,
		Base:   objVR,
		Offset: field.Offset,
		Size:   uint8(field.Type.Size() * 8),
	})
	return dst
}

// ============================================================================
// Operator mapping helpers
// ============================================================================

// binaryToCmpOp maps a BinaryOperator to a TacCmpOp.
// Returns (op, true) if the operator is a comparison; (0, false) otherwise.
func binaryToCmpOp(op zsm.BinaryOperator) (TacCmpOp, bool) {
	switch op {
	case zsm.OpEqual:
		return TacCmpEqual, true
	case zsm.OpNotEqual:
		return TacCmpNotEqual, true
	case zsm.OpLessThan:
		return TacCmpLess, true
	case zsm.OpLessEqual:
		return TacCmpLessEq, true
	case zsm.OpGreaterThan:
		return TacCmpGreater, true
	case zsm.OpGreaterEqual:
		return TacCmpGreaterEq, true
	}
	return 0, false
}

// binaryToTacOp maps an arithmetic/bitwise BinaryOperator to a TacOp.
func binaryToTacOp(op zsm.BinaryOperator) (TacOp, bool) {
	switch op {
	case zsm.OpAdd:
		return TacAdd, true
	case zsm.OpSubtract:
		return TacSub, true
	case zsm.OpMultiply:
		return TacMul, true
	case zsm.OpDivide:
		return TacDiv, true
	case zsm.OpBitwiseAnd:
		return TacAnd, true
	case zsm.OpBitwiseOr:
		return TacOr, true
	case zsm.OpBitwiseXor:
		return TacXor, true
	}
	return 0, false
}
