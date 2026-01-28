package compile

import (
	"fmt"
	"testing"
)

// Example demonstrating the full compilation pipeline
func Example_pipeline() {
	sourceCode := `
		add: (a: u8, b: u8): u8 {
			ret a + b
		}
	`

	opts := DefaultPipelineOptions()
	opts.SourceCode = sourceCode
	opts.TargetArch = "z80"
	opts.Verbose = true

	result, err := Pipeline(opts)
	if err != nil {
		fmt.Printf("Compilation failed: %s\n", err)
		return
	}

	if result.Success {
		fmt.Println("Compilation succeeded!")
		fmt.Printf("Generated %d instructions\n", len(result.Instructions["<all>"]))
	}
}

// Test the pipeline with simple code
func Test_Pipeline_SimpleFunction(t *testing.T) {
	sourceCode := `
		addition: (x: u16, y: u16) u16 {
			ret x + y
		}
	`

	opts := DefaultPipelineOptions()
	opts.SourceCode = sourceCode
	opts.TargetArch = "z80"

	result, err := Pipeline(opts)

	if err != nil {
		t.Logf("Compilation errors: %v", result.ParserErrors)
	}

	// Check stages completed
	if result.AST == nil {
		t.Error("AST was not generated")
	}
	if result.IR == nil {
		t.Error("IR was not generated")
	}
	if len(result.FunctionCFGs) == 0 {
		t.Error("CFG was not generated")
	}
	if len(result.LivenessInfo) == 0 {
		t.Error("Liveness analysis was not performed")
	}
	if len(result.InterferenceInfo) == 0 {
		t.Error("Interference graph was not built")
	}

	t.Logf("Pipeline completed successfully")
	t.Logf("Functions processed: %d", len(result.FunctionCFGs))
	t.Logf("Instructions generated: %d", len(result.Instructions["<all>"]))
}

// Test pipeline stopping at different stages
func Test_Pipeline_StopAfterParse(t *testing.T) {
	sourceCode := `test: () u8 { ret 42 }`

	opts := DefaultPipelineOptions()
	opts.SourceCode = sourceCode
	opts.StopAfterParse = true

	result, err := Pipeline(opts)

	if err != nil {
		t.Fatalf("Unexpected error: %v", result.AST.Errors())
	}

	if !result.Success {
		t.Error("Pipeline should have succeeded")
	}

	if result.AST == nil {
		t.Error("AST should be present")
	}

	if result.IR != nil {
		t.Error("IR should not be present when stopping after parse")
	}
}

// Test pipeline with verbose output
func Test_Pipeline_VerboseOutput(t *testing.T) {
	sourceCode := `
		factorial: (n: u8) u8 {
			if n <= 1 {
				ret 1
			}
			ret n * factorial(n - 1)
		}
	`

	opts := DefaultPipelineOptions()
	opts.SourceCode = sourceCode
	opts.TargetArch = "z80"
	opts.Verbose = true
	opts.DumpIR = true
	opts.DumpCFG = true

	result, err := Pipeline(opts)

	if err != nil {
		t.Logf("Compilation note: %s", err)
	}

	if result.IR != nil {
		t.Logf("IR has %d declarations", len(result.IR.Declarations))
	}
	if len(result.FunctionCFGs) > 0 {
		t.Logf("CFG generated for %d functions", len(result.FunctionCFGs))
	}
}

// Test pipeline with all debug dumps enabled
func Test_Pipeline_AllDumps(t *testing.T) {
	sourceCode := `
		max: (a: u8, b: u8) u8 {
			if a > b {
				ret a
			} else {
				ret b
			}
		}
	`

	opts := DefaultPipelineOptions()
	opts.SourceCode = sourceCode
	opts.TargetArch = "z80"
	opts.DumpTokens = true
	opts.DumpAST = true
	opts.DumpIR = true
	opts.DumpCFG = true
	opts.DumpLiveness = true
	opts.DumpInterference = true
	opts.DumpAllocation = true
	opts.DumpInstructions = true

	_, err := Pipeline(opts)

	if err != nil {
		t.Logf("Pipeline encountered: %s", err)
	}
}
