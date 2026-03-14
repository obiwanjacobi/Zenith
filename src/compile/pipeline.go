package compile

import (
	"fmt"
	"io"
	"strings"

	"zenith/compiler"
	"zenith/compiler/cfg"
	z80 "zenith/compiler/cfg/z80"
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
	FunctionCFGs map[string]*cfg.CFG
	VRAllocators map[string]*cfg.TempVRAllocator

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
	Output  io.Writer
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
		FunctionCFGs: make(map[string]*cfg.CFG),
		VRAllocators: make(map[string]*cfg.TempVRAllocator),
		Success:      false,
	}

	// ==========================================================================
	// Stage 1: Lexical Analysis (Tokenization)
	// ==========================================================================
	if opts.Verbose {
		fmt.Fprintln(opts.Output, "==> Stage 1: Lexical Analysis")
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
		fmt.Fprintln(opts.Output, "==> Stage 2: Syntax Analysis (Parsing)")
	}

	source := &compiler.Source{Name: "pipeline_input"}
	astNode, parserErrors := parser.Parse(source, result.Tokens)
	result.AST = astNode
	result.Diagnostics = append(result.Diagnostics, parserErrors...)

	if len(parserErrors) > 0 {
		if opts.Verbose {
			fmt.Fprintf(opts.Output, "Parser found %d errors\n", len(parserErrors))
			for _, err := range parserErrors {
				fmt.Fprintf(opts.Output, "  %s\n", err.Error())
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
		fmt.Fprintln(opts.Output, "==> Stage 3: Semantic Analysis & IR Generation")
	}

	analyzer := zsm.NewSemanticAnalyzer()
	semCompilationUnit, semanticErrors := analyzer.Analyze(compilationUnit)
	result.SemCU = semCompilationUnit
	result.SemanticErrors = semanticErrors

	if len(semanticErrors) > 0 {
		if opts.Verbose {
			fmt.Fprintf(opts.Output, "Semantic analysis found %d errors\n", len(semanticErrors))
			for _, err := range semanticErrors {
				fmt.Fprintf(opts.Output, "  %s\n", err.Error())
			}
		}
		return result, fmt.Errorf("semantic analysis failed with %d errors", len(semanticErrors))
	}

	// ==========================================================================
	// Stage 4: Control Flow Graph Construction
	// ==========================================================================
	if opts.Verbose {
		fmt.Fprintln(opts.Output, "==> Stage 4: Control Flow Graph Construction")
	}

	cfgBuilder := cfg.NewCFGBuilder()
	for _, decl := range semCompilationUnit.Declarations {
		if fnDecl, ok := decl.(*zsm.SemFunctionDecl); ok {
			functionCFG := cfgBuilder.BuildCFG(fnDecl)
			result.FunctionCFGs[fnDecl.Name] = functionCFG

			if opts.Verbose {
				fmt.Fprintf(opts.Output, "  Built CFG for function '%s' with %d blocks\n", fnDecl.Name, len(functionCFG.Blocks))
			}
		}
	}

	// ==========================================================================
	// Prep: Target selection
	// ==========================================================================

	if opts.TargetArch != "z80" {
		return result, fmt.Errorf("unsupported target architecture: %s", opts.TargetArch)
	}

	regsets := cfg.RegisterSets{
		Regs8:  z80.Z80Registers8,
		Regs16: z80.Z80Registers16,
	}

	// ==========================================================================
	// Stages 5–9: Per-function code generation
	// ==========================================================================
	// All backend stages run to completion for one function before moving to
	// the next, so verbose output groups naturally by function.
	//
	//   5: TAC Lowering             (target-independent)
	//   6: Instruction Selection    (target-specific, pre-regalloc)
	//   7: Register Allocation      (target-independent linear scan)
	//   8: Peephole Optimisation    (target-specific)
	//   9: Prologue / Epilogue      (target-specific, post-regalloc)

	for fnName, funcCFG := range result.FunctionCFGs {
		// ── Stage 5: TAC Lowering ─────────────────────────────────────────────
		if opts.Verbose {
			fmt.Fprintf(opts.Output, "==> [%s] Stage 5: TAC Lowering\n", fnName)
		}

		alloc := &cfg.TempVRAllocator{}
		if err := cfg.LowerTAC(funcCFG, alloc, regsets); err != nil {
			result.CodeGenErrors = append(result.CodeGenErrors, err)
			return result, fmt.Errorf("TAC lowering failed for '%s': %w", fnName, err)
		}
		result.VRAllocators[fnName] = alloc

		if opts.Verbose {
			cfg.DumpTAC(opts.Output, fnName, funcCFG)
		}

		// ── Stage 6: Instruction Selection ───────────────────────────────────
		if opts.Verbose {
			fmt.Fprintf(opts.Output, "==> [%s] Stage 6: Instruction Selection\n", fnName)
		}

		sel := z80.NewInstructionSelectorZ80(alloc)
		if err := cfg.SelectInstructions(sel, funcCFG); err != nil {
			result.CodeGenErrors = append(result.CodeGenErrors, err)
			return result, fmt.Errorf("instruction selection failed for '%s': %w", fnName, err)
		}

		if opts.Verbose {
			for _, block := range funcCFG.Blocks {
				z80.DumpMachineInstructions(opts.Output, block)
			}
		}

		// ── Stage 7: Register Allocation ─────────────────────────────────────
		if opts.Verbose {
			fmt.Fprintf(opts.Output, "==> [%s] Stage 7: Register Allocation\n", fnName)
		}

		if err := cfg.AllocateRegisters(funcCFG); err != nil {
			result.CodeGenErrors = append(result.CodeGenErrors, err)
			return result, fmt.Errorf("register allocation failed for '%s': %w", fnName, err)
		}

		// ── Stage 8: Peephole Optimisation ───────────────────────────────────
		if opts.Verbose {
			fmt.Fprintf(opts.Output, "==> [%s] Stage 8: Peephole Optimisation\n", fnName)
		}

		z80.RunPeephole(funcCFG)

		if opts.Verbose {
			for _, block := range funcCFG.Blocks {
				z80.DumpMachineInstructions(opts.Output, block)
			}
		}

		// ── Stage 9: Prologue / Epilogue ──────────────────────────────────────
		// StackFrame.Size() is final after regalloc — spill slots have been added.
		if opts.Verbose {
			fmt.Fprintf(opts.Output, "==> [%s] Stage 9: Prologue / Epilogue\n", fnName)
		}

		sel.SelectPrologue(funcCFG.Entry, funcCFG)
		sel.SelectEpilogue(funcCFG.Exit, funcCFG)

		if opts.Verbose {
			for _, block := range funcCFG.Blocks {
				z80.DumpMachineInstructions(opts.Output, block)
			}
		}
	}

	// ==========================================================================
	// Stage 10: Code Emission         (TODO)
	// ==========================================================================

	// ==========================================================================
	// Pipeline Complete
	// ==========================================================================
	result.Success = true
	return result, nil
}
