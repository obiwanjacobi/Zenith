# Zenith Compiler Architecture & Implementation Plan

**Last Updated:** January 29, 2026

**Purpose:** Comprehensive architecture documentation for code generation decisions

---

## Table of Contents

1. [System Architecture Overview](#system-architecture-overview)
2. [ZIR - Zenith Intermediate Representation](#zir---zenith-intermediate-representation)
3. [Control Flow Graph (CFG)](#control-flow-graph-cfg)
4. [Virtual Registers](#virtual-registers)
5. [Instruction Selection](#instruction-selection)
6. [Register Allocation](#register-allocation)
7. [Calling Conventions](#calling-conventions)
8. [Z80 Target Implementation](#z80-target-implementation)
9. [Current Issues & Decisions](#current-issues--decisions)

---

## System Architecture Overview

### Compilation Pipeline

```

Source Code
    ↓
[Lexer] → Token Stream
    ↓
[Parser] → AST (Abstract Syntax Tree)
    ↓
[Semantic Analyzer] → ZIR (Zenith IR) + Symbol Tables
    ↓
[CFG Builder] → Control Flow Graphs (one per function)
    ↓
[Instruction Selection] → Virtual Register Machine Code
    ↓
[Liveness Analysis] → Variable lifetime information
    ↓
[Interference Graph] → Register conflict information
    ↓
[Register Allocation] → Physical register assignments
    ↓
[Code Emission] → Target assembly/machine code

```

**Key Principle:** Each stage transforms a representation while preserving semantics. Later stages cannot create new control flow or change program logic - they only refine the representation.

### Component Separation

#### Frontend (AST → ZIR)

- **Purpose:** Language semantics, type checking, name resolution

- **Output:** Fully-typed IR with symbol tables

- **Location:** `compiler/parser/`, `compiler/zir/semantic_analyzer.go`

#### Backend (ZIR → Machine Code)

- **Purpose:** Target-specific code generation, optimization

- **Output:** Assembly or machine code

- **Location:** `compiler/cfg/`, `compiler/emit/`

**Critical Boundary:** ZIR is target-independent. All target-specific decisions happen in CFG/instruction selection.

---

## ZIR - Zenith Intermediate Representation

### Purpose

- **Target-independent** intermediate form

- Represents program semantics after type checking and name resolution

- Similar to LLVM IR but simpler and tree-based

### Node Hierarchy

**IRNode** - Base interface for all IR nodes, links back to AST

**Three main categories:**

- **IRDeclaration** - Top-level declarations (functions, variables, types)
- **IRStatement** - Executable statements (assignments, control flow, blocks)
- **IRExpression** - Value-producing expressions (operations, constants, calls)

All expressions have resolved types - no type inference needed during code generation.

### Key Interfaces

#### IRExpression

**Location:** `compiler/zir/ir_nodes.go`

Every expression in ZIR has a `Type() Type` method that returns its resolved type. This is critical for instruction selection - you never need to infer types, just call `expr.Type()`.

**Design Principle:** Type checking is complete before code generation begins. Expression types are immutable.

#### Type System

#### Type System

**Location:** `compiler/zir/types.go`

**Type interface** provides:

- `Size() int` - Returns size in bytes
- `String() string` - Human-readable representation

**Concrete type implementations:**

- **PrimitiveType** - u8, u16, i8, i16, d8, d16, bool
- **FunctionType** - Function signatures
- **StructType** - Struct definitions
- **ArrayType** - Array types

**When to use:**

- Call `expr.Type().Size()` to determine register size (8-bit or 16-bit)
- Use Type for register allocation size decisions
- **Never** inspect AST node types - always work with the Type interface

### Symbol Tables

**Location:** `compiler/zir/symbols.go`

**Symbol** represents a named entity (variable, function, parameter, etc.) with:

- Name
- Type
- Kind (Variable, Function, Parameter, etc.)
- Usage tracking information

**Purpose:** Maps names to semantic information. Used by:

- `InstructionSelector.SelectLoadVariable(symbol)` - needs type/size

- Register allocator - needs usage patterns

- Calling convention - needs parameter info

---

## Control Flow Graph (CFG)

### Purpose

- Represents control flow explicitly (jumps, branches, merge points)

- Created from tree-based ZIR statements

- Enables analysis (liveness, dataflow) and optimization

### Structure

**Location:** `compiler/cfg/cfg.go`

#### BasicBlock

Represents a sequence of instructions with single entry and single exit.

**Key properties:**

- Contains only straight-line IR statements (no branches mid-block)
- Has unique ID and optional label (Entry, Exit, IfThen, etc.)
- Connected to other blocks via Successor and Predecessor edges
- Instructions are `[]zir.IRStatement`

**Critical Invariant:** Control flow between blocks is explicit through edges, not through instructions within blocks.

#### CFG (Control Flow Graph)

Represents the control flow structure of a function.

**Contains:**

- Entry block (function start, no predecessors)
- Exit block (all returns converge here, no successors)
- All blocks in the function
- Function name for qualified variable names

**Properties:**

- Exactly one Entry, exactly one Exit
- All blocks reachable from Entry
- Entry has no predecessors, Exit has no successors

### BlockLabel Types

**Location:** `compiler/cfg/cfg.go`

Labels identify block roles in control flow: Entry, Exit, IfThen, IfElse, IfMerge, ForCond, ForBody, ForInc, ForExit, etc.

**When to use:** Labels are for debugging and visualization. Don't use them for code generation logic - work with BasicBlock references directly.

### CFG Builder

**Location:** `compiler/cfg/cfg.go`

Transforms ZIR statements into CFG structure by creating basic blocks and connecting them with edges.

**Usage:** `builder := NewCFGBuilder(); cfg := builder.BuildCFG(functionDecl)`

**Design Principle:** CFG builder is responsible for ALL control flow creation. Instruction selection NEVER creates new basic blocks or modifies control flow.

---

## Virtual Registers

### Purpose

- Abstract away physical register constraints during instruction selection

- Allow unlimited "registers" - makes instruction selection simple

- Register allocator later maps VRs → physical registers

### VirtualRegister Structure

**Location:** `compiler/cfg/instruction_selector.go`

Represents an abstract register before physical allocation.

**Key properties:**

- Unique ID for tracking
- Size in bits (8, 16, 32, 64) - determines compatible physical registers
- Optional constraints: AllowedSet (specific registers) or RequiredClass
- PhysicalReg field populated after register allocation
- Can be backed by stack location (HasStackHome, StackOffset)

### Allocation Methods

**Location:** `VirtualRegisterAllocator` in `compiler/cfg/instruction_selector.go`

#### Unconstrained

`vr := vrAlloc.Allocate(8)` - Allocate any 8-bit register

**When to use:** Most temporary values, expression results

#### Constrained by Register Set

`vrA := vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)`

Allocate specific register(s).

**When to use:** Target requires specific register (e.g., Z80 ADD requires A register)

#### Constrained by Class

`vrIndex := vrAlloc.AllocateConstrained(16, nil, RegisterClassIndex)`

Allocate any register of a certain class.

**When to use:** Need register of certain type but not specific one (e.g., any index register)

#### Stack-Backed

`vrParam := vrAlloc.AllocateWithStackHome("param1", 8, stackOffset)`

Allocate with permanent stack location.

**When to use:** Function parameters and locals that live on stack. Register allocator can spill back to this location without allocating new stack space.

### Design Principles

1. **One VR per value** - Each expression result gets a fresh VR
2. **No reuse during selection** - Don't try to optimize register usage; allocator does that
3. **Constraints express requirements** - If instruction needs specific register, constrain it
4. **Size matters** - 8-bit VR can only map to 8-bit physical register (or low byte of 16-bit pair)

---

## Instruction Selection

### Purpose

- Transform ZIR expressions/statements into target-specific machine instructions

- Work with Virtual Registers (not physical registers)

- Generate straight-line instruction sequences

### InstructionSelector Interface

Large interface (~40 methods) covering:

- **Arithmetic:** `SelectAdd`, `SelectSubtract`, `SelectMultiply`, `SelectDivide`, `SelectNegate`

- **Bitwise:** `SelectBitwiseAnd/Or/Xor/Not`, `SelectShiftLeft/Right`

- **Logical:** `SelectLogicalAnd/Or/Not` (boolean operations)

- **Comparison:** `SelectEqual`, `SelectNotEqual`, `SelectLessThan`, etc.

- **Memory:** `SelectLoad`, `SelectStore`, `SelectLoadConstant`

- **Variables:** `SelectLoadVariable`, `SelectStoreVariable`

- **Control:** `SelectBranch`, `SelectJump`, `SelectCall`, `SelectReturn`

- **Utility:** `SelectMove`, `EmitInstruction`

**Every method returns** `(*VirtualRegister, error)` or `error`

### InstructionSelectionContext

**Location:** `compiler/cfg/instruction_selection.go`

Maintains state during instruction selection for one compilation unit.

**Key responsibilities:**

- Tracks VirtualRegister allocator
- Maps symbols to their VirtualRegisters (variables → VRs)
- Caches expression results (for CSE, avoiding recomputation)
- Maintains reference to current function and CFG being processed
- Coordinates with calling convention for parameter/return handling

**Key mappings:**

- `symbolToVReg`: Variables → their VirtualRegisters

- `exprToVReg`: Expressions → cached results (for CSE, avoiding recomputation)

### Selection Process

**High-level process:**

1. Build CFG for each function
2. Allocate VRs for parameters (per calling convention: registers or stack)
3. Map parameters to their VRs in context
4. For each basic block, select instructions for each statement
5. Expression selection recursively allocates VRs and emits instructions

**Parameter handling:**

- Calling convention determines register vs stack placement
- Stack parameters get `AllocateWithStackHome()` (can spill to same location)
- Register parameters get `AllocateConstrained()` (must be in specific register)

### Design Principles

#### 1. Expression-Level Only

**Selector methods generate straight-line instruction sequences** - no control flow.

❌ **NEVER DO THIS:**

```go
func (z *Z80) SelectAdd(left, right *VR, size int) (*VR, error) {
    if size > 8 {
        // Jump to helper...  ← WRONG! No jumps in selector
    }
}

```

✅ **DO THIS INSTEAD:**

Call runtime helper for operations beyond target capabilities: `__add32`, `__mul16`, etc.

#### 2. Virtual Registers, Not Physical

**Work with VRs. Express constraints, don't manually allocate.**

❌ **WRONG:** Manually assigning physical registers

✅ **CORRECT:** Use `AllocateConstrained()` with register class/allowed set, let allocator handle assignment

#### 3. No Control Flow References

**BasicBlocks exist, but don't create new ones. Reference existing blocks only.**

✅ **SelectBranch usage:** Accept BasicBlock parameters that CFG builder already created

❌ **Creating blocks in selector:** Never instantiate new BasicBlocks in selector methods

#### 4. Runtime Helpers for Complex Operations

**Operations needing loops/complex control flow → runtime library.**

Examples:

- Multiplication/division (no native instructions on Z80)
- Variable shifts (shift by non-constant amount)
- Logical operations with short-circuit evaluation
- Flag-to-boolean conversion

Use `NewZ80Call()` to invoke runtime functions like `__mul8`, `__shl16_var`, etc.

#### 5. Check Before Using

**Never call non-existent functions. Verify constructors exist.**

✅ **Correct approach:**

1. Check what constructors exist: `NewZ80Instruction`, `NewZ80InstructionImm8`, etc.
2. Check what opcodes are defined: `Z80_LD_R_R`, `Z80_ADD_A_R`, etc.
3. Only use what exists, or add missing pieces to plan

---

## Register Allocation

### Purpose

- Maps VirtualRegisters → Physical Registers

- Handles register pressure (more VRs than physical registers)

- Respects constraints from instruction selection

### Physical Register Model

```go
type Register struct {
    Name        string
    Size        int  // 8 or 16 bits
    Class       RegisterClass
    Composition []*Register  // For pairs (HL = H + L)
    RegisterId  int         // Encoding ID
}

```

#### RegisterClass

```go
const (
    RegisterClassGeneral       // General purpose (B, C, D, E on Z80)
    RegisterClassAccumulator   // Special arithmetic reg (A on Z80)
    RegisterClassIndex         // Address/pointer regs (HL, IX, IY)
    RegisterClassStackPointer  // SP
    RegisterClassFlags         // Flag register
)

```

### Allocation Algorithm

1. **Liveness Analysis** - Determine where each variable is live
2. **Interference Graph** - Build graph of conflicting variables
3. **Graph Coloring** - Assign registers (colors) to non-interfering variables
4. **Spilling** - If not enough registers, spill some to stack

### Precoloring

Function parameters can be pre-assigned to specific registers per calling convention.

**Location:** [instruction_selector.go](src/compiler/cfg/instruction_selector.go)

**Method:** `AllocateWithPrecoloring(graph, symbolInfo, precolored map)` accepts precolored parameters

**When to use:** Parameters passed in specific registers per calling convention.

### Capabilities

**Location:** [register_capabilities.go](src/compiler/cfg/register_capabilities.go)

**Purpose:** Target-specific heuristics for register selection.

**Interface:** Provides `ScoreRegisterForUsage()` and `IsRegisterPair()` methods

Z80 implementation prefers:

- A register for arithmetic
- HL for pointers
- BC/DE for counters

---

## Calling Conventions

### Purpose

- Defines how functions communicate (parameters, return values)
- Where arguments go (registers vs stack)
- Who saves what registers

### CallingConvention Interface

**Location:** [calling_convention.go](src/compiler/cfg/calling_convention.go)

**Key responsibilities:**

- Map parameter index/size → register or stack location
- Return appropriate register for return values by size
- Define caller-saved vs callee-saved register sets
- Specify stack alignment and growth direction

### Usage in Instruction Selection

During function prologue generation:

- Query parameter locations for each parameter (register or stack)
- Allocate VRs constrained to specified registers or with stack homes
- Query return value register for function return type
- Use caller/callee-saved sets to generate save/restore sequences

**Design Principle:** Calling convention is pluggable. Different conventions for different platforms/ABIs.

---

## Z80 Target Implementation

### Current State - What Exists

#### Instruction Constructors

**Location:** [instruction_selector_z80.go](src/compiler/cfg/instruction_selector_z80.go) and related files

**Available constructors:**

- `NewZ80Instruction(opcode, result, operand)` - Basic 2-operand instructions
- `NewZ80InstructionImm8(opcode, result, imm8)` - With immediate byte
- `NewZ80InstructionImm16(opcode, result, imm16)` - With immediate word
- `NewZ80Branch(opcode, condition, trueBlock, falseBlock)` - Conditional branch
- `NewZ80Jump(opcode, target *BasicBlock)` - Unconditional jump
- `NewZ80Call(functionName string)` - Function call
- `NewZ80Return()` - Return

**Important:** These are the ONLY constructors. Don't invent new ones.

#### Z80MachineInstruction

**Location:** [instruction_selector_z80.go](src/compiler/cfg/instruction_selector_z80.go)

**Purpose:** Target-specific representation of Z80 instructions

**Key properties:** Opcode, result/operand VRs, immediates (8/16-bit), branch targets, function name, optional comment

#### Opcode Categories

- **Load/Store:** LD_R_R, LD_R_N, LD_R_HL, LD_HL_R, LD_RR_NN, etc.

- **8-bit Arithmetic:** ADD_A_R, SUB_R, ADC_A_R, SBC_A_R, INC_R, DEC_R

- **16-bit Arithmetic:** ADD_HL_RR, ADC_HL_RR, SBC_HL_RR, INC_RR, DEC_RR

- **Bitwise:** AND_R, OR_R, XOR_R, CPL (complement A)

- **Shifts (CB prefix):** SLA_R, SRL_R, SRA_R, RLC_R, RRC_R, RL_R, RR_R

- **Compare:** CP_R, CP_N, CP_HL

- **Control Flow:** JP_NN, JP_CC_NN, JR_E, JR_CC_E, CALL_NN, RET, RET_CC

- **Stack:** PUSH_RR, POP_RR

- **Special:** NOP, HALT, DI, EI

#### Z80 Registers

```go
// 8-bit
RegA, RegB, RegC, RegD, RegE, RegH, RegL

// 16-bit pairs
RegBC, RegDE, RegHL, RegSP, RegIX, RegIY
```

**Available Z80 registers:**

A, B, C, D, E, H, L (8-bit) | BC, DE, HL, SP, IX, IY (16-bit) | F (flags)

### Implementation Status

#### ✅ Implemented & Working

- **Arithmetic (8/16-bit):** Add, Subtract via native instructions; Multiply, Divide via runtime helpers
- **Bitwise (8-bit):** AND, OR, XOR, NOT via native instructions
- **Shifts:** Via runtime helpers (`__shl8`, `__shr8`, etc.)
- **Logical:** Via runtime helpers (`__logical_and`, `__logical_or`, `__logical_not`)
- **Comparisons:** Via runtime helpers (`__cmp_lt8`, `__cmp_eq8`, etc.)
- **Memory:** Load/Store via HL pointer
- **Load Constant:** Immediate values
- **Control Flow:** Branch, Jump work with BasicBlocks
- **Function Calls:** Parameter setup + CALL instruction
- **Return:** Return value setup + RET

#### ❌ Not Implemented / Broken

- **Variable Load/Store:** Uses non-existent `NewZ80InstructionOffset` and IX+d opcodes
- **16-bit Bitwise:** AND, OR, XOR only implemented for 8-bit
- **16-bit Memory:** Load/Store only supports 8-bit values

---

## Current Issues & Decisions

### 🔴 BLOCKED: Variable Load/Store

**Problem:** `SelectLoadVariable` and `SelectStoreVariable` reference infrastructure that doesn't exist (IX indexed addressing with displacement).

#### Option A: Add IX Indexed Addressing (Proper Z80 Support)

**Pros:** Correct Z80 instruction set, efficient, matches hardware

**Cons:** Need DD-prefix opcodes, new constructor, extended MachineInstruction

**Requirements:** Define opcodes with DD prefix, add `NewZ80InstructionOffset()` constructor, update code emission

#### Option B: Use HL Indirection (Simpler, Less Efficient)

**Pros:** Uses existing infrastructure, no new opcodes, works immediately

**Cons:** Less efficient (multiple instructions), more register pressure

**Approach:** Calculate address into HL, then use `LD R, (HL)` / `LD (HL), R`

#### Option C: Runtime Helpers (Defer to Library)

**Pros:** Consistent with multiply/divide approach

**Cons:** Function call overhead for every variable access, not practical

#### Option D: Not Implemented (Placeholder)

**Pros:** Honest about current state, defer until architecture settled

**Cons:** Can't compile real programs yet

**Approach:** Return error until design finalized

### Decision Required

**Question for user:** Which approach for variable load/store?

**Recommendation:** Option A (proper IX support) for correctness, or Option D (defer) until calling convention and stack frame layout finalized.

### Other Known Issues

1. **Runtime helpers undefined:** All `__*` functions need assembly implementations
2. **Short-circuit evaluation:** Logical AND/OR can't short-circuit at expression level
3. **Flag-to-boolean:** Comparisons returning 0/1 require control flow or special sequences
4. **Stack frame convention:** How are locals laid out? IX or SP relative?

---

## Design Principles Summary

### When Generating Code

#### ✅ DO

- Read existing implementations first

- Verify functions/types exist before using

- Use VirtualRegisters, express constraints

- Reference existing BasicBlocks only

- Follow established patterns

- Check this plan document for decisions

#### ❌ DON'T

- Invent functions that don't exist

- Create new BasicBlocks in selector

- Use string labels for jumps

- Manually assign PhysicalReg

- Implement loops/branches in expression selection

- Assume something exists - verify first

### Code Review Checklist

Before implementing:

1. [ ] Read similar existing code
2. [ ] List all functions/types I'll use
3. [ ] Verify each exists (grep/read)
4. [ ] Check plan document for decisions
5. [ ] Update plan with new decisions

---

**Next Action:** Resolve variable load/store decision, then continue implementation.

#### What Exists ✅

**Instruction Constructors:**

- `NewZ80Instruction(opcode, result, operand)` - Basic 2-operand instruction

- `NewZ80InstructionImm8(opcode, result, imm8)` - With 8-bit immediate

- `NewZ80InstructionImm16(opcode, result, imm16)` - With 16-bit immediate

- `NewZ80Call(functionName)` - Function call
- `NewZ80Jump(opcode, target *BasicBlock)` - Unconditional jump to block
- `NewZ80Branch(opcode, condition, trueBlock, falseBlock)` - Conditional branch
- `NewZ80Return()` - Return instruction

**Available opcodes:** Standard load/store, 8/16-bit arithmetic, bitwise, shifts/rotates (CB prefix), comparisons, control flow

**What's Missing:**

- IX/IY indexed addressing opcodes (DD/FD prefix)
- `NewZ80InstructionOffset()` constructor for indexed addressing
- Many 16-bit bitwise operations (AND, OR, XOR for 16-bit)

### Implementation Summary

#### ✅ Completed Selector Functions

**Arithmetic:** Add/Subtract (native for 8/16-bit), Multiply/Divide (runtime helpers `__mul*`, `__div*`)

**Bitwise:** AND/OR/XOR/NOT (8-bit native instructions)

**Shifts:** Left/Right via runtime helpers (`__shl*`, `__shr*`)

**Logical:** AND/OR/NOT via runtime helpers (short-circuit evaluation needs CFG support)

**Comparisons:** Equal/NotEqual/LessThan/LessEqual/GreaterThan/GreaterEqual via runtime helpers (`__cmp_*`)

**Memory:** Load/Store via HL pointer, LoadConstant via immediate instructions

**Control Flow:** Branch (conditional jumps between blocks), Jump (unconditional), Call (with arg setup), Return (with value handling)

#### ❌ Not Implemented

**Variable Load/Store:** Uses non-existent IX indexed addressing (see Current Issues section)

**16-bit Bitwise:** AND/OR/XOR only implemented for 8-bit

**16-bit Memory:** Load/Store only supports 8-bit values

### Known Issues & TODOs

1. **Short-circuit evaluation** - Logical AND/OR can't short-circuit at expression level, needs CFG support
2. **Flag to boolean conversion** - Comparisons need control flow to convert flags to 0/1 values, or special instruction sequences
3. **Stack frame convention** - How are variables laid out? What register points to frame? (IX? SP?)

#### Future Enhancements

- Optimize constant shifts (inline SLA/SRL instructions instead of loops)

- Optimize comparisons against zero (just test flags from previous operation)

- Add peephole optimization for redundant loads

- Support for IX/IY index registers properly

### Design Principles Established

#### Expression-Level Operations

- Selector functions generate **straight-line code** only

- No branches/jumps/labels within expression selection

- Complex control flow delegated to CFG builder or runtime helpers

#### Control Flow via Basic Blocks

- All jumps reference `*BasicBlock`, never string labels

- No `nextLabel()` or label generation in selector

- Branch targets determined by CFG structure

#### Virtual Registers

- All operations work with VirtualRegisters

- Physical register constraints expressed via `AllocateConstrained()`

- Register allocator handles mapping VR → physical register

#### Runtime Helpers for Complex Operations

- Multiply, divide, shifts, logical ops, comparisons use `__*` helpers

- Helpers implement operations requiring loops or complex sequences

- Keeps instruction selector simple and focused

### Testing Status

- ✅ All tests passing as of last run

- ⚠️  No specific tests for variable load/store (broken functions not called yet)

- ⚠️  No tests for 16-bit bitwise operations (not implemented)

---

**Next Steps:** Resolve variable load/store strategy decision, then proceed with implementation.
