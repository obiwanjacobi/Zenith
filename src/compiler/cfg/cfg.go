package cfg

import (
	"fmt"
	"strings"

	"zenith/compiler/zsm"
)

// ============================================================================
// Control Flow Graph (CFG) Model
// ============================================================================

// BlockLabel represents a CFG block label type
type BlockLabel int

// Block label constants
const (
	LabelEntry BlockLabel = iota
	LabelExit
	LabelIfThen
	LabelIfElse
	LabelIfMerge
	LabelElsifCond
	LabelElsifThen
	LabelForCond
	LabelForBody
	LabelForInc
	LabelForExit
	LabelSelectCase
	LabelSelectElse
	LabelSelectMerge
)

// String returns the string representation of a BlockLabel
func (l BlockLabel) String() string {
	switch l {
	case LabelEntry:
		return "entry"
	case LabelExit:
		return "exit"
	case LabelIfThen:
		return "if.then"
	case LabelIfElse:
		return "if.else"
	case LabelIfMerge:
		return "if.merge"
	case LabelElsifCond:
		return "elsif.cond"
	case LabelElsifThen:
		return "elsif.then"
	case LabelForCond:
		return "for.cond"
	case LabelForBody:
		return "for.body"
	case LabelForInc:
		return "for.inc"
	case LabelForExit:
		return "for.exit"
	case LabelSelectCase:
		return "select.case"
	case LabelSelectElse:
		return "select.else"
	case LabelSelectMerge:
		return "select.merge"
	default:
		return "unknown"
	}
}

// BasicBlock represents a sequence of instructions with one entry and one exit
type BasicBlock struct {
	ID                  int                  // Unique identifier
	Label               BlockLabel           // Label for this block
	LabelID             int                  // Optional numeric suffix for label uniqueness
	Instructions        []zsm.SemStatement   // Statements in this block (IR level)
	MachineInstructions []MachineInstruction // Generated machine instructions for this block
	Successors          []*BasicBlock        // Blocks that can follow this one
	Predecessors        []*BasicBlock        // Blocks that can jump to this one
}

// CFG represents a control flow graph for a function
type CFG struct {
	Entry        *BasicBlock   // Entry block
	Exit         *BasicBlock   // Exit block (for return statements)
	Blocks       []*BasicBlock // All blocks in the graph
	FunctionName string        // Name of the function (for qualified variable names)
}

// ============================================================================
// CFG Builder - Transforms IR to CFG
// ============================================================================

// CFGBuilder builds a control flow graph from IR
type CFGBuilder struct {
	nextBlockID  int
	blocks       []*BasicBlock
	currentBlock *BasicBlock
}

// NewCFGBuilder creates a new CFG builder
func NewCFGBuilder() *CFGBuilder {
	return &CFGBuilder{
		nextBlockID: 0,
		blocks:      []*BasicBlock{},
	}
}

// BuildCFG transforms a function's IR into a CFG
func (b *CFGBuilder) BuildCFG(funcDecl *zsm.SemFunctionDecl) *CFG {
	// Create entry block
	entry := b.newBlock(LabelEntry, -1)
	b.currentBlock = entry

	// Create exit block (for returns)
	exit := b.newBlock(LabelExit, -1)

	// Process function body
	if funcDecl.Body != nil {
		b.processBlock(funcDecl.Body, exit)
	}

	// Connect current block to exit if it doesn't already have successors
	if len(b.currentBlock.Successors) == 0 {
		b.addEdge(b.currentBlock, exit)
	}

	return &CFG{
		Entry:        entry,
		Exit:         exit,
		Blocks:       b.blocks,
		FunctionName: funcDecl.Name,
	}
}

// newBlock creates a new basic block
// If referenceID >= 0, stores it as LabelID for uniqueness
func (b *CFGBuilder) newBlock(label BlockLabel, referenceID int) *BasicBlock {
	block := &BasicBlock{
		ID:                  b.nextBlockID,
		Label:               label,
		LabelID:             referenceID,
		Instructions:        []zsm.SemStatement{},
		MachineInstructions: []MachineInstruction{},
		Successors:          []*BasicBlock{},
		Predecessors:        []*BasicBlock{},
	}
	b.nextBlockID++
	b.blocks = append(b.blocks, block)
	return block
}

