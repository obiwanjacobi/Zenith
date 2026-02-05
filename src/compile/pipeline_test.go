package compile

import (
	"fmt"
	"testing"
	"zenith/compiler/cfg"
)

func RunPipeline(t *testing.T, source string) *CompilationResult {
	opts := DefaultPipelineOptions()
	opts.SourceCode = source
	opts.TargetArch = "z80"
	//opts.Verbose = true

	result, err := Pipeline(opts)

	if err != nil {
		t.Logf("Compilation failed: %s", err)
	}
	for _, perr := range result.ParserErrors {
		fmt.Printf("  ParseErr: %s\n", perr.Error())
	}
	for _, serr := range result.SemanticErrors {
		fmt.Printf("  SemErr: %s\n", serr.Error())
	}

	for fnName, funcCFG := range result.FunctionCFGs {
		cfg.DumpCFG(fnName, funcCFG, cfg.DumpInstructions)
	}

	if result.VRAllocator != nil {
		cfg.DumpAllocation(result.VRAllocator)
	}

	return result
}

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
	if result.SemCU == nil {
		t.Error("Semantic compilation unit was not generated")
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

	if result.SemCU != nil {
		t.Error("Semantic compilation unit should not be present when stopping after parse")
	}
}

// Test pipeline with verbose output
func Test_Pipeline_Factorial(t *testing.T) {
	sourceCode := `
		factorial: (n: u8) u8 {
			if n <= 1 {
				ret 1
			}
			ret n * factorial(n - 1)
		}
	`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_Max(t *testing.T) {
	sourceCode := `
		max: (a: u8, b: u8) u8 {
			if a > b {
				ret a
			} else {
				ret b
			}
		}
	`

	RunPipeline(t, sourceCode)
}

func Test_Pipeline_ArrMax(t *testing.T) {
	sourceCode := `
		max: (arr: u8[]) u8 {
			if arr[0] > arr[1] {
				ret arr[0]
			} else {
				ret arr[1]
			}
		}
	`

	RunPipeline(t, sourceCode)
}
