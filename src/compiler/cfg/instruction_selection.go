package cfg

import (
	"fmt"
	"zenith/compiler/zir"
)

// InstructionSelectionContext manages the state during instruction selection
type InstructionSelectionContext struct {
	selector          InstructionSelector
	vrAlloc           *VirtualRegisterAllocator
	callingConvention CallingConvention

	// Maps ZIR symbols to their VirtualRegisters
	symbolToVReg map[*zir.Symbol]*VirtualRegister

	// Maps expression nodes to their result VirtualRegisters (for reuse)
	exprToVReg map[zir.IRExpression]*VirtualRegister

	// Current function being processed
	currentFunction *zir.IRFunctionDecl

	// Current CFG being processed
	currentCFG *CFG
}

// NewInstructionSelectionContext creates a new context for instruction selection
func NewInstructionSelectionContext(selector InstructionSelector, cc CallingConvention) *InstructionSelectionContext {
	return &InstructionSelectionContext{
		selector:          selector,
		vrAlloc:           NewVirtualRegisterAllocator(),
		callingConvention: cc,
		symbolToVReg:      make(map[*zir.Symbol]*VirtualRegister),
		exprToVReg:        make(map[zir.IRExpression]*VirtualRegister),
	}
}

// SelectInstructions walks the IR and generates machine instructions
// It first builds a CFG for each function, then generates instructions for each block
func SelectInstructions(compilationUnit *zir.IRCompilationUnit, selector InstructionSelector, cc CallingConvention) error {
	ctx := NewInstructionSelectionContext(selector, cc)

	// Process each function
	for _, decl := range compilationUnit.Declarations {
		if funcDecl, ok := decl.(*zir.IRFunctionDecl); ok {
			if err := ctx.selectFunction(funcDecl); err != nil {
				return fmt.Errorf("selecting instructions for function %s: %w", funcDecl.Name, err)
			}
		}
	}

	return nil
}

// selectFunction processes a single function
func (ctx *InstructionSelectionContext) selectFunction(fn *zir.IRFunctionDecl) error {
	ctx.currentFunction = fn

	// Build CFG for this function
	builder := NewCFGBuilder()
	cfg := builder.BuildCFG(fn)
	ctx.currentCFG = cfg

	// Allocate VirtualRegisters for parameters based on calling convention
	for i, param := range fn.Parameters {
		size := param.Type.Size() * 8 // Convert bytes to bits
		sizeBytes := param.Type.Size()

		// Ask calling convention where this parameter should be
		reg, stackOffset, useStack := ctx.callingConvention.GetParameterLocation(i, sizeBytes)

		if useStack {
			// Parameter is on the stack - allocate VirtualRegister with stack home
			// The register allocator can use this stack location for spilling
			vr := ctx.vrAlloc.AllocateWithStackHome(param.Name, size, stackOffset)
			ctx.symbolToVReg[param] = vr

			// Note: We don't eagerly load from stack here. The VirtualRegister
			// represents the parameter value, and the register allocator will
			// generate loads when the value is actually used in a physical register.
		} else {
			// Parameter is in a register - allocate VirtualRegister with constraint
			vr := ctx.vrAlloc.AllocateConstrained(size, []*Register{reg}, reg.Class)
			vr.Name = param.Name
			ctx.symbolToVReg[param] = vr
		}
	}

	// Process each basic block in the CFG
	for _, block := range cfg.Blocks {
		if err := ctx.selectBasicBlock(block); err != nil {
			return err
		}
	}

	return nil
}

