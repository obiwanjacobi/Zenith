package zir

import (
	"testing"

	"zenith/compiler/lexer"
	"zenith/compiler/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to build a CFG from code
func buildCFGFromCode(t *testing.T, code string) *CFG {
	// Tokenize
	tokens := lexer.OpenTokenStream(code)

	// Parse
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors))

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	// Analyze to get IR
	analyzer := NewSemanticAnalyzer()
	irCU, irErrors := analyzer.Analyze(cu)
	if len(irErrors) > 0 {
		t.Logf("IR errors: %v", irErrors)
	}
	require.Equal(t, 0, len(irErrors))
	require.Greater(t, len(irCU.Declarations), 0)

	// Get function declaration
	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
	require.True(t, ok)

	// Build CFG
	builder := NewCFGBuilder()
	cfg := builder.BuildCFG(funcDecl)
	require.NotNil(t, cfg)

	return cfg
}

// Helper to find a block by label prefix
func findBlockByLabel(cfg *CFG, labelPrefix string) *BasicBlock {
	for _, block := range cfg.Blocks {
		if len(block.Label) >= len(labelPrefix) && block.Label[:len(labelPrefix)] == labelPrefix {
			return block
		}
	}
	return nil
}

// ============================================================================
// Basic CFG Tests
// ============================================================================

func Test_CFG_EmptyFunction(t *testing.T) {
	code := `main: () {
	}`
	cfg := buildCFGFromCode(t, code)

	// Should have entry and exit blocks
	assert.NotNil(t, cfg.Entry)
	assert.NotNil(t, cfg.Exit)
	assert.Equal(t, "entry", cfg.Entry.Label)
	assert.Equal(t, "exit", cfg.Exit.Label)

	// Entry should connect to exit
	assert.Equal(t, 1, len(cfg.Entry.Successors))
	assert.Equal(t, cfg.Exit, cfg.Entry.Successors[0])

	// Exit should have entry as predecessor
	assert.Equal(t, 1, len(cfg.Exit.Predecessors))
	assert.Equal(t, cfg.Entry, cfg.Exit.Predecessors[0])
}

func Test_CFG_SimpleStatements(t *testing.T) {
	code := `main: () {
		x: = 5
		y: = 10
		z: = x + y
	}`
	cfg := buildCFGFromCode(t, code)

	// All statements should be in the entry block
	assert.Equal(t, 3, len(cfg.Entry.Instructions))

	// Entry connects to exit
	assert.Equal(t, 1, len(cfg.Entry.Successors))
	assert.Equal(t, cfg.Exit, cfg.Entry.Successors[0])
}

// ============================================================================
// If Statement Tests
// ============================================================================

func Test_CFG_IfStatement(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Should have: entry, if.then, if.merge, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 4)

	// Find blocks
	thenBlock := findBlockByLabel(cfg, "if.then")
	mergeBlock := findBlockByLabel(cfg, "if.merge")

	require.NotNil(t, thenBlock, "Should have if.then block")
	require.NotNil(t, mergeBlock, "Should have if.merge block")

	// Entry should have if statement
	assert.Equal(t, 1, len(cfg.Entry.Instructions))

	// Then block should have 1 instruction
	assert.Equal(t, 1, len(thenBlock.Instructions))

	// Merge block should connect to exit
	assert.Equal(t, 1, len(mergeBlock.Successors))
	assert.Equal(t, cfg.Exit, mergeBlock.Successors[0])
}

func Test_CFG_IfElseStatement(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		} else {
			y: = 2
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Should have: entry, if.then, if.else, if.merge, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 5)

	// Find blocks
	thenBlock := findBlockByLabel(cfg, "if.then")
	elseBlock := findBlockByLabel(cfg, "if.else")
	mergeBlock := findBlockByLabel(cfg, "if.merge")

	require.NotNil(t, thenBlock)
	require.NotNil(t, elseBlock)
	require.NotNil(t, mergeBlock)

	// Then and else blocks should each have 1 instruction
	assert.Equal(t, 1, len(thenBlock.Instructions))
	assert.Equal(t, 1, len(elseBlock.Instructions))

	// Both then and else should connect to merge
	assert.Contains(t, mergeBlock.Predecessors, thenBlock)
	assert.Contains(t, mergeBlock.Predecessors, elseBlock)
}

