package cfg

import (
	"fmt"
	"zenith/compiler/zsm"
)

// InstructionSelectionContext manages the state during instruction selection
type InstructionSelectionContext struct {
	selector          InstructionSelector
	vrAlloc           *VirtualRegisterAllocator
	callingConvention CallingConvention

	// Maps zsm symbols to their VirtualRegisters
	symbolToVReg map[*zsm.Symbol]*VirtualRegister

	// Maps expression nodes to their result VirtualRegisters (for reuse)
	exprToVReg map[zsm.SemExpression]*VirtualRegister

	// Current function being processed
	currentFunction *zsm.SemFunctionDecl

	// Current CFG being processed
	currentCFG *CFG

	// Current basic block being processed
	currentBlock *BasicBlock
}

// NewInstructionSelectionContext creates a new context for instruction selection
func NewInstructionSelectionContext(selector InstructionSelector, vrAlloc *VirtualRegisterAllocator) *InstructionSelectionContext {
	return &InstructionSelectionContext{
		selector:          selector,
		vrAlloc:           vrAlloc,
		callingConvention: selector.GetCallingConvention(),
		symbolToVReg:      make(map[*zsm.Symbol]*VirtualRegister),
		exprToVReg:        make(map[zsm.SemExpression]*VirtualRegister),
	}
}

// SelectInstructions generates machine instructions for pre-built CFGs
// Takes a slice of CFGs and populates their MachineInstructions fields
// Returns the same CFGs with machine instructions added
func SelectInstructions(cfgs []*CFG, vrAlloc *VirtualRegisterAllocator, selector InstructionSelector) error {
	// Process each CFG with the shared allocator
	for _, cfg := range cfgs {
		ctx := NewInstructionSelectionContext(selector, vrAlloc)

		if err := ctx.selectCFG(cfg); err != nil {
			return fmt.Errorf("selecting instructions for function %s: %w", cfg.FunctionName, err)
		}
	}

	return nil
}

// selectCFG processes a single CFG and generates instructions for all its blocks
func (ctx *InstructionSelectionContext) selectCFG(cfg *CFG) error {
	ctx.currentCFG = cfg
	ctx.allocateFrameSlots()

	// Allocate VirtualRegisters for parameters based on calling convention
	if cfg.FunctionDecl != nil {
		for i, param := range cfg.FunctionDecl.Parameters {
			regSize := RegisterSize(param.Type.Size() * 8) // Convert bytes to bits

			// Ask calling convention where this parameter should be
			reg, stackOffset, useStack := ctx.callingConvention.GetParameterLocation(i, regSize)

			if useStack {
				// Parameter is on the stack - allocate VirtualRegister with stack home
				// The register allocator can use this stack location for spilling
				vr := ctx.vrAlloc.AllocateOnStack(param.Name, regSize, stackOffset)
				ctx.symbolToVReg[param] = vr

				// Note: We don't eagerly load from stack here. The VirtualRegister
				// represents the parameter value, and the register allocator will
				// generate loads when the value is actually used in a physical register.
			} else {
				// Parameter is in a register - allocate VirtualRegister with constraint
				vr := ctx.vrAlloc.AllocateNamed(param.Name, []*Register{reg})
				vr.Assign(reg)
				ctx.symbolToVReg[param] = vr
			}
		}
	}

	// Process each basic block in the CFG (skip entry and exit - they're reserved)
	for _, block := range cfg.Blocks {
		// Skip entry and exit blocks - they're reserved for prologue/epilogue only
		if block == cfg.Entry || block == cfg.Exit {
			continue
		}

		if err := ctx.selectBasicBlock(block); err != nil {
			return err
		}
	}

	// check if function needs stack frame
	if ctx.currentCFG.FrameLayout.nextOffset > 0 {
		// Generate prologue in the reserved entry block
		// Note: Prologue emits instructions to currentBlock, so we set it to entry
		ctx.selector.SetCurrentBlock(cfg.Entry)
		ctx.selector.SelectFunctionPrologue(cfg.FunctionDecl, ctx.currentCFG.FrameLayout.nextOffset)

		// Generate epilogue in the reserved exit block
		// The exit block is reached by all return statements
		ctx.selector.SetCurrentBlock(cfg.Exit)
		ctx.selector.SelectFunctionEpilogue(cfg.FunctionDecl, ctx.currentCFG.FrameLayout.nextOffset)
	}
	return nil
}