// GetFullLabel returns the full label string with ID suffix if present
func (b *BasicBlock) GetFullLabel() string {
	if b.LabelID >= 0 {
		return fmt.Sprintf("%s.%d", b.Label.String(), b.LabelID)
	}
	return b.Label.String()
}

// addEdge adds a control flow edge between two blocks
func (b *CFGBuilder) addEdge(from, to *BasicBlock) {
	from.Successors = append(from.Successors, to)
	to.Predecessors = append(to.Predecessors, from)
}

// processBlock processes an IR block and builds CFG blocks
func (b *CFGBuilder) processBlock(block *zsm.SemBlock, exitBlock *BasicBlock) {
	for _, stmt := range block.Statements {
		b.processStatement(stmt, exitBlock)
	}
}

// processStatement processes a single IR statement
func (b *CFGBuilder) processStatement(stmt zsm.SemStatement, exitBlock *BasicBlock) {
	switch s := stmt.(type) {
	case *zsm.SemVariableDecl:
		// Variable declarations are simple statements
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, s)

	case *zsm.SemAssignment:
		// Assignments are simple statements
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, s)

	case *zsm.SemExpressionStmt:
		// Expression statements (e.g., function calls)
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, s)

	case *zsm.SemReturn:
		// Return statement - add to current block and connect to exit
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, s)
		b.addEdge(b.currentBlock, exitBlock)
		// Create a new block for any statements after return (unreachable code)
		b.currentBlock = b.newBlock(LabelEntry, 0) // Use generic label for continuation

	case *zsm.SemIf:
		b.processIf(s, exitBlock)

	case *zsm.SemFor:
		b.processFor(s, exitBlock)

	case *zsm.SemSelect:
		b.processSelect(s, exitBlock)

	default:
		// Unknown statement type - add it anyway
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, stmt)
	}
}

// processIf processes an if statement, creating blocks for branches
func (b *CFGBuilder) processIf(ifStmt *zsm.SemIf, exitBlock *BasicBlock) {
	// Current block evaluates condition and branches
	condBlock := b.currentBlock
	condBlock.Instructions = append(condBlock.Instructions, ifStmt)

	// Create then block
	thenBlock := b.newBlock(LabelIfThen, condBlock.ID)
	b.addEdge(condBlock, thenBlock)
	b.currentBlock = thenBlock
	b.processBlock(ifStmt.ThenBlock, exitBlock)
	thenExitBlock := b.currentBlock

	// Create merge block (where all branches converge)
	mergeBlock := b.newBlock(LabelIfMerge, condBlock.ID)

	// Process elsif blocks
	var elsifExitBlocks []*BasicBlock
	prevCondBlock := condBlock
	for _, elsif := range ifStmt.ElsifBlocks {
		elsifCondBlock := b.newBlock(LabelElsifCond, prevCondBlock.ID)
		b.addEdge(prevCondBlock, elsifCondBlock)
		elsifCondBlock.Instructions = append(elsifCondBlock.Instructions, elsif)

		elsifThenBlock := b.newBlock(LabelElsifThen, elsifCondBlock.ID)
		b.addEdge(elsifCondBlock, elsifThenBlock)
		b.currentBlock = elsifThenBlock
		b.processBlock(elsif.ThenBlock, exitBlock)
		elsifExitBlocks = append(elsifExitBlocks, b.currentBlock)

		prevCondBlock = elsifCondBlock
	}

	// Process else block if present
	var elseExitBlock *BasicBlock
	if ifStmt.ElseBlock != nil {
		elseBlock := b.newBlock(LabelIfElse, prevCondBlock.ID)
		b.addEdge(prevCondBlock, elseBlock)
		b.currentBlock = elseBlock
		b.processBlock(ifStmt.ElseBlock, exitBlock)
		elseExitBlock = b.currentBlock
	} else {
		// If no else block, condition can fall through to merge
		b.addEdge(prevCondBlock, mergeBlock)
	}

	// Connect all exit blocks to merge block
	b.addEdge(thenExitBlock, mergeBlock)
	for _, elsifExit := range elsifExitBlocks {
		b.addEdge(elsifExit, mergeBlock)
	}
	if elseExitBlock != nil {
		b.addEdge(elseExitBlock, mergeBlock)
	}

	// Continue from merge block
	b.currentBlock = mergeBlock
}