func Test_CFG_IfElsifElseStatement(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		} elsif false {
			y: = 2
		} else {
			z: = 3
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Should have: entry, if.then, elsif.0.cond, elsif.0.then, if.else, if.merge, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 7)

	// Find blocks
	thenBlock := findBlockByLabel(cfg, "if.then")
	elsifCondBlock := findBlockByLabel(cfg, "elsif.0.cond")
	elsifThenBlock := findBlockByLabel(cfg, "elsif.0.then")
	elseBlock := findBlockByLabel(cfg, "if.else")
	mergeBlock := findBlockByLabel(cfg, "if.merge")

	require.NotNil(t, thenBlock)
	require.NotNil(t, elsifCondBlock)
	require.NotNil(t, elsifThenBlock)
	require.NotNil(t, elseBlock)
	require.NotNil(t, mergeBlock)

	// All branches should connect to merge
	assert.Contains(t, mergeBlock.Predecessors, thenBlock)
	assert.Contains(t, mergeBlock.Predecessors, elsifThenBlock)
	assert.Contains(t, mergeBlock.Predecessors, elseBlock)
}

// ============================================================================
// For Loop Tests
// ============================================================================

func Test_CFG_ForLoop(t *testing.T) {
	code := `main: () {
		for i: = 0; i < 10; i + 1 {
			x: = i
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Should have: entry, for.cond, for.body, for.inc, for.exit, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 6)

	// Find blocks
	condBlock := findBlockByLabel(cfg, "for.cond")
	bodyBlock := findBlockByLabel(cfg, "for.body")
	incBlock := findBlockByLabel(cfg, "for.inc")
	exitBlock := findBlockByLabel(cfg, "for.exit")

	require.NotNil(t, condBlock)
	require.NotNil(t, bodyBlock)
	require.NotNil(t, incBlock)
	require.NotNil(t, exitBlock)

	// Condition should branch to body and exit
	assert.Contains(t, condBlock.Successors, bodyBlock)
	assert.Contains(t, condBlock.Successors, exitBlock)

	// Body should connect to increment
	assert.Contains(t, bodyBlock.Successors, incBlock)

	// Increment should loop back to condition
	assert.Contains(t, incBlock.Successors, condBlock)

	// Condition should have increment as predecessor (loop back edge)
	assert.Contains(t, condBlock.Predecessors, incBlock)
}

func Test_CFG_ForLoopOnlyCondition(t *testing.T) {
	code := `main: () {
		for true {
			x: = 1
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Should still have loop structure
	condBlock := findBlockByLabel(cfg, "for.cond")
	bodyBlock := findBlockByLabel(cfg, "for.body")
	incBlock := findBlockByLabel(cfg, "for.inc")

	require.NotNil(t, condBlock)
	require.NotNil(t, bodyBlock)
	require.NotNil(t, incBlock)

	// Increment should loop back even if empty
	assert.Contains(t, incBlock.Successors, condBlock)
}

// ============================================================================
// Select Statement Tests
// ============================================================================

func Test_CFG_SelectStatement(t *testing.T) {
	code := `main: () {
		x: = 5
		select x {
			case 1 {
				a: = 10
			}
			case 2 {
				b: = 20
			}
			else {
				c: = 30
			}
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Should have: entry, select.case.0, select.case.1, select.else, select.merge, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 6)

	// Find blocks
	case0Block := findBlockByLabel(cfg, "select.case.0")
	case1Block := findBlockByLabel(cfg, "select.case.1")
	elseBlock := findBlockByLabel(cfg, "select.else")
	mergeBlock := findBlockByLabel(cfg, "select.merge")

	require.NotNil(t, case0Block)
	require.NotNil(t, case1Block)
	require.NotNil(t, elseBlock)
	require.NotNil(t, mergeBlock)

	// All cases should connect to merge
	assert.Contains(t, mergeBlock.Predecessors, case0Block)
	assert.Contains(t, mergeBlock.Predecessors, case1Block)
	assert.Contains(t, mergeBlock.Predecessors, elseBlock)
}

func Test_CFG_SelectStatementNoElse(t *testing.T) {
	code := `main: () {
		x: = 5
		select x {
			case 1 {
				a: = 10
			}
			case 2 {
				b: = 20
			}
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Find blocks
	case0Block := findBlockByLabel(cfg, "select.case.0")
	case1Block := findBlockByLabel(cfg, "select.case.1")
	mergeBlock := findBlockByLabel(cfg, "select.merge")

	require.NotNil(t, case0Block)
	require.NotNil(t, case1Block)
	require.NotNil(t, mergeBlock)

	// Cases should connect to merge
	assert.Contains(t, mergeBlock.Predecessors, case0Block)
	assert.Contains(t, mergeBlock.Predecessors, case1Block)

	// Entry (with select expression) should also connect to merge (fall-through)
	assert.GreaterOrEqual(t, len(mergeBlock.Predecessors), 2)
}

// ============================================================================
// Complex CFG Tests
// ============================================================================

func Test_CFG_NestedIfInFor(t *testing.T) {
	code := `main: () {
		for i: = 0; i < 10; i + 1 {
			if i < 5 {
				x: = i
			}
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Should have loop structure with if inside body
	forBodyBlock := findBlockByLabel(cfg, "for.body")
	require.NotNil(t, forBodyBlock)

	// Body should have the if statement
	assert.GreaterOrEqual(t, len(forBodyBlock.Instructions), 1)
}

func Test_CFG_BlockLabelsUnique(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		}
		if false {
			y: = 2
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Check that all block labels are unique
	labelCount := make(map[string]int)
	for _, block := range cfg.Blocks {
		labelCount[block.Label]++
	}

	for label, count := range labelCount {
		assert.Equal(t, 1, count, "Label '%s' should be unique", label)
	}
}

func Test_CFG_BlockIDsUnique(t *testing.T) {
	code := `main: () {
		x: = 1
		if true {
			y: = 2
		}
		for i: = 0; i < 10; i + 1 {
			z: = i
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// Check that all block IDs are unique
	idCount := make(map[int]int)
	for _, block := range cfg.Blocks {
		idCount[block.ID]++
	}

	for id, count := range idCount {
		assert.Equal(t, 1, count, "ID %d should be unique", id)
	}

	// IDs should be sequential starting from 0
	assert.Equal(t, 0, cfg.Entry.ID)
	assert.Equal(t, len(cfg.Blocks)-1, cfg.Blocks[len(cfg.Blocks)-1].ID)
}

// ============================================================================
// Edge Tests
// ============================================================================

func Test_CFG_PredecessorsSuccessorsConsistent(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		} else {
			y: = 2
		}
	}`
	cfg := buildCFGFromCode(t, code)

	// For every edge A -> B, B should have A as predecessor
	for _, block := range cfg.Blocks {
		for _, successor := range block.Successors {
			assert.Contains(t, successor.Predecessors, block,
				"Block %d (%s) has successor %d (%s), but successor doesn't have it as predecessor",
				block.ID, block.Label, successor.ID, successor.Label)
		}
	}

	// For every edge A <- B, A should have B as successor
	for _, block := range cfg.Blocks {
		for _, predecessor := range block.Predecessors {
			assert.Contains(t, predecessor.Successors, block,
				"Block %d (%s) has predecessor %d (%s), but predecessor doesn't have it as successor",
				block.ID, block.Label, predecessor.ID, predecessor.Label)
		}
	}
}
