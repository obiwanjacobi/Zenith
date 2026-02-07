# Zenith Compiler - AI Coding Agent Instructions

## Project Overview
Zenith is a compiler for a custom programming language targeting the Z80 CPU. It's written in Go and implements a multi-stage compilation pipeline from source code to Z80 machine code.

## Architecture: Multi-Stage Compilation Pipeline

The compilation follows a strict single-pass flow (see [compile/pipeline.go](../src/compile/pipeline.go)):

```
Source → [Lexer] → Tokens → [Parser] → AST → [Semantic Analyzer] → ZIR/SemNodes
    → [CFG Builder] → Control Flow Graphs → [Instruction Selection] → Virtual Register Machine Code
    → [Liveness Analysis] → [Interference Graph] → [Register Allocation] → Physical Registers
```

**Critical Boundary**: `compiler/zsm` (Zenith Semantic Model) produces target-independent IR. All Z80-specific decisions happen in `compiler/cfg`.

## Key Components

### 1. Frontend (`compiler/lexer`, `compiler/parser`, `compiler/zsm`)
- **Lexer**: Token-based, streams via channels. See [tokens.go](../src/compiler/lexer/tokens.go) for TokenId enum.
- **Parser**: Recursive descent, produces AST as `ParserNode` interface hierarchy. Grammar documented in [parser/grammar.md](../src/compiler/parser/grammar.md).
- **Semantic Analyzer**: Two-pass analysis in [sem_analyzer.go](../src/compiler/zsm/sem_analyzer.go):
  - Pass 1: Register all declarations (enables forward references)
  - Pass 2: Full type checking and build `Sem*` nodes (e.g., `SemFunctionDecl`, `SemBinaryOp`)

**IR Naming**: All semantic IR nodes start with `Sem*` prefix (e.g., `SemExpression`, `SemStatement`). These inherit from `SemNode` interface.

### 2. Backend (`compiler/cfg`)
- **CFG**: Control flow graphs built from semantic IR. Each function gets one CFG with `BasicBlock` nodes.
- **Virtual Registers**: Pre-allocation representation. See [virtual_register.go](../src/compiler/cfg/virtual_register.go). Types: `CandidateRegister`, `ImmediateValue`, `StackLocation`, `AllocatedRegister`.
- **Instruction Selection**: Z80-specific in [instruction_selector_z80.go](../src/compiler/cfg/instruction_selector_z80.go). Maps semantic operations to Z80 opcodes (`Z80_ADD_A_N`, `Z80_LD_R_R`, etc).
- **Register Allocation**: Graph coloring algorithm in [register_allocation.go](../src/compiler/cfg/register_allocation.go).
- **Calling Convention**: Z80 ABI defined in [calling_convention_z80.go](../src/compiler/cfg/calling_convention_z80.go).

### 3. Pipeline Orchestration (`compile/pipeline.go`)
- Entry point: `Pipeline(opts *PipelineOptions) (*CompilationResult, error)`
- `CompilationResult` accumulates all IR stages (Tokens, AST, SemCU, FunctionCFGs, etc)
- Pipeline stages can be stopped early via `StopAfter*` flags for debugging

## Critical Conventions

### Type System
- Primitive types: `u8`, `u16`, `i8`, `i16`, `d8` (BCD), `d16`, `bool`
- Defined in [zsm/types.go](../src/compiler/zsm/types.go) with global instances (`U8Type`, `U16Type`, etc)
- All `SemExpression` nodes have `.ResolvedType()` - never nil after semantic analysis

### Naming Patterns
- **AST nodes**: `ParserNode` interface, concrete types like `VariableDeclaration`, `BinaryOp`
- **Semantic IR**: `Sem*` prefix (e.g., `SemBinaryOp`, `SemFunctionDecl`)
- **CFG instructions**: `MachineInstruction` interface, Z80 opcodes use `Z80_` prefix
- **Tests**: `Test_<Component>_<Feature>` (e.g., `Test_InstructionSelection_BinaryOp_Add`)

### Error Handling
- Lexer/Parser: Collect errors in slices, continue parsing to find more errors
- Semantic: `SemError` type with source location
- CFG/Codegen: Return errors immediately (fail-fast)

## Developer Workflows

### Building
```bash
# Run from src/ directory
go build -o ../zenith.exe main.go

# Or use build.sh for cross-platform builds (from root)
bash build.sh
```

### Testing
```bash
# Run from src/ directory
go test ./...                                    # All tests
go test ./compiler/cfg -v                        # Component tests
go test ./compile -v -run Test_Pipeline_AllDumps # Specific test with verbose output
```

**Test Structure**: Most tests use table-driven approach with `testify/assert`. Integration tests in [compile/pipeline_test.go](../src/compile/pipeline_test.go) use `RunPipeline()` helper.

### Debugging
- Enable verbose output: `opts.Verbose = true` in PipelineOptions
- CFG dump functions: `cfg.DumpCFG()`, `cfg.DumpAllocation()`, `cfg.DumpInstructions`
- Stop pipeline early: Use `StopAfterParse`, `StopAfterSemantic`, etc.

## Code Patterns to Follow

### Adding a New Instruction Type
1. Define opcode in [instructions_z80.go](../src/compiler/cfg/instructions_z80.go) with `Z80_` prefix
2. Add descriptor in [instruction_descriptor_z80.go](../src/compiler/cfg/instruction_descriptor_z80.go) (operand types, clobbers)
3. Implement selector method in [instruction_selector_z80.go](../src/compiler/cfg/instruction_selector_z80.go)
4. Add test in [instruction_selection_test.go](../src/compiler/cfg/instruction_selection_test.go)

### Working with Virtual Registers
```go
// Allocate new VR with register constraint
vrA := z.vrAlloc.Allocate(Z80RegA) // Must use register A

// Allocate any 8-bit register
vrResult := z.vrAlloc.Allocate8()

// Set immediate value
vrImm := z.vrAlloc.AllocateImmediate(42, 8)
```

### Semantic Analysis Pattern
When adding language features:
1. Add AST node in [parser_nodes.go](../src/compiler/parser/parser_nodes.go)
2. Add parser rule in [parser_rules.go](../src/compiler/parser/parser_rules.go)
3. Add `Sem*` node in [sem_nodes.go](../src/compiler/zsm/sem_nodes.go)
4. Implement analysis in [sem_analyzer.go](../src/compiler/zsm/sem_analyzer.go)
5. Add CFG handling in [cfg.go](../src/compiler/cfg/cfg.go)'s `buildStatement()`
6. Add instruction selection in Z80 selector

## Language Reference
See [docs/zentih.md](../docs/zentih.md) for Zenith language syntax and semantics.

## Architecture Documentation
- [implementation-plan.md](../docs/implementation-plan.md): Detailed design decisions for code generation
- [compiler.md](../docs/compiler.md): High-level compiler design goals
- [assembly.md](../docs/assembly.md): Inline assembly reference

## Module Structure
- **Module name**: `zenith` (defined in [go.mod](../src/go.mod))
- **Import paths**: Use `zenith/compiler/cfg`, `zenith/compiler/zsm`, etc.
- **Dependencies**: Only `github.com/stretchr/testify` for testing
