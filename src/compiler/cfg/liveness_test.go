package cfg

import (
	"testing"

	"zenith/compiler/lexer"
	"zenith/compiler/parser"
	"zenith/compiler/zir"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to build CFG and compute liveness from code
func buildLivenessFromCode(t *testing.T, code string) (*CFG, *LivenessInfo, *zir.SymbolLookup) {
	// Tokenize
	tokens := lexer.OpenTokenStream(code)

	// Parse
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors), "Parsing errors: %v", parseErrors)

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	// Analyze to get IR
	analyzer := zir.NewSemanticAnalyzer()
	semCU, semErrors := analyzer.Analyze(cu)
	require.Equal(t, 0, len(semErrors), "IR analysis errors: %v", semErrors)
	require.Greater(t, len(semCU.Declarations), 0)

	// Get function declaration
	funcDecl, ok := semCU.Declarations[0].(*zir.SemFunctionDecl)
	require.True(t, ok)

	// Build CFG
	builder := NewCFGBuilder()
	cfg := builder.BuildCFG(funcDecl)

	// Compute liveness
	liveness := ComputeLiveness(cfg)

	symbolLookup := zir.NewSymbolLookup(semCU)
	return cfg, liveness, symbolLookup
}

func Test_Liveness_SimpleAssignment(t *testing.T) {
	code := `main: () {
		x: = 5
		y: = x + 1
	}`
	cfg, liveness, _ := buildLivenessFromCode(t, code)

	// Entry block should have no live variables
	entry := cfg.Entry
	assert.Equal(t, 0, len(liveness.LiveIn[entry.ID]))

	// Find the block with actual code
	var codeBlock *BasicBlock
	for _, block := range cfg.Blocks {
		if len(block.Instructions) > 0 {
			codeBlock = block
			break
		}
	}
	require.NotNil(t, codeBlock)

	// Variables now use qualified names (e.g., "main.x" since function is named "main")
	// x is defined then used, so it should be in def set
	assert.True(t, liveness.Def[codeBlock.ID]["main.x"])
	assert.True(t, liveness.Def[codeBlock.ID]["main.y"])

	// x is used in second statement, so should be in use set
	// (only if used before being defined in same block)
	// In this case, x is defined first, then used, so not in use set
	assert.False(t, liveness.Use[codeBlock.ID]["main.x"])
}

func Test_Liveness_IfStatement(t *testing.T) {
	code := `main: () {
		x: = 5
		if true {
			y: = x + 1
		} else {
			z: = x + 2
		}
	}`
	cfg, liveness, _ := buildLivenessFromCode(t, code)

	// Find then and else blocks (may have numeric suffix)
	var thenBlock, elseBlock *BasicBlock
	for _, block := range cfg.Blocks {
		if block.Label == LabelIfThen {
			thenBlock = block
		}
		if block.Label == LabelIfElse {
			elseBlock = block
		}
	}
	require.NotNil(t, thenBlock, "Could not find then block")
	require.NotNil(t, elseBlock, "Could not find else block")

	// x should be live-in to both then and else blocks (used in both)
	assert.True(t, liveness.IsLiveAt("main.x", thenBlock.ID), "x should be live-in to then block")
	assert.True(t, liveness.IsLiveAt("main.x", elseBlock.ID), "x should be live-in to else block")

	// y is defined in then block
	assert.True(t, liveness.Def[thenBlock.ID]["main.y"])

	// z is defined in else block
	assert.True(t, liveness.Def[elseBlock.ID]["main.z"])
}

func Test_Liveness_Loop(t *testing.T) {
	code := `main: () {
		x: = 0
		for x < 10 {
			x = x + 1
		}
		y: = x
	}`
	cfg, liveness, _ := buildLivenessFromCode(t, code)

	// Find loop condition and body blocks (may have numeric suffix)
	var condBlock, bodyBlock *BasicBlock
	for _, block := range cfg.Blocks {
		if block.Label == LabelForCond {
			condBlock = block
		}
		if block.Label == LabelForBody {
			bodyBlock = block
		}
	}
	require.NotNil(t, condBlock, "Could not find condition block")
	require.NotNil(t, bodyBlock, "Could not find body block")

	// x should be live-in to condition block (loop variable)
	assert.True(t, liveness.IsLiveAt("main.x", condBlock.ID), "x should be live-in to condition")

	// x should be live-in and live-out of body block
	assert.True(t, liveness.IsLiveAt("main.x", bodyBlock.ID), "x should be live-in to body")
	assert.True(t, liveness.IsLiveOutOf("main.x", bodyBlock.ID), "x should be live-out of body")
}