func (ctx *InstructionSelectionContext) allocateFrameSlots() {
	for _, block := range ctx.currentCFG.Blocks {
		for _, stmt := range block.Instructions {
			// TODO: track arrayInitializer expressions.
			if varDecl, ok := stmt.(*zsm.SemVariableDecl); ok {
				var size uint16
				if arrType, ok := varDecl.TypeInfo.(*zsm.ArrayType); ok {
					size = arrType.DataSize()
				} else {
					size = varDecl.TypeInfo.Size()
				}
				ctx.currentCFG.FrameLayout.AddSlot(varDecl.Symbol, size)
			}
		}
	}
}

// selectBasicBlock processes a single basic block
func (ctx *InstructionSelectionContext) selectBasicBlock(block *BasicBlock) error {
	// Skip unreachable blocks (blocks with no predecessors, except entry)
	if len(block.Predecessors) == 0 && block.Label != LabelEntry {
		return nil
	}

	// Set the current block in both context and selector
	ctx.currentBlock = block
	ctx.selector.SetCurrentBlock(block)

	// Process all statements in this block
	for _, stmt := range block.Instructions {
		if err := ctx.selectStatement(stmt); err != nil {
			return err
		}
	}

	// Handle control flow at the end of the block
	// The successors are already established in the CFG
	if len(block.Successors) > 0 {
		// Generate appropriate branch/jump instructions based on successors
		if err := ctx.generateBlockTransition(block); err != nil {
			return err
		}
	}

	return nil
}

// generateBlockTransition generates branch/jump instructions for block transitions
func (ctx *InstructionSelectionContext) generateBlockTransition(block *BasicBlock) error {
	// Check if the last instruction is a control flow statement
	if len(block.Instructions) > 0 {
		lastStmt := block.Instructions[len(block.Instructions)-1]

		switch stmt := lastStmt.(type) {
		case *zsm.SemIf:
			// The SemIf is stored in the condition block
			// Evaluate condition in BranchMode
			// Successors: [0] = then, [1] = else/merge
			if len(block.Successors) >= 2 {
				branchCtx := NewExprContextBranch(block.Successors[0], block.Successors[1])
				_, err := ctx.selectExpressionWithContext(branchCtx, stmt.Condition)
				return err
			}

		case *zsm.SemElsif:
			// Similar to SemIf
			if len(block.Successors) >= 2 {
				branchCtx := NewExprContextBranch(block.Successors[0], block.Successors[1])
				_, err := ctx.selectExpressionWithContext(branchCtx, stmt.Condition)
				return err
			}

		case *zsm.SemFor:
			// For loop condition
			if stmt.Condition != nil {
				// Successors: [0] = body, [1] = exit
				if len(block.Successors) >= 2 {
					branchCtx := NewExprContextBranch(block.Successors[0], block.Successors[1])
					_, err := ctx.selectExpressionWithContext(branchCtx, stmt.Condition)
					return err
				}
			} else {
				// Infinite loop - just jump to body
				if len(block.Successors) > 0 {
					err := ctx.selector.SelectJump(block.Successors[0])
					return err
				}
			}

		case *zsm.SemSelect:
			// Select statement - generate comparison and branches for each case
			// Note: stmt.Expression will be re-evaluated for each case comparison
			// TODO: Optimize by evaluating once and passing VR to comparison
			for i, caseStmt := range stmt.Cases {
				// Create a comparison expression for this case
				cmpExpr := &zsm.SemBinaryOp{
					Op:    zsm.OpEqual,
					Left:  stmt.Expression,
					Right: caseStmt.Value,
				}

				// Branch to case block or next comparison
				if i < len(block.Successors)-1 {
					branchCtx := NewExprContextBranch(block.Successors[i], block.Successors[i+1])
					_, err := ctx.selectExpressionWithContext(branchCtx, cmpExpr)
					if err != nil {
						return err
					}
				}
			}

		case *zsm.SemReturn:
			// Return already handled in selectReturn
			return nil
		}
	}

	// Default: if one successor and not already handled, generate jump
	if len(block.Successors) == 1 {
		err := ctx.selector.SelectJump(block.Successors[0])
		return err
	}

	return nil
}