// processFor processes a for loop, creating blocks for loop structure
func (b *CFGBuilder) processFor(forStmt *zsm.SemFor, exitBlock *BasicBlock) {
	// Process initializer in current block
	if forStmt.Initializer != nil {
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, forStmt.Initializer)
	}

	// Create condition block
	initBlock := b.currentBlock
	condBlock := b.newBlock(LabelForCond, initBlock.ID)
	b.addEdge(b.currentBlock, condBlock)
	condBlock.Instructions = append(condBlock.Instructions, forStmt)

	// Create body block
	bodyBlock := b.newBlock(LabelForBody, condBlock.ID)
	b.addEdge(condBlock, bodyBlock)
	b.currentBlock = bodyBlock
	if forStmt.Body != nil {
		b.processBlock(forStmt.Body, exitBlock)
	}

	// Create increment block
	incBlock := b.newBlock(LabelForInc, condBlock.ID)
	b.addEdge(b.currentBlock, incBlock)
	if forStmt.Increment != nil {
		// Store increment as an expression statement
		incBlock.Instructions = append(incBlock.Instructions, &zsm.SemExpressionStmt{
			Expression: forStmt.Increment,
		})
	}

	// Loop back to condition
	b.addEdge(incBlock, condBlock)

	// Create exit block (for loop exit)
	loopExitBlock := b.newBlock(LabelForExit, condBlock.ID)
	b.addEdge(condBlock, loopExitBlock)

	// Continue from loop exit
	b.currentBlock = loopExitBlock
}

// processSelect processes a select statement, creating blocks for each case
func (b *CFGBuilder) processSelect(selectStmt *zsm.SemSelect, exitBlock *BasicBlock) {
	// Current block evaluates the select expression
	exprBlock := b.currentBlock
	exprBlock.Instructions = append(exprBlock.Instructions, selectStmt)

	// Create merge block (where all cases converge)
	mergeBlock := b.newBlock(LabelSelectMerge, exprBlock.ID)

	// Process each case
	for _, caseStmt := range selectStmt.Cases {
		caseBlock := b.newBlock(LabelSelectCase, exprBlock.ID)
		b.addEdge(exprBlock, caseBlock)
		b.currentBlock = caseBlock
		b.processBlock(caseStmt.Body, exitBlock)
		b.addEdge(b.currentBlock, mergeBlock)
	}

	// Process else block if present
	if selectStmt.Else != nil {
		elseBlock := b.newBlock(LabelSelectElse, exprBlock.ID)
		b.addEdge(exprBlock, elseBlock)
		b.currentBlock = elseBlock
		b.processBlock(selectStmt.Else, exitBlock)
		b.addEdge(b.currentBlock, mergeBlock)
	} else {
		// If no else, fall through to merge
		b.addEdge(exprBlock, mergeBlock)
	}

	// Continue from merge block
	b.currentBlock = mergeBlock
}

// ============================================================================
// CFG Utilities
// ============================================================================

// String returns a string representation of the CFG
func (cfg *CFG) String() string {
	var sb strings.Builder
	sb.WriteString("CFG:\n")
	for _, block := range cfg.Blocks {
		sb.WriteString(fmt.Sprintf("  Block %d (%s):\n", block.ID, block.GetFullLabel()))
		sb.WriteString(fmt.Sprintf("    IR Instructions: %d\n", len(block.Instructions)))
		sb.WriteString(fmt.Sprintf("    Machine Instructions: %d\n", len(block.MachineInstructions)))
		sb.WriteString("    Successors: ")
		for _, succ := range block.Successors {
			sb.WriteString(fmt.Sprintf("%d ", succ.ID))
		}
		sb.WriteString("\n")
		sb.WriteString("    Predecessors: ")
		for _, pred := range block.Predecessors {
			sb.WriteString(fmt.Sprintf("%d ", pred.ID))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
