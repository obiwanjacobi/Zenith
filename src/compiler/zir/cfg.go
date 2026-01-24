package zir

import (
	"fmt"
)

// ============================================================================
// Control Flow Graph (CFG) Model
// ============================================================================

// BasicBlock represents a sequence of instructions with one entry and one exit
type BasicBlock struct {
	ID           int           // Unique identifier
	Label        string        // Optional label for this block
	Instructions []IRStatement // Statements in this block
	Successors   []*BasicBlock // Blocks that can follow this one
	Predecessors []*BasicBlock // Blocks that can jump to this one
}

// CFG represents a control flow graph for a function
type CFG struct {
	Entry  *BasicBlock   // Entry block
	Exit   *BasicBlock   // Exit block (for return statements)
	Blocks []*BasicBlock // All blocks in the graph
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
func (b *CFGBuilder) BuildCFG(funcDecl *IRFunctionDecl) *CFG {
	// Create entry block
	entry := b.newBlock("entry", -1)
	b.currentBlock = entry

	// Create exit block (for returns)
	exit := b.newBlock("exit", -1)

	// Process function body
	if funcDecl.Body != nil {
		b.processBlock(funcDecl.Body, exit)
	}

	// Connect current block to exit if it doesn't already have successors
	if len(b.currentBlock.Successors) == 0 {
		b.addEdge(b.currentBlock, exit)
	}

	return &CFG{
		Entry:  entry,
		Exit:   exit,
		Blocks: b.blocks,
	}
}

// newBlock creates a new basic block
// If referenceID >= 0, appends it to the label (e.g., "if.then" + 5 = "if.then.5")
func (b *CFGBuilder) newBlock(label string, referenceID int) *BasicBlock {
	finalLabel := label
	if referenceID >= 0 {
		finalLabel = fmt.Sprintf("%s.%d", label, referenceID)
	}
	block := &BasicBlock{
		ID:           b.nextBlockID,
		Label:        finalLabel,
		Instructions: []IRStatement{},
		Successors:   []*BasicBlock{},
		Predecessors: []*BasicBlock{},
	}
	b.nextBlockID++
	b.blocks = append(b.blocks, block)
	return block
}

// addEdge adds a control flow edge between two blocks
func (b *CFGBuilder) addEdge(from, to *BasicBlock) {
	from.Successors = append(from.Successors, to)
	to.Predecessors = append(to.Predecessors, from)
}

// processBlock processes an IR block and builds CFG blocks
func (b *CFGBuilder) processBlock(block *IRBlock, exitBlock *BasicBlock) {
	for _, stmt := range block.Statements {
		b.processStatement(stmt, exitBlock)
	}
}

// processStatement processes a single IR statement
func (b *CFGBuilder) processStatement(stmt IRStatement, exitBlock *BasicBlock) {
	switch s := stmt.(type) {
	case *IRVariableDecl:
		// Variable declarations are simple statements
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, s)

	case *IRAssignment:
		// Assignments are simple statements
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, s)

	case *IRExpressionStmt:
		// Expression statements (e.g., function calls)
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, s)

	case *IRIf:
		b.processIf(s, exitBlock)

	case *IRFor:
		b.processFor(s, exitBlock)

	case *IRSelect:
		b.processSelect(s, exitBlock)

	default:
		// Unknown statement type - add it anyway
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, stmt)
	}
}

