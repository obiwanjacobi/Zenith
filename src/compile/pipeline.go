package compile

import (
	"fmt"
	"strings"

	"zenith/compiler"
	"zenith/compiler/cfg"
	"zenith/compiler/lexer"
	"zenith/compiler/parser"
	"zenith/compiler/zsm"
)

// CompilationResult contains the output of the compilation pipeline
type CompilationResult struct {
	// Intermediate representations
	Tokens lexer.TokenStream
	AST    parser.ParserNode
	SemCU  *zsm.SemCompilationUnit

	// Per-function CFG and analysis results
	FunctionCFGs     map[string]*cfg.CFG
	LivenessInfo     map[string]*cfg.LivenessInfo
	InterferenceInfo map[string]*cfg.InterferenceGraph
	VrAllocators     map[string]*cfg.VirtualRegisterAllocator

	// Error tracking
	Diagnostics    []*compiler.Diagnostic
	LexerErrors    []error
	SemanticErrors []*compiler.Diagnostic
	CodeGenErrors  []error

	// Success flag
	Success bool
}

// PipelineOptions configures the compilation pipeline
type PipelineOptions struct {
	// for now...
	Source string

	// Target architecture
	TargetArch string // "z80", etc.

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
		FunctionCFGs:     make(map[string]*cfg.CFG),
		LivenessInfo:     make(map[string]*cfg.LivenessInfo),
		InterferenceInfo: make(map[string]*cfg.InterferenceGraph),
		VrAllocators:     make(map[string]*cfg.VirtualRegisterAllocator),
		Success:          false,
	}

	// ==========================================================================
	// Stage 1: Lexical Analysis (Tokenization)
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 1: Lexical Analysis")
	}

	var tokenizer *lexer.Tokenizer

	if opts.Source != "" {
		tokenizer = lexer.TokenizerFromReader(strings.NewReader(opts.Source))
	} else {
		return result, fmt.Errorf("no source provided")
	}

	tokenChan := tokenizer.Tokens()
	result.Tokens = lexer.NewTokenStream(tokenChan, 100)

	// ==========================================================================
	// Stage 2: Syntax Analysis (Parsing)
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 2: Syntax Analysis (Parsing)")
	}

	source := &compiler.Source{Name: "pipeline_input"}
	astNode, parserErrors := parser.Parse(source, result.Tokens)
	result.AST = astNode
	result.Diagnostics = append(result.Diagnostics, parserErrors...)

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

	// ==========================================================================
	// Stage 5: Instruction Selection
	// ==========================================================================
	if opts.Verbose {
		fmt.Println("==> Stage 5: Instruction Selection")
	}

	// TODO: Allow different selectors based on target architecture
	if opts.TargetArch != "z80" {
		return result, fmt.Errorf("unsupported target architecture: %s", opts.TargetArch)
	}

	// per function processing
	for _, funcCFG := range result.FunctionCFGs {
		symbolContext := make(map[string]cfg.VirtualRegisterType)

		allocationRetryCount := 0
		allocationSucceeded := false
		// retry loop for register allocation
		for ; !allocationSucceeded && allocationRetryCount < 10; allocationRetryCount++ {
			// clear machine instructions and VR allocations from previous attempt
			for _, block := range funcCFG.Blocks {
				block.MachineInstructions = make([]cfg.MachineInstruction, 0)
			}

			vrAlloc := cfg.NewVirtualRegisterAllocator()
			result.VrAllocators[funcCFG.FunctionName] = vrAlloc
			selector := cfg.NewInstructionSelectorZ80(vrAlloc)

			// Run instruction selection on the CFG (modifies CFG in-place, adds MachineInstructions)
			err := cfg.SelectInstructions(funcCFG, vrAlloc, selector, symbolContext)
			if err != nil {
				result.CodeGenErrors = append(result.CodeGenErrors, err)
				return result, fmt.Errorf("instruction selection failed: %w", err)
			}

			instructions := funcCFG.GetAllInstructions()
			cfg.MarkUnusedVirtualRegisters(vrAlloc.GetAll(), instructions)

			// ==========================================================================
			// Stage 6: Liveness Analysis
			// ==========================================================================
			if opts.Verbose {
				fmt.Println("==> Stage 6: Liveness Analysis")
			}

			liveness := cfg.ComputeLiveness(funcCFG)
			result.LivenessInfo[funcCFG.FunctionName] = liveness

			if opts.Verbose {
				fmt.Printf("  Computed liveness for function '%s' (%d)\n", funcCFG.FunctionName, allocationRetryCount)
			}

			// ==========================================================================
			// Stage 7: Interference Graph Construction
			// ==========================================================================
			if opts.Verbose {
				fmt.Println("==> Stage 7: Interference Graph Construction")
			}

			interference := cfg.BuildInterferenceGraph(funcCFG, liveness, vrAlloc.GetAll())
			result.InterferenceInfo[funcCFG.FunctionName] = interference

			if opts.Verbose {
				nodes := interference.GetNodes()
				edgeCount := 0
				for _, node := range nodes {
					edgeCount += interference.GetDegree(node)
				}
				edgeCount /= 2 // Each edge counted twice
				fmt.Printf("  Built interference graph for function '%s' with %d nodes, %d edges (%d)\n",
					funcCFG.FunctionName, len(nodes), edgeCount, allocationRetryCount)
			}

			// ==========================================================================
			// Stage 8: Register Allocation
			// ==========================================================================
			if opts.Verbose {
				fmt.Println("==> Stage 8: Register Allocation")
			}

			// Create register allocator with target registers
			allocator := cfg.NewRegisterAllocator(selector.GetTargetRegisters())

			//interference := result.InterferenceInfo[funcCFG.FunctionName]

			// Run register allocation (assigns PhysicalReg to each VirtualRegister)
			// Parent-child VR allocations are kept in sync automatically during allocation
			allocationSucceeded, spilled := allocator.Allocate(funcCFG, interference, vrAlloc.GetAll())

			if spilled != "" {
				if _, ok := symbolContext[spilled]; ok {
					return result, fmt.Errorf("    Spilled variable '%s' was not honored by the Instruction Selection. Aborting.\n", spilled)
				}

				symbolContext[spilled] = cfg.StackLocation

				if opts.Verbose {
					fmt.Printf("  Spilled variable '%s' to stack for function '%s' (%d)\n", spilled, funcCFG.FunctionName, allocationRetryCount)
				}
			}

			// If there are unallocated VRs, run second pass to resolve them
			if !allocationSucceeded && spilled == "" {
				var err error
				allocationSucceeded, err = allocator.ResolveUnallocated(funcCFG, interference, selector)
				if err != nil {
					result.CodeGenErrors = append(result.CodeGenErrors, err)
					return result, fmt.Errorf("failed to resolve unallocated VRs for %s: %w", funcCFG.FunctionName, err)
				}
			}

			if allocationSucceeded {
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
					fmt.Printf("  Allocated %d registers, spilled %d for function '%s' (%d)\n", allocated, spilled, funcCFG.FunctionName, allocationRetryCount)
				}
			}
		}

		if !allocationSucceeded {
			result.CodeGenErrors = append(result.CodeGenErrors, fmt.Errorf("register allocation failed for function '%s' after %d attempts", funcCFG.FunctionName, allocationRetryCount))
			return result, fmt.Errorf("register allocation failed for function '%s' after %d attempts", funcCFG.FunctionName, allocationRetryCount)
		}
	}

	// ==========================================================================
	// Stage 9: Code Generation (emit final instructions)
	// ==========================================================================

	// ==========================================================================
	// Pipeline Complete
	// ==========================================================================
	result.Success = true
	return result, nil
}
