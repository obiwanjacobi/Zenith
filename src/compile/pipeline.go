package compile

import (
	"fmt"
	"os"
	"strings"

	"zenith/compiler/cfg"
	"zenith/compiler/lexer"
	"zenith/compiler/parser"
	"zenith/compiler/zir"
)

// CompilationResult contains the output of the compilation pipeline
type CompilationResult struct {
	// Source information
	SourceFile string

	// Intermediate representations
	Tokens lexer.TokenStream
	AST    parser.ParserNode
	IR     *zir.SemCompilationUnit

	// Per-function CFG and analysis results
	FunctionCFGs     map[string]*cfg.CFG
	LivenessInfo     map[string]*cfg.LivenessInfo
	InterferenceInfo map[string]*cfg.InterferenceGraph
	AllocationInfo   map[string]*cfg.AllocationResult

	// Machine code
	Instructions map[string][]cfg.MachineInstruction

	// Error tracking
	LexerErrors    []error
	ParserErrors   []parser.ParserError
	SemanticErrors []*zir.SemError
	CodeGenErrors  []error

	// Success flag
	Success bool
}

// PipelineOptions configures the compilation pipeline
type PipelineOptions struct {
	// Source input
	SourceFile string
	SourceCode string

	// Target architecture
	TargetArch string // "z80", etc.

	// Pipeline control flags
	StopAfterLex      bool
	StopAfterParse    bool
	StopAfterSemantic bool
	StopAfterCFG      bool
	StopAfterLiveness bool
	StopAfterRegAlloc bool

	// Optimization flags
	EnableOptimizations bool
	OptimizationLevel   int

	// Debug output
	DumpTokens       bool
	DumpAST          bool
	DumpIR           bool
	DumpCFG          bool
	DumpLiveness     bool
	DumpInterference bool
	DumpAllocation   bool
	DumpInstructions bool
	Verbose          bool
}

// DefaultPipelineOptions returns default pipeline options
func DefaultPipelineOptions() *PipelineOptions {
	return &PipelineOptions{
		TargetArch:          "z80",
		EnableOptimizations: false,
		OptimizationLevel:   0,
		Verbose:             false,
	}
}