// processIf processes an if statement, creating blocks for branches
func (b *CFGBuilder) processIf(ifStmt *IRIf, exitBlock *BasicBlock) {
	// Current block evaluates condition and branches
	condBlock := b.currentBlock
	condBlock.Instructions = append(condBlock.Instructions, ifStmt)

	// Create then block
	thenBlock := b.newBlock("if.then", condBlock.ID)
	b.addEdge(condBlock, thenBlock)
	b.currentBlock = thenBlock
	b.processBlock(ifStmt.ThenBlock, exitBlock)
	thenExitBlock := b.currentBlock

	// Create merge block (where all branches converge)
	mergeBlock := b.newBlock("if.merge", condBlock.ID)

	// Process elsif blocks
	var elsifExitBlocks []*BasicBlock
	prevCondBlock := condBlock
	for i, elsif := range ifStmt.ElsifBlocks {
		elsifCondBlock := b.newBlock(fmt.Sprintf("elsif.%d.cond", i), prevCondBlock.ID)
		b.addEdge(prevCondBlock, elsifCondBlock)
		elsifCondBlock.Instructions = append(elsifCondBlock.Instructions, elsif)

		elsifThenBlock := b.newBlock(fmt.Sprintf("elsif.%d.then", i), elsifCondBlock.ID)
		b.addEdge(elsifCondBlock, elsifThenBlock)
		b.currentBlock = elsifThenBlock
		b.processBlock(elsif.ThenBlock, exitBlock)
		elsifExitBlocks = append(elsifExitBlocks, b.currentBlock)

		prevCondBlock = elsifCondBlock
	}

	// Process else block if present
	var elseExitBlock *BasicBlock
	if ifStmt.ElseBlock != nil {
		elseBlock := b.newBlock("if.else", prevCondBlock.ID)
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
func (b *CFGBuilder) processFor(forStmt *IRFor, exitBlock *BasicBlock) {
	// Process initializer in current block
	if forStmt.Initializer != nil {
		b.currentBlock.Instructions = append(b.currentBlock.Instructions, forStmt.Initializer)
	}

	// Create condition block
	initBlock := b.currentBlock
	condBlock := b.newBlock("for.cond", initBlock.ID)
	b.addEdge(b.currentBlock, condBlock)
	condBlock.Instructions = append(condBlock.Instructions, forStmt)

	// Create body block
	bodyBlock := b.newBlock("for.body", condBlock.ID)
	b.addEdge(condBlock, bodyBlock)
	b.currentBlock = bodyBlock
	if forStmt.Body != nil {
		b.processBlock(forStmt.Body, exitBlock)
	}

	// Create increment block
	incBlock := b.newBlock("for.inc", condBlock.ID)
	b.addEdge(b.currentBlock, incBlock)
	if forStmt.Increment != nil {
		// Store increment as an expression statement
		incBlock.Instructions = append(incBlock.Instructions, &IRExpressionStmt{
			Expression: forStmt.Increment,
		})
	}

	// Loop back to condition
	b.addEdge(incBlock, condBlock)

	// Create exit block (for loop exit)
	loopExitBlock := b.newBlock("for.exit", condBlock.ID)
	b.addEdge(condBlock, loopExitBlock)

	// Continue from loop exit
	b.currentBlock = loopExitBlock
}

// processSelect processes a select statement, creating blocks for each case
func (b *CFGBuilder) processSelect(selectStmt *IRSelect, exitBlock *BasicBlock) {
	// Current block evaluates the select expression
	exprBlock := b.currentBlock
	exprBlock.Instructions = append(exprBlock.Instructions, selectStmt)

	// Create merge block (where all cases converge)
	mergeBlock := b.newBlock("select.merge", exprBlock.ID)

	// Process each case
	for i, caseStmt := range selectStmt.Cases {
		caseBlock := b.newBlock(fmt.Sprintf("select.case.%d", i), exprBlock.ID)
		b.addEdge(exprBlock, caseBlock)
		b.currentBlock = caseBlock
		b.processBlock(caseStmt.Body, exitBlock)
		b.addEdge(b.currentBlock, mergeBlock)
	}

	// Process else block if present
	if selectStmt.Else != nil {
		elseBlock := b.newBlock("select.else", exprBlock.ID)
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
	result := "CFG:\n"
	for _, block := range cfg.Blocks {
		result += fmt.Sprintf("  Block %d (%s):\n", block.ID, block.Label)
		result += fmt.Sprintf("    Instructions: %d\n", len(block.Instructions))
		result += fmt.Sprintf("    Successors: ")
		for _, succ := range block.Successors {
			result += fmt.Sprintf("%d ", succ.ID)
		}
		result += "\n"
		result += fmt.Sprintf("    Predecessors: ")
		for _, pred := range block.Predecessors {
			result += fmt.Sprintf("%d ", pred.ID)
		}
		result += "\n"
	}
	return result
}