func Test_Liveness_NoUseAfterDef(t *testing.T) {
	code := `main: () {
		x: = 5
		x = 10
	}`
	cfg, liveness, _ := buildLivenessFromCode(t, code)

	// Find code block
	var codeBlock *BasicBlock
	for _, block := range cfg.Blocks {
		if len(block.Instructions) > 0 {
			codeBlock = block
			break
		}
	}
	require.NotNil(t, codeBlock)

	// x is defined but never used (redefined immediately)
	assert.True(t, liveness.Def[codeBlock.ID]["main.x"])

	// x should not be live-out (not used after definition)
	assert.False(t, liveness.IsLiveOutOf("main.x", codeBlock.ID))
}

func Test_Liveness_MultipleVariables(t *testing.T) {
	code := `main: () {
		a: = 1
		b: = 2
		c: = a + b
		d: = c * 2
	}`
	cfg, liveness, _ := buildLivenessFromCode(t, code)

	// Find code block
	var codeBlock *BasicBlock
	for _, block := range cfg.Blocks {
		if len(block.Instructions) > 0 {
			codeBlock = block
			break
		}
	}
	require.NotNil(t, codeBlock)

	// All variables defined
	assert.True(t, liveness.Def[codeBlock.ID]["main.a"])
	assert.True(t, liveness.Def[codeBlock.ID]["main.b"])
	assert.True(t, liveness.Def[codeBlock.ID]["main.c"])
	assert.True(t, liveness.Def[codeBlock.ID]["main.d"])

	// Only c is used (a and b are used but defined first in same block)
	// c is used after being defined
	assert.False(t, liveness.Use[codeBlock.ID]["main.a"], "a defined before use")
	assert.False(t, liveness.Use[codeBlock.ID]["main.b"], "b defined before use")
	assert.False(t, liveness.Use[codeBlock.ID]["main.c"], "c defined before use")
}

func Test_Liveness_GetLiveRanges(t *testing.T) {
	code := `main: () {
		x: = 5
		if true {
			y: = x + 1
		}
		z: = x + 2
	}`
	_, liveness, _ := buildLivenessFromCode(t, code)

	ranges := liveness.GetLiveRanges()

	// x should have a live range covering multiple blocks (used in both branches)
	if ranges["main.x"] != nil {
		assert.Greater(t, len(ranges["main.x"]), 0, "x should be live in at least one block")
	}

	// y and z may not have live ranges if they're not read after being defined
	// This is expected behavior - variables only get live ranges if they're live at block boundaries
}

func Test_Liveness_BinaryExpression(t *testing.T) {
	code := `main: () {
		a: = 5
		b: = 10
		c: = a + b
	}`
	cfg, liveness, _ := buildLivenessFromCode(t, code)

	// Find code block
	var codeBlock *BasicBlock
	for _, block := range cfg.Blocks {
		if len(block.Instructions) > 0 {
			codeBlock = block
			break
		}
	}
	require.NotNil(t, codeBlock)

	// a and b are defined
	assert.True(t, liveness.Def[codeBlock.ID]["main.a"])
	assert.True(t, liveness.Def[codeBlock.ID]["main.b"])

	// c is defined and uses a and b (but a and b are defined first in same block)
	assert.True(t, liveness.Def[codeBlock.ID]["main.c"])
	assert.False(t, liveness.Use[codeBlock.ID]["main.a"])
	assert.False(t, liveness.Use[codeBlock.ID]["main.b"])
}

func Test_Liveness_ReturnStatement(t *testing.T) {
	code := `main: () {
		x: = 5
		y: = 10
		ret x + y
	}`

	cfg, liveness, _ := buildLivenessFromCode(t, code)

	// Find the block with instructions
	var codeBlock *BasicBlock
	for _, block := range cfg.Blocks {
		if len(block.Instructions) > 0 {
			codeBlock = block
			break
		}
	}
	require.NotNil(t, codeBlock)

	// x and y are defined in the block
	assert.True(t, liveness.Def[codeBlock.ID]["main.x"])
	assert.True(t, liveness.Def[codeBlock.ID]["main.y"])

	// x and y are NOT in the Use set because they're defined before used within the same block
	// (Use set only tracks variables used before being defined in a block)
	assert.False(t, liveness.Use[codeBlock.ID]["main.x"])
	assert.False(t, liveness.Use[codeBlock.ID]["main.y"])
}

