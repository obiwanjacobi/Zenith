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

	// Maps ZIR symbols to their VirtualRegisters
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
func NewInstructionSelectionContext(selector InstructionSelector) *InstructionSelectionContext {
	return &InstructionSelectionContext{
		selector:          selector,
		vrAlloc:           NewVirtualRegisterAllocator(),
		callingConvention: selector.GetCallingConvention(),
		symbolToVReg:      make(map[*zsm.Symbol]*VirtualRegister),
		exprToVReg:        make(map[zsm.SemExpression]*VirtualRegister),
	}
}

// SelectInstructions generates machine instructions for pre-built CFGs
// Takes a slice of CFGs and populates their MachineInstructions fields
// Returns the same CFGs with machine instructions added
func SelectInstructions(cfgs []*CFG, selector InstructionSelector) ([]*CFG, error) {
	ctx := NewInstructionSelectionContext(selector)

	// Process each CFG
	for _, cfg := range cfgs {
		if err := ctx.selectCFG(cfg); err != nil {
			return nil, fmt.Errorf("selecting instructions for function %s: %w", cfg.FunctionName, err)
		}
	}

	return cfgs, nil
}

// selectCFG processes a single CFG and generates instructions for all its blocks
func (ctx *InstructionSelectionContext) selectCFG(cfg *CFG) error {
	ctx.currentCFG = cfg

	// Allocate VirtualRegisters for parameters based on calling convention
	if cfg.FunctionDecl != nil {
		for i, param := range cfg.FunctionDecl.Parameters {
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

		case *zsm.SemElsif:
			// Similar to SemIf
			condVR, err := ctx.selectExpression(stmt.Condition)
			if err != nil {
				return err
			}

			if len(block.Successors) >= 2 {
				err = ctx.selector.SelectBranch(condVR, block.Successors[0], block.Successors[1])
				return err
			}

		case *zsm.SemFor:
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

		case *zsm.SemSelect:
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
	size := assign.Target.Type.Size() * 8
	err = ctx.selector.SelectMove(targetVR, valueVR, size)
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
func (ctx *InstructionSelectionContext) selectExpression(expr zsm.SemExpression) (*VirtualRegister, error) {
	// Check if we've already processed this expression
	if vr, ok := ctx.exprToVReg[expr]; ok {
		return vr, nil
	}

	var resultVR *VirtualRegister
	var err error

	switch e := expr.(type) {
	case *zsm.SemConstant:
		resultVR, err = ctx.selectConstant(e)

	case *zsm.SemSymbolRef:
		resultVR, err = ctx.selectSymbolRef(e)

	case *zsm.SemBinaryOp:
		resultVR, err = ctx.selectBinaryOp(e)

	case *zsm.SemUnaryOp:
		resultVR, err = ctx.selectUnaryOp(e)

	case *zsm.SemFunctionCall:
		resultVR, err = ctx.selectFunctionCall(e)

	case *zsm.SemMemberAccess:
		resultVR, err = ctx.selectMemberAccess(e)

	case *zsm.SemTypeInitializer:
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
func (ctx *InstructionSelectionContext) selectConstant(constant *zsm.SemConstant) (*VirtualRegister, error) {
	size := constant.Type().Size() * 8
	return ctx.selector.SelectLoadConstant(constant.Value, size)
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
func (ctx *InstructionSelectionContext) selectBinaryOp(op *zsm.SemBinaryOp) (*VirtualRegister, error) {
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
	case zsm.OpAdd:
		return ctx.selector.SelectAdd(leftVR, rightVR, size)

	case zsm.OpSubtract:
		return ctx.selector.SelectSubtract(leftVR, rightVR, size)

	case zsm.OpMultiply:
		return ctx.selector.SelectMultiply(leftVR, rightVR, size)

	case zsm.OpDivide:
		return ctx.selector.SelectDivide(leftVR, rightVR, size)

	case zsm.OpBitwiseAnd:
		return ctx.selector.SelectBitwiseAnd(leftVR, rightVR, size)

	case zsm.OpBitwiseOr:
		return ctx.selector.SelectBitwiseOr(leftVR, rightVR, size)

	case zsm.OpBitwiseXor:
		return ctx.selector.SelectBitwiseXor(leftVR, rightVR, size)

	case zsm.OpEqual:
		return ctx.selector.SelectEqual(leftVR, rightVR, size)

	case zsm.OpNotEqual:
		return ctx.selector.SelectNotEqual(leftVR, rightVR, size)

	case zsm.OpLessThan:
		return ctx.selector.SelectLessThan(leftVR, rightVR, size)

	case zsm.OpLessEqual:
		return ctx.selector.SelectLessEqual(leftVR, rightVR, size)

	case zsm.OpGreaterThan:
		return ctx.selector.SelectGreaterThan(leftVR, rightVR, size)

	case zsm.OpGreaterEqual:
		return ctx.selector.SelectGreaterEqual(leftVR, rightVR, size)

	case zsm.OpLogicalAnd:
		return ctx.selector.SelectLogicalAnd(leftVR, rightVR)

	case zsm.OpLogicalOr:
		return ctx.selector.SelectLogicalOr(leftVR, rightVR)

	default:
		return nil, fmt.Errorf("unknown binary operator: %v", op.Op)
	}
}

// selectUnaryOp processes unary operations
func (ctx *InstructionSelectionContext) selectUnaryOp(op *zsm.SemUnaryOp) (*VirtualRegister, error) {
	// Evaluate operand
	operandVR, err := ctx.selectExpression(op.Operand)
	if err != nil {
		return nil, err
	}

	size := op.Type().Size() * 8

	// Dispatch to appropriate selector method
	switch op.Op {
	case zsm.OpNegate:
		return ctx.selector.SelectNegate(operandVR, size)

	case zsm.OpNot:
		return ctx.selector.SelectLogicalNot(operandVR)

	case zsm.OpBitwiseNot:
		return ctx.selector.SelectBitwiseNot(operandVR, size)

	default:
		return nil, fmt.Errorf("unknown unary operator: %v", op.Op)
	}
}

// selectFunctionCall processes function calls
func (ctx *InstructionSelectionContext) selectFunctionCall(call *zsm.SemFunctionCall) (*VirtualRegister, error) {
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
func (ctx *InstructionSelectionContext) selectMemberAccess(access *zsm.SemMemberAccess) (*VirtualRegister, error) {
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
func (ctx *InstructionSelectionContext) selectTypeInitializer(init *zsm.SemTypeInitializer) (*VirtualRegister, error) {
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