// selectStatement processes a single statement
func (ctx *InstructionSelectionContext) selectStatement(stmt zsm.SemStatement) error {
	switch s := stmt.(type) {
	case *zsm.SemVariableDecl:
		return ctx.selectVariableDecl(s)

	case *zsm.SemAssignment:
		return ctx.selectAssignment(s)

	case *zsm.SemExpressionStmt:
		// Evaluate expression for side effects
		_, err := ctx.selectExpression(s.Expression)
		return err

	case *zsm.SemReturn:
		return ctx.selectReturn(s)

	case *zsm.SemIf, *zsm.SemElsif, *zsm.SemFor, *zsm.SemSelect:
		// Control flow statements are handled by generateBlockTransition
		// Don't process them here as they're only for branching
		return nil

	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// selectVariableDecl processes a variable declaration
func (ctx *InstructionSelectionContext) selectVariableDecl(decl *zsm.SemVariableDecl) error {
	// Allocate a VirtualRegister for this variable
	// For arrays, this will be a pointer (2 bytes) since ArrayType.Size() returns 2
	regSize := RegisterSize(decl.TypeInfo.Size() * 8) // Convert bytes to bits
	var vrVar *VirtualRegister

	// Special handling for array types
	if arrayType, ok := decl.TypeInfo.(*zsm.ArrayType); ok {
		if arrayType.Length() > 0 {
			// Fixed-size array
			vrVar = ctx.vrAlloc.AllocateNamed(decl.Symbol.Name, Z80Registers16)
			ctx.symbolToVReg[decl.Symbol] = vrVar

			if decl.Initializer == nil {
				dataSize := arrayType.DataSize()
				offset := ctx.currentCFG.FrameLayout.AddSlot(decl.Symbol, dataSize)

				// load value from stack into vrVar
				vrAddress, err := ctx.selector.SelectLoadStackAddress(offset)
				if err != nil {
					return err
				}

				// Move the address into the pointer VR
				err = ctx.selector.SelectMove(vrVar, vrAddress, regSize)
				if err != nil {
					return err
				}
			} else {
				// Has initializer: do not allocate frame storage here.
				// The initializer expression allocates and initializes array data.

			}
		} else {
			// Dynamic/zero-length array: allocate a pointer-sized frame slot
			offset := ctx.currentCFG.FrameLayout.AddSlot(decl.Symbol, uint16(decl.TypeInfo.Size()))
			vrVar = ctx.vrAlloc.AllocateOnStack(decl.Symbol.Name, regSize, uint8(offset))
			ctx.symbolToVReg[decl.Symbol] = vrVar
		}
	} else {
		// Non-array types: allocate as regular VR
		var regs []*Register
		if regSize == Bits8 {
			regs = Z80Registers8
		} else {
			regs = Z80Registers16
		}
		vrVar = ctx.vrAlloc.AllocateNamed(decl.Symbol.Name, regs)
		ctx.symbolToVReg[decl.Symbol] = vrVar
	}

	// If there's an initializer, evaluate it and assign
	if decl.Initializer != nil {
		// Pass target symbol to initializer so it can allocate proper frame slot
		initCtx := NewExprContextSymbol(decl.Symbol)
		initVR, err := ctx.selectExpressionWithContext(initCtx, decl.Initializer)
		if err != nil {
			return err
		}

		// Generate move instruction
		// For arrays, this moves the pointer from the initializer to the variable
		err = ctx.selector.SelectMove(vrVar, initVR, regSize)
		if err != nil {
			return err
		}
	}

	return nil
}

// selectAssignment processes an assignment statement
func (ctx *InstructionSelectionContext) selectAssignment(assign *zsm.SemAssignment) error {
	// Get the target variable's VirtualRegister
	targetVR, ok := ctx.symbolToVReg[assign.Target]
	if !ok {
		return fmt.Errorf("undefined variable: %s", assign.Target.Name)
	}

	// Evaluate the right-hand side
	valueVR, err := ctx.selectExpression(assign.Value)
	if err != nil {
		return err
	}

	// Generate move instruction
	regSize := RegisterSize(assign.Target.Type.Size() * 8)
	err = ctx.selector.SelectMove(targetVR, valueVR, regSize)
	return err
}

// selectReturn processes a return statement
func (ctx *InstructionSelectionContext) selectReturn(ret *zsm.SemReturn) error {
	if ret.Value != nil {
		// Evaluate return value
		valueVR, err := ctx.selectExpression(ret.Value)
		if err != nil {
			return err
		}

		// Get the return register from calling convention
		// Use the function's declared return type size, not the expression's type size
		var returnSize RegisterSize
		if ctx.currentCFG != nil && ctx.currentCFG.FunctionDecl != nil && ctx.currentCFG.FunctionDecl.ReturnType != nil {
			returnSize = RegisterSize(ctx.currentCFG.FunctionDecl.ReturnType.Size() * 8)
		} else {
			// Fallback to expression type if function context not available
			returnSize = RegisterSize(ret.Value.Type().Size() * 8)
		}
		returnReg := ctx.callingConvention.GetReturnValueRegister(returnSize)

		// Move value to the return register
		returnVR := ctx.vrAlloc.Allocate([]*Register{returnReg})
		if err := ctx.selector.SelectMove(returnVR, valueVR, returnSize); err != nil {
			return err
		}

		// Generate return with value in correct register
		return ctx.selector.SelectReturn(returnVR)
	}

	// Generate void return
	return ctx.selector.SelectReturn(nil)
}

// selectExpression processes an expression and returns its result VirtualRegister
// exprCtx: optional context for branch-mode evaluation (nil for value mode)
func (ctx *InstructionSelectionContext) selectExpression(expr zsm.SemExpression) (*VirtualRegister, error) {
	return ctx.selectExpressionWithContext(nil, expr)
}

// selectExpressionWithContext processes an expression with an evaluation context
func (ctx *InstructionSelectionContext) selectExpressionWithContext(exprCtx *ExprContext, expr zsm.SemExpression) (*VirtualRegister, error) {
	// In ValueMode, check cache (BranchMode never caches)
	if exprCtx == nil || exprCtx.Mode == ValueMode {
		if vr, ok := ctx.exprToVReg[expr]; ok {
			return vr, nil
		}
	}

	var resultVR *VirtualRegister
	var err error

	switch e := expr.(type) {
	case *zsm.SemConstant:
		resultVR, err = ctx.selectConstant(e)
	case *zsm.SemSymbolRef:
		resultVR, err = ctx.selectSymbolRef(e)
	case *zsm.SemBinaryOp:
		resultVR, err = ctx.selectBinaryOp(exprCtx, e)
	case *zsm.SemUnaryOp:
		resultVR, err = ctx.selectUnaryOp(exprCtx, e)
	case *zsm.SemFunctionCall:
		resultVR, err = ctx.selectFunctionCall(exprCtx, e)
	case *zsm.SemMemberAccess:
		resultVR, err = ctx.selectMemberAccess(e)
	case *zsm.SemSubscript:
		resultVR, err = ctx.selectSubscript(exprCtx, e)
	case *zsm.SemArrayInitializer:
		resultVR, err = ctx.selectArrayInitializer(exprCtx, e)
	case *zsm.SemTypeInitializer:
		resultVR, err = ctx.selectTypeInitializer(exprCtx, e)
	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result (only in ValueMode)
	if exprCtx == nil || exprCtx.Mode == ValueMode {
		ctx.exprToVReg[expr] = resultVR
	}
	return resultVR, nil
}

// selectConstant loads a constant value
func (ctx *InstructionSelectionContext) selectConstant(constant *zsm.SemConstant) (*VirtualRegister, error) {
	regSize := RegisterSize(constant.Type().Size() * 8)
	return ctx.selector.SelectLoadConstant(constant.Value, regSize)
}

// selectSymbolRef loads a variable value
func (ctx *InstructionSelectionContext) selectSymbolRef(ref *zsm.SemSymbolRef) (*VirtualRegister, error) {
	// Look up the VirtualRegister for this symbol
	vr, ok := ctx.symbolToVReg[ref.Symbol]
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s", ref.Symbol.Name)
	}
	return vr, nil
}

// selectBinaryOp processes binary operations
func (ctx *InstructionSelectionContext) selectBinaryOp(exprCtx *ExprContext, op *zsm.SemBinaryOp) (*VirtualRegister, error) {
	// Handle logical operators specially - they take expressions, not VRs
	if op.Op == zsm.OpLogicalAnd {
		return ctx.selector.SelectLogicalAnd(exprCtx, op.Left, op.Right, ctx.selectExpressionWithContext)
	}
	if op.Op == zsm.OpLogicalOr {
		return ctx.selector.SelectLogicalOr(exprCtx, op.Left, op.Right, ctx.selectExpressionWithContext)
	}

	leftVR, err := ctx.selectExpressionWithContext(exprCtx, op.Left)
	if err != nil {
		return nil, err
	}
	rightVR, err := ctx.selectExpressionWithContext(exprCtx, op.Right)
	if err != nil {
		return nil, err
	}

	// Dispatch to appropriate selector method
	switch op.Op {
	case zsm.OpAdd:
		return ctx.selector.SelectAdd(leftVR, rightVR)

	case zsm.OpSubtract:
		return ctx.selector.SelectSubtract(leftVR, rightVR)

	case zsm.OpMultiply:
		return ctx.selector.SelectMultiply(leftVR, rightVR)

	case zsm.OpDivide:
		return ctx.selector.SelectDivide(leftVR, rightVR)

	case zsm.OpBitwiseAnd:
		return ctx.selector.SelectBitwiseAnd(leftVR, rightVR)
	case zsm.OpBitwiseOr:
		return ctx.selector.SelectBitwiseOr(leftVR, rightVR)

	case zsm.OpBitwiseXor:
		return ctx.selector.SelectBitwiseXor(leftVR, rightVR)

	case zsm.OpEqual:
		return ctx.selector.SelectEqual(exprCtx, leftVR, rightVR)

	case zsm.OpNotEqual:
		return ctx.selector.SelectNotEqual(exprCtx, leftVR, rightVR)

	case zsm.OpLessThan:
		return ctx.selector.SelectLessThan(exprCtx, leftVR, rightVR)

	case zsm.OpLessEqual:
		return ctx.selector.SelectLessEqual(exprCtx, leftVR, rightVR)
	case zsm.OpGreaterThan:
		return ctx.selector.SelectGreaterThan(exprCtx, leftVR, rightVR)

	case zsm.OpGreaterEqual:
		return ctx.selector.SelectGreaterEqual(exprCtx, leftVR, rightVR)

	default:
		return nil, fmt.Errorf("unknown binary operator: %v", op.Op)
	}
}

// selectUnaryOp processes unary operations
func (ctx *InstructionSelectionContext) selectUnaryOp(exprCtx *ExprContext, op *zsm.SemUnaryOp) (*VirtualRegister, error) {
	// Handle LogicalNot specially - it takes expressions
	if op.Op == zsm.OpLogicalNot {
		return ctx.selector.SelectLogicalNot(exprCtx, op.Operand, ctx.selectExpressionWithContext)
	}

	// Other unary ops need VR operand
	operandVR, err := ctx.selectExpressionWithContext(exprCtx, op.Operand)
	if err != nil {
		return nil, err
	}

	// Dispatch to appropriate selector method
	switch op.Op {
	case zsm.OpNegate:
		return ctx.selector.SelectNegate(operandVR)
	case zsm.OpBitwiseNot:
		return ctx.selector.SelectBitwiseNot(operandVR)
	case zsm.OpIncrement:
		return ctx.selector.SelectIncrement(operandVR)
	case zsm.OpDecrement:
		return ctx.selector.SelectDecrement(operandVR)
	default:
		return nil, fmt.Errorf("unknown unary operator: %v", op.Op)
	}
}

// selectFunctionCall processes function calls
func (ctx *InstructionSelectionContext) selectFunctionCall(exprCtx *ExprContext, call *zsm.SemFunctionCall) (*VirtualRegister, error) {
	// Evaluate arguments with parameter symbols for proper stack tracking
	argVRs := make([]*VirtualRegister, len(call.Arguments))
	for i, arg := range call.Arguments {
		// Create a synthetic parameter symbol for stack allocation tracking
		// This allows array literals in arguments to be tracked: foo([1,2,3])
		paramSymbol := &zsm.Symbol{
			Name: fmt.Sprintf("%s.arg%d", call.Function.Name, i),
			Kind: zsm.SymbolVariable,
		}

		// Pass parameter symbol as target for array initializer tracking
		argCtx := exprCtx.WithSymbol(paramSymbol)
		vr, err := ctx.selectExpressionWithContext(argCtx, arg)
		if err != nil {
			return nil, err
		}
		argVRs[i] = vr
	}

	// Get return size
	returnSize := RegisterSize(0)
	if call.Type() != nil {
		returnSize = RegisterSize(call.Type().Size() * 8)
	}

	// Generate call
	return ctx.selector.SelectCall(call.Function.Name, argVRs, returnSize)
}

// selectMemberAccess processes struct member access
func (ctx *InstructionSelectionContext) selectMemberAccess(access *zsm.SemMemberAccess) (*VirtualRegister, error) {
	// Get the object
	objectVR, err := ctx.selectExpression(*access.Object)
	if err != nil {
		return nil, err
	}

	// Load member at offset
	offset := access.Field.Offset
	regSize := RegisterSize(access.Type().Size() * 8)
	return ctx.selector.SelectLoad(objectVR, offset, regSize)
}

// selectSubscript processes array subscripting
func (ctx *InstructionSelectionContext) selectSubscript(exprCtx *ExprContext, subscript *zsm.SemSubscript) (*VirtualRegister, error) {
	// Get the array base address
	arrayVR, err := ctx.selectExpressionWithContext(exprCtx, subscript.Array)
	if err != nil {
		return nil, err
	}

	// Get the index
	indexVR, err := ctx.selectExpressionWithContext(exprCtx, subscript.Index)
	if err != nil {
		return nil, err
	}

	// Calculate element size
	elementSize := subscript.Type().Size()
	regSize := RegisterSize(subscript.Type().Size() * 8)

	// Generate indexed load
	return ctx.selector.SelectLoadIndexed(arrayVR, indexVR, elementSize, regSize)
}

// selectTypeInitializer processes struct initialization
func (ctx *InstructionSelectionContext) selectTypeInitializer(exprCtx *ExprContext, init *zsm.SemTypeInitializer) (*VirtualRegister, error) {
	// Allocate space for the struct
	structVR := ctx.vrAlloc.Allocate(Z80Registers16)

	// Initialize each field
	for _, fieldInit := range init.Fields {
		valueVR, err := ctx.selectExpressionWithContext(exprCtx, fieldInit.Value)
		if err != nil {
			return nil, err
		}

		offset := fieldInit.Field.Offset
		fieldRegSize := RegisterSize(fieldInit.Field.Type.Size() * 8)
		if err := ctx.selector.SelectStore(structVR, valueVR, offset, fieldRegSize); err != nil {
			return nil, err
		}
	}

	return structVR, nil
}

func (ctx *InstructionSelectionContext) selectArrayInitializer(exprCtx *ExprContext, init *zsm.SemArrayInitializer) (*VirtualRegister, error) {
	arrayType, ok := init.Type().(*zsm.ArrayType)
	if !ok {
		return nil, fmt.Errorf("array initializer doesn't have array type")
	}

	if exprCtx.TargetSymbol == nil {
		return nil, fmt.Errorf("array initializer missing target symbol in expression context")
	}

	// Allocate stack space for array data via FrameLayout
	dataSize := arrayType.DataSize()
	dataOffset := ctx.currentCFG.FrameLayout.AddSlot(exprCtx.TargetSymbol, dataSize)

	// Compute address of array data: SP + offset
	addressVR, err := ctx.selector.SelectLoadStackAddress(dataOffset)
	if err != nil {
		return nil, err
	}

	// Initialize each element
	elementSize := arrayType.ElementType().Size()
	elementRegSize := RegisterSize(elementSize * 8)

	for _, elemExpr := range init.Elements {
		// Evaluate element expression with cleared target symbol
		// (nested expressions should not inherit the array's target)
		valueVR, err := ctx.selectExpressionWithContext(nil, elemExpr)
		if err != nil {
			return nil, err
		}
		if err := ctx.selector.SelectStoreSequential(addressVR, valueVR, elementSize, elementRegSize); err != nil {
			return nil, err
		}
	}

	// Return the pointer to the array
	return addressVR, nil
}

func DumpInstructions(instructions []MachineInstruction) {
	fmt.Println("========== INSTRUCTIONS ==========")
	for i, instr := range instructions {
		fmt.Printf("  [%4d] %s\n", i, instr.String())
	}
	fmt.Println()
}