// Pipeline runs the complete compilation pipeline
func Pipeline(opts *PipelineOptions) (*CompilationResult, error) {
	result := &CompilationResult{
		SourceFile:       opts.SourceFile,
		FunctionCFGs:     make(map[string]*cfg.CFG),
		LivenessInfo:     make(map[string]*cfg.LivenessInfo),
		InterferenceInfo: make(map[string]*cfg.InterferenceGraph),
		AllocationInfo:   make(map[string]*cfg.AllocationResult),
		Instructions:     make(map[string][]cfg.MachineInstruction),
		Success:          false,
	}

	// ==========================================================================
	// Stage 1: Lexical Analysis (Tokenization)
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 1: Lexical Analysis")
	}

	var tokenizer *lexer.Tokenizer
	// TODO: sourceCode vs sourceFile handling
	if opts.SourceCode != "" {
		// Compile from string
		tokenizer = lexer.TokenizerFromReader(strings.NewReader(opts.SourceCode))
	} else if opts.SourceFile != "" {
		// Compile from file
		file, err := os.Open(opts.SourceFile)
		if err != nil {
			return result, fmt.Errorf("failed to open source file: %w", err)
		}
		defer file.Close()
		tokenizer = lexer.TokenizerFromFile(file)
	} else {
		return result, fmt.Errorf("no source provided")
	}

	tokenChan := tokenizer.Tokens()
	result.Tokens = lexer.NewTokenStream(tokenChan, 100)

	if opts.DumpTokens {
		dumpTokens(result.Tokens)
	}

	if opts.StopAfterLex {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 2: Syntax Analysis (Parsing)
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 2: Syntax Analysis (Parsing)")
	}

	sourceID := opts.SourceFile
	if sourceID == "" {
		sourceID = "<string>"
	}

	astNode, parserErrors := parser.Parse(sourceID, result.Tokens)
	result.AST = astNode
	result.ParserErrors = parserErrors

	if len(parserErrors) > 0 {
		if opts.Verbose {
			fmt.Printf("Parser found %d errors\n", len(parserErrors))
			for _, err := range parserErrors {
				fmt.Printf("  %s\n", err.Error())
			}
		}
		return result, fmt.Errorf("parsing failed with %d errors", len(parserErrors))
	}

	// Ensure AST is a CompilationUnit
	compilationUnit, ok := astNode.(parser.CompilationUnit)
	if !ok {
		return result, fmt.Errorf("parser did not return CompilationUnit")
	}

	if opts.DumpAST {
		dumpAST(compilationUnit)
	}

	if opts.StopAfterParse {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 3: Semantic Analysis & IR Generation
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 3: Semantic Analysis & IR Generation")
	}

	analyzer := zir.NewSemanticAnalyzer()
	semCompilationUnit, semanticErrors := analyzer.Analyze(compilationUnit)
	result.IR = semCompilationUnit
	result.SemanticErrors = semanticErrors

	if len(semanticErrors) > 0 {
		if opts.Verbose {
			fmt.Printf("Semantic analysis found %d errors\n", len(semanticErrors))
			for _, err := range semanticErrors {
				fmt.Printf("  %s\n", err.Error())
			}
		}
		return result, fmt.Errorf("semantic analysis failed with %d errors", len(semanticErrors))
	}

	if opts.DumpIR {
		dumpIR(semCompilationUnit)
	}

	if opts.StopAfterSemantic {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 4: Control Flow Graph Construction
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 4: Control Flow Graph Construction")
	}

	cfgBuilder := cfg.NewCFGBuilder()
	for _, decl := range semCompilationUnit.Declarations {
		if fnDecl, ok := decl.(*zir.SemFunctionDecl); ok {
			functionCFG := cfgBuilder.BuildCFG(fnDecl)
			result.FunctionCFGs[fnDecl.Name] = functionCFG

			if opts.Verbose {
				fmt.Printf("  Built CFG for function '%s' with %d blocks\n", fnDecl.Name, len(functionCFG.Blocks))
			}
		}
	}

	if opts.DumpCFG {
		for fnName, fnCFG := range result.FunctionCFGs {
			dumpCFG(fnName, fnCFG)
		}
	}

	if opts.StopAfterCFG {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 5: Liveness Analysis
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 5: Liveness Analysis")
	}

	for fnName, fnCFG := range result.FunctionCFGs {
		liveness := cfg.ComputeLiveness(fnCFG)
		result.LivenessInfo[fnName] = liveness

		if opts.Verbose {
			fmt.Printf("  Computed liveness for function '%s'\n", fnName)
		}
	}

	if opts.DumpLiveness {
		for fnName, liveness := range result.LivenessInfo {
			dumpLiveness(fnName, liveness)
		}
	}

	if opts.StopAfterLiveness {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 6: Interference Graph Construction
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 6: Interference Graph Construction")
	}

	for fnName, liveness := range result.LivenessInfo {
		fnCFG := result.FunctionCFGs[fnName]
		interference := cfg.BuildInterferenceGraph(fnCFG, liveness)
		result.InterferenceInfo[fnName] = interference

		if opts.Verbose {
			nodes := interference.GetNodes()
			edgeCount := 0
			for _, node := range nodes {
				edgeCount += interference.GetDegree(node)
			}
			edgeCount /= 2 // Each edge counted twice
			fmt.Printf("  Built interference graph for function '%s' with %d nodes, %d edges\n",
				fnName, len(nodes), edgeCount)
		}
	}

	if opts.DumpInterference {
		for fnName, interference := range result.InterferenceInfo {
			dumpInterference(fnName, interference)
		}
	}

	// ==========================================================================
	// Stage 7: Register Allocation
	// ==========================================================================
	// Note: Register allocation happens during instruction selection using
	// virtual registers and the register allocator
	if opts.StopAfterRegAlloc {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 8: Instruction Selection
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 8: Instruction Selection")
	}

	// Get architecture-specific calling convention and instruction selector
	var callingConvention cfg.CallingConvention
	var instructionSelector cfg.InstructionSelector

	switch opts.TargetArch {
	case "z80":
		callingConvention = cfg.NewCallingConvention_Z80()
		instructionSelector = cfg.NewZ80InstructionSelector(callingConvention)
	default:
		return result, fmt.Errorf("unsupported target architecture: %s", opts.TargetArch)
	}

	// Run instruction selection on the IR
	err := cfg.SelectInstructions(semCompilationUnit, instructionSelector, callingConvention)
	if err != nil {
		result.CodeGenErrors = append(result.CodeGenErrors, err)
		return result, fmt.Errorf("instruction selection failed: %w", err)
	}

	// Extract instructions per function
	// Note: The instruction selector accumulates all instructions
	// We need to segment them by function
	allInstructions := instructionSelector.GetInstructions()
	result.Instructions["<all>"] = allInstructions

	if opts.Verbose {
		fmt.Printf("  Generated %d machine instructions\n", len(allInstructions))
	}

	if opts.DumpInstructions {
		dumpInstructions(allInstructions)
	}

	// ==========================================================================
	// Pipeline Complete
	// ==========================================================================
	result.Success = true
	return result, nil
}