// selectBasicBlock processes a single basic block
func (ctx *InstructionSelectionContext) selectBasicBlock(block *BasicBlock) error {
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
		case *zir.IRIf:
			// The IRIf is stored in the condition block
			// Evaluate condition and branch to successors
			condVR, err := ctx.selectExpression(stmt.Condition)
			if err != nil {
				return err
			}

			// Successors: [0] = then, [1] = else/merge
			if len(block.Successors) >= 2 {
				err = ctx.selector.SelectBranch(condVR, block.Successors[0], block.Successors[1])
				return err
			}

		case *zir.IRElsif:
			// Similar to IRIf
			condVR, err := ctx.selectExpression(stmt.Condition)
			if err != nil {
				return err
			}

			if len(block.Successors) >= 2 {
				err = ctx.selector.SelectBranch(condVR, block.Successors[0], block.Successors[1])
				return err
			}

		case *zir.IRFor:
			// For loop condition
			if stmt.Condition != nil {
				condVR, err := ctx.selectExpression(stmt.Condition)
				if err != nil {
					return err
				}

				// Successors: [0] = body, [1] = exit
				if len(block.Successors) >= 2 {
					err = ctx.selector.SelectBranch(condVR, block.Successors[0], block.Successors[1])
					return err
				}
			} else {
				// Infinite loop - just jump to body
				if len(block.Successors) > 0 {
					err := ctx.selector.SelectJump(block.Successors[0])
					return err
				}
			}

		case *zir.IRSelect:
			// Select statement - generate comparison and branches for each case
			exprVR, err := ctx.selectExpression(stmt.Expression)
			if err != nil {
				return err
			}

			size := stmt.Expression.Type().Size() * 8
			for i, caseStmt := range stmt.Cases {
				caseValueVR, err := ctx.selectExpression(caseStmt.Value)
				if err != nil {
					return err
				}

				cmpVR, err := ctx.selector.SelectEqual(exprVR, caseValueVR, size)
				if err != nil {
					return err
				}

				// Branch to case block or next comparison
				if i < len(block.Successors)-1 {
					err = ctx.selector.SelectBranch(cmpVR, block.Successors[i], block.Successors[i+1])
					if err != nil {
						return err
					}
				}
			}

		case *zir.IRReturn:
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
func (ctx *InstructionSelectionContext) selectStatement(stmt zir.IRStatement) error {
	switch s := stmt.(type) {
	case *zir.IRVariableDecl:
		return ctx.selectVariableDecl(s)

	case *zir.IRAssignment:
		return ctx.selectAssignment(s)

	case *zir.IRExpressionStmt:
		// Evaluate expression for side effects
		_, err := ctx.selectExpression(s.Expression)
		return err

	case *zir.IRReturn:
		return ctx.selectReturn(s)

	case *zir.IRIf, *zir.IRElsif, *zir.IRFor, *zir.IRSelect:
		// Control flow statements are handled by generateBlockTransition
		// Don't process them here as they're only for branching
		return nil

	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// selectVariableDecl processes a variable declaration
func (ctx *InstructionSelectionContext) selectVariableDecl(decl *zir.IRVariableDecl) error {
	// Allocate a VirtualRegister for this variable
	size := decl.TypeInfo.Size() * 8 // Convert bytes to bits
	vr := ctx.vrAlloc.AllocateNamed(decl.Symbol.Name, size)
	ctx.symbolToVReg[decl.Symbol] = vr

	// If there's an initializer, evaluate it and assign
	if decl.Initializer != nil {
		initVR, err := ctx.selectExpression(decl.Initializer)
		if err != nil {
			return err
		}

		// Generate move instruction
		err = ctx.selector.SelectMove(vr, initVR, size)
		if err != nil {
			return err
		}
	}

	return nil
}

// selectAssignment processes an assignment statement
func (ctx *InstructionSelectionContext) selectAssignment(assign *zir.IRAssignment) error {
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
	size := assign.Target.Type.Size() * 8
	err = ctx.selector.SelectMove(targetVR, valueVR, size)
	return err
}

// selectReturn processes a return statement
func (ctx *InstructionSelectionContext) selectReturn(ret *zir.IRReturn) error {
	if ret.Value != nil {
		// Evaluate return value
		valueVR, err := ctx.selectExpression(ret.Value)
		if err != nil {
			return err
		}

		// Get the return register from calling convention
		returnSize := ret.Value.Type().Size()
		returnReg := ctx.callingConvention.GetReturnValueRegister(returnSize)

		// Move value to the return register
		size := returnSize * 8
		returnVR := ctx.vrAlloc.AllocateConstrained(size, []*Register{returnReg}, returnReg.Class)
		if err := ctx.selector.SelectMove(returnVR, valueVR, size); err != nil {
			return err
		}

		// Generate return with value in correct register
		return ctx.selector.SelectReturn(returnVR)
	}

	// Generate void return
	return ctx.selector.SelectReturn(nil)
}

// selectExpression processes an expression and returns its result VirtualRegister
func (ctx *InstructionSelectionContext) selectExpression(expr zir.IRExpression) (*VirtualRegister, error) {
	// Check if we've already processed this expression
	if vr, ok := ctx.exprToVReg[expr]; ok {
		return vr, nil
	}

	var resultVR *VirtualRegister
	var err error

	switch e := expr.(type) {
	case *zir.IRConstant:
		resultVR, err = ctx.selectConstant(e)

	case *zir.IRSymbolRef:
		resultVR, err = ctx.selectSymbolRef(e)

	case *zir.IRBinaryOp:
		resultVR, err = ctx.selectBinaryOp(e)

	case *zir.IRUnaryOp:
		resultVR, err = ctx.selectUnaryOp(e)

	case *zir.IRFunctionCall:
		resultVR, err = ctx.selectFunctionCall(e)

	case *zir.IRMemberAccess:
		resultVR, err = ctx.selectMemberAccess(e)

	case *zir.IRTypeInitializer:
		resultVR, err = ctx.selectTypeInitializer(e)

	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result
	ctx.exprToVReg[expr] = resultVR
	return resultVR, nil
}

// selectConstant loads a constant value
func (ctx *InstructionSelectionContext) selectConstant(constant *zir.IRConstant) (*VirtualRegister, error) {
	size := constant.Type().Size() * 8
	return ctx.selector.SelectLoadConstant(constant.Value, size)
}

// selectSymbolRef loads a variable value
func (ctx *InstructionSelectionContext) selectSymbolRef(ref *zir.IRSymbolRef) (*VirtualRegister, error) {
	// Look up the VirtualRegister for this symbol
	vr, ok := ctx.symbolToVReg[ref.Symbol]
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s", ref.Symbol.Name)
	}
	return vr, nil
}

// selectBinaryOp processes binary operations
func (ctx *InstructionSelectionContext) selectBinaryOp(op *zir.IRBinaryOp) (*VirtualRegister, error) {
	// Evaluate operands
	leftVR, err := ctx.selectExpression(op.Left)
	if err != nil {
		return nil, err
	}

	rightVR, err := ctx.selectExpression(op.Right)
	if err != nil {
		return nil, err
	}

	size := op.Type().Size() * 8

	// Dispatch to appropriate selector method
	switch op.Op {
	case zir.OpAdd:
		return ctx.selector.SelectAdd(leftVR, rightVR, size)

	case zir.OpSubtract:
		return ctx.selector.SelectSubtract(leftVR, rightVR, size)

	case zir.OpMultiply:
		return ctx.selector.SelectMultiply(leftVR, rightVR, size)

	case zir.OpDivide:
		return ctx.selector.SelectDivide(leftVR, rightVR, size)

	case zir.OpBitwiseAnd:
		return ctx.selector.SelectBitwiseAnd(leftVR, rightVR, size)

	case zir.OpBitwiseOr:
		return ctx.selector.SelectBitwiseOr(leftVR, rightVR, size)

	case zir.OpBitwiseXor:
		return ctx.selector.SelectBitwiseXor(leftVR, rightVR, size)

	case zir.OpEqual:
		return ctx.selector.SelectEqual(leftVR, rightVR, size)

	case zir.OpNotEqual:
		return ctx.selector.SelectNotEqual(leftVR, rightVR, size)

	case zir.OpLessThan:
		return ctx.selector.SelectLessThan(leftVR, rightVR, size)

	case zir.OpLessEqual:
		return ctx.selector.SelectLessEqual(leftVR, rightVR, size)

	case zir.OpGreaterThan:
		return ctx.selector.SelectGreaterThan(leftVR, rightVR, size)

	case zir.OpGreaterEqual:
		return ctx.selector.SelectGreaterEqual(leftVR, rightVR, size)

	case zir.OpLogicalAnd:
		return ctx.selector.SelectLogicalAnd(leftVR, rightVR)

	case zir.OpLogicalOr:
		return ctx.selector.SelectLogicalOr(leftVR, rightVR)

	default:
		return nil, fmt.Errorf("unknown binary operator: %v", op.Op)
	}
}

// selectUnaryOp processes unary operations
func (ctx *InstructionSelectionContext) selectUnaryOp(op *zir.IRUnaryOp) (*VirtualRegister, error) {
	// Evaluate operand
	operandVR, err := ctx.selectExpression(op.Operand)
	if err != nil {
		return nil, err
	}

	size := op.Type().Size() * 8

	// Dispatch to appropriate selector method
	switch op.Op {
	case zir.OpNegate:
		return ctx.selector.SelectNegate(operandVR, size)

	case zir.OpNot:
		return ctx.selector.SelectLogicalNot(operandVR)

	case zir.OpBitwiseNot:
		return ctx.selector.SelectBitwiseNot(operandVR, size)

	default:
		return nil, fmt.Errorf("unknown unary operator: %v", op.Op)
	}
}

// selectFunctionCall processes function calls
func (ctx *InstructionSelectionContext) selectFunctionCall(call *zir.IRFunctionCall) (*VirtualRegister, error) {
	// Evaluate arguments
	argVRs := make([]*VirtualRegister, len(call.Arguments))
	for i, arg := range call.Arguments {
		vr, err := ctx.selectExpression(arg)
		if err != nil {
			return nil, err
		}
		argVRs[i] = vr
	}

	// Get return size
	returnSize := 0
	if call.Type() != nil {
		returnSize = call.Type().Size() * 8
	}

	// Generate call
	return ctx.selector.SelectCall(call.Function.Name, argVRs, returnSize)
}

// selectMemberAccess processes struct member access
func (ctx *InstructionSelectionContext) selectMemberAccess(access *zir.IRMemberAccess) (*VirtualRegister, error) {
	// Get the object
	objectVR, err := ctx.selectExpression(*access.Object)
	if err != nil {
		return nil, err
	}

	// Load member at offset
	offset := access.Field.Offset
	size := access.Type().Size() * 8
	return ctx.selector.SelectLoad(objectVR, offset, size)
}

// selectTypeInitializer processes struct initialization
func (ctx *InstructionSelectionContext) selectTypeInitializer(init *zir.IRTypeInitializer) (*VirtualRegister, error) {
	// Allocate space for the struct
	size := init.Type().Size() * 8
	structVR := ctx.vrAlloc.Allocate(size)

	// Initialize each field
	for _, fieldInit := range init.Fields {
		valueVR, err := ctx.selectExpression(fieldInit.Value)
		if err != nil {
			return nil, err
		}

		offset := fieldInit.Field.Offset
		fieldSize := fieldInit.Field.Type.Size() * 8
		if err := ctx.selector.SelectStore(structVR, valueVR, offset, fieldSize); err != nil {
			return nil, err
		}
	}

	return structVR, nil
}
