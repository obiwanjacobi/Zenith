package compile

import (
	"fmt"
	"os"
	"strings"

	"zenith/compiler/cfg"
	"zenith/compiler/lexer"
	"zenith/compiler/parser"
	"zenith/compiler/zsm"
)

// CompilationResult contains the output of the compilation pipeline
type CompilationResult struct {
	// Source information
	SourceFile string

	// Intermediate representations
	Tokens lexer.TokenStream
	AST    parser.ParserNode
	SemCU  *zsm.SemCompilationUnit

	// Per-function CFG and analysis results
	FunctionCFGs     map[string]*cfg.CFG
	LivenessInfo     map[string]*cfg.LivenessInfo
	InterferenceInfo map[string]*cfg.InterferenceGraph
	// Note: Register allocation results are stored in VirtualRegister.PhysicalReg

	// Machine code
	Instructions map[string][]cfg.MachineInstruction

	// Error tracking
	LexerErrors    []error
	ParserErrors   []parser.ParserError
	SemanticErrors []*zsm.SemError
	CodeGenErrors  []error

	// Success flag
	Success bool

	// internals
	VRAllocator       *cfg.VirtualRegisterAllocator
	SelectorForTarget cfg.InstructionSelector
}

// PipelineOptions configures the compilation pipeline
type PipelineOptions struct {
	// Source input
	SourceFile string
	SourceCode string

	// Target architecture
	TargetArch string // "z80", etc.

	// Pipeline control flags
	StopAfterLex                  bool
	StopAfterParse                bool
	StopAfterSemantic             bool
	StopAfterCFG                  bool
	StopAfterInstructionSelection bool
	StopAfterLiveness             bool
	StopAfterInterference         bool
	StopAfterRegAlloc             bool

	// Debug output
	Verbose bool
}

// DefaultPipelineOptions returns default pipeline options
func DefaultPipelineOptions() *PipelineOptions {
	return &PipelineOptions{
		TargetArch: "z80",
		Verbose:    false,
	}
}

// Pipeline runs the complete compilation pipeline
func Pipeline(opts *PipelineOptions) (*CompilationResult, error) {
	result := &CompilationResult{
		SourceFile:       opts.SourceFile,
		FunctionCFGs:     make(map[string]*cfg.CFG),
		LivenessInfo:     make(map[string]*cfg.LivenessInfo),
		InterferenceInfo: make(map[string]*cfg.InterferenceGraph),
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

	analyzer := zsm.NewSemanticAnalyzer()
	semCompilationUnit, semanticErrors := analyzer.Analyze(compilationUnit)
	result.SemCU = semCompilationUnit
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
		if fnDecl, ok := decl.(*zsm.SemFunctionDecl); ok {
			functionCFG := cfgBuilder.BuildCFG(fnDecl)
			result.FunctionCFGs[fnDecl.Name] = functionCFG

			if opts.Verbose {
				fmt.Printf("  Built CFG for function '%s' with %d blocks\n", fnDecl.Name, len(functionCFG.Blocks))
			}
		}
	}

	if opts.StopAfterCFG {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 5: Instruction Selection
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 5: Instruction Selection")
	}

	// Create virtual register allocator (shared across all functions)
	vrAlloc := cfg.NewVirtualRegisterAllocator()
	result.VRAllocator = vrAlloc

	// Collect CFGs from result.FunctionCFGs map into a slice
	cfgs := make([]*cfg.CFG, 0, len(result.FunctionCFGs))
	for _, funcCFG := range result.FunctionCFGs {
		cfgs = append(cfgs, funcCFG)
	}

	// TODO: Allow different selectors based on target architecture
	if opts.TargetArch != "z80" {
		return result, fmt.Errorf("unsupported target architecture: %s", opts.TargetArch)
	}
	selector := cfg.NewInstructionSelectorZ80(vrAlloc)
	result.SelectorForTarget = selector
	// Run instruction selection on the CFGs (modifies CFGs in-place, adds MachineInstructions)
	err := cfg.SelectInstructions(cfgs, vrAlloc, selector)
	if err != nil {
		result.CodeGenErrors = append(result.CodeGenErrors, err)
		return result, fmt.Errorf("instruction selection failed: %w", err)
	}

	if opts.Verbose {
		totalInstrs := 0
		for _, funcCFG := range result.FunctionCFGs {
			totalInstrs += len(funcCFG.GetAllInstructions())
		}
		fmt.Printf("  Generated %d machine instructions with virtual registers\n", totalInstrs)
	}

	if opts.StopAfterInstructionSelection {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 6: Liveness Analysis
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 6: Liveness Analysis")
	}

	for fnName, fnCFG := range result.FunctionCFGs {
		liveness := cfg.ComputeLiveness(fnCFG)
		result.LivenessInfo[fnName] = liveness

		if opts.Verbose {
			fmt.Printf("  Computed liveness for function '%s'\n", fnName)
		}
	}

	if opts.StopAfterLiveness {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 7: Interference Graph Construction
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 7: Interference Graph Construction")
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

	if opts.StopAfterInterference {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 8: Register Allocation
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 8: Register Allocation")
	}

	// Create register allocator with target registers
	allocator := cfg.NewRegisterAllocator(selector.GetTargetRegisters())

	for fnName, fnCFG := range result.FunctionCFGs {
		interference := result.InterferenceInfo[fnName]

		// Run register allocation (assigns PhysicalReg to each VirtualRegister)
		err := allocator.Allocate(fnCFG, interference)
		if err != nil {
			result.CodeGenErrors = append(result.CodeGenErrors, err)
			return result, fmt.Errorf("register allocation failed for %s: %w", fnName, err)
		}

		if opts.Verbose {
			allocated := 0
			spilled := 0
			for _, vr := range vrAlloc.GetAll() {
				switch vr.Type {
				case cfg.AllocatedRegister:
					allocated++
				case cfg.StackLocation:
					spilled++
				}
			}
			fmt.Printf("  Allocated %d registers, spilled %d for function '%s'\n", allocated, spilled, fnName)
		}
	}

	if opts.StopAfterRegAlloc {
		result.Success = true
		return result, nil
	}

	// ==========================================================================
	// Stage 9: Code Generation (emit final instructions)
	// ==========================================================================
	// Extract instructions from each function's CFG with physical registers assigned
	allInstructions := []cfg.MachineInstruction{}
	for _, funcCFG := range result.FunctionCFGs {
		funcInstructions := funcCFG.GetAllInstructions()
		result.Instructions[funcCFG.FunctionName] = funcInstructions
		allInstructions = append(allInstructions, funcInstructions...)
	}
	result.Instructions["<all>"] = allInstructions

	// ==========================================================================
	// Pipeline Complete
	// ==========================================================================
	result.Success = true
	return result, nil
}