// =============================================================================
// Debug Dump Functions
// =============================================================================

func dumpTokens(tokens lexer.TokenStream) {
	fmt.Println("========== TOKENS ==========")
	mark := tokens.Mark()
	for {
		tok := tokens.Peek()
		if tok == nil || tok.Id() == lexer.TokenEOF {
			break
		}
		tokens.Read()
		fmt.Printf("  %v: %s\n", tok.Id(), tok.Text())
	}
	tokens.GotoMark(mark)
	fmt.Println()
}

func dumpAST(ast parser.CompilationUnit) {
	fmt.Println("========== AST ==========")
	fmt.Printf("Compilation Unit with %d declarations\n", len(ast.Declarations()))
	for i, decl := range ast.Declarations() {
		fmt.Printf("  [%d] %T\n", i, decl)
	}
	fmt.Println()
}

func dumpIR(ir *zir.SemCompilationUnit) {
	fmt.Println("========== IR ===========")
	fmt.Printf("IR Compilation Unit with %d declarations\n", len(ir.Declarations))
	for _, decl := range ir.Declarations {
		switch d := decl.(type) {
		case *zir.SemFunctionDecl:
			fmt.Printf("  Function: %s (params=%d)\n",
				d.Name, len(d.Parameters))
		case *zir.SemVariableDecl:
			fmt.Printf("  Variable: %s\n", d.Symbol.Name)
		case *zir.SemTypeDecl:
			fmt.Printf("  Type: %s\n", d.TypeInfo.Name())
		default:
			fmt.Printf("  Unknown: %T\n", decl)
		}
	}
	fmt.Println()
}

func dumpCFG(fnName string, fnCFG *cfg.CFG) {
	fmt.Printf("========== CFG: %s ==========\n", fnName)
	fmt.Printf("Entry: Block %d\n", fnCFG.Entry.ID)
	fmt.Printf("Exit:  Block %d\n", fnCFG.Exit.ID)
	fmt.Printf("Blocks: %d\n", len(fnCFG.Blocks))
	for _, block := range fnCFG.Blocks {
		fmt.Printf("  Block %d [%s]: %d instructions, %d successors\n",
			block.ID, block.Label, len(block.Instructions), len(block.Successors))
	}
	fmt.Println()
}

func dumpLiveness(fnName string, liveness *cfg.LivenessInfo) {
	fmt.Printf("========== LIVENESS: %s ==========\n", fnName)
	for blockID, liveIn := range liveness.LiveIn {
		fmt.Printf("  Block %d:\n", blockID)
		fmt.Printf("    LiveIn:  %v\n", setToSlice(liveIn))
		fmt.Printf("    LiveOut: %v\n", setToSlice(liveness.LiveOut[blockID]))
	}
	fmt.Println()
}

func dumpInterference(fnName string, interference *cfg.InterferenceGraph) {
	fmt.Printf("========== INTERFERENCE: %s ==========\n", fnName)
	nodes := interference.GetNodes()
	edgeCount := 0
	for _, node := range nodes {
		edgeCount += interference.GetDegree(node)
	}
	edgeCount /= 2 // Each edge counted twice
	fmt.Printf("Nodes: %d\n", len(nodes))
	fmt.Printf("Edges: %d\n", edgeCount)
	for _, varName := range nodes {
		neighbors := interference.GetNeighbors(varName)
		if len(neighbors) > 0 {
			fmt.Printf("  %s interferes with: %v\n", varName, neighbors)
		}
	}
	fmt.Println()
}

func dumpAllocation(fnName string, allocation *cfg.AllocationResult) {
	fmt.Printf("========== ALLOCATION: %s ==========\n", fnName)
	fmt.Printf("Register allocation complete\n")
	// Note: AllocationResult fields are not exported
	fmt.Println()
}

func dumpInstructions(instructions []cfg.MachineInstruction) {
	fmt.Println("========== INSTRUCTIONS ==========")
	for i, instr := range instructions {
		result := instr.GetResult()
		operands := instr.GetOperands()
		resultStr := "<none>"
		if result != nil {
			resultStr = result.Name
		}
		operandStrs := make([]string, len(operands))
		for j, op := range operands {
			operandStrs[j] = op.Name
		}
		fmt.Printf("  [%4d] %s (result=%s, operands=%v)\n",
			i, instr.String(), resultStr, operandStrs)
	}
	fmt.Println()
}

// Helper function to convert map[string]bool to []string
func setToSlice(set map[string]bool) []string {
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	return result
}
