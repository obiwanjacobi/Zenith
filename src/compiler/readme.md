# Zenith Compiler

A multi-phase compiler for the Zenith programming language targeting the Z80 architecture.

## Compilation Pipeline

### 1. Lexer

**Input:** Source code text
**Output:** Stream of tokens

The lexer (tokenizer) scans the source code character by character and groups them into meaningful tokens (keywords, identifiers, operators, literals). It even preserves whitespace and comments and produces a flat sequence of tokens that the parser can consume.

**Key responsibilities:**

- Character-level scanning
- Token classification (keyword, identifier, number, string, operator)
- Position tracking for error messages
- Handling escape sequences in strings (TODO)

**Sets up:** Token stream for the parser

---

### 2. Parser

**Input:** Token stream from lexer
**Output:** Abstract Syntax Tree (AST)

The parser consumes tokens and builds a tree structure representing the syntactic structure of the program. It validates that the token sequence follows the language grammar rules (e.g., statements, expressions, declarations) but doesn't check semantic meaning yet. It provides error messages when it cannot find the tokens it expects accoording to the grammar.

**Key responsibilities:**

- Grammar validation (syntax checking)
- AST construction (representing program structure)
- Operator precedence and associativity
- Error recovery for better diagnostics (TODO)

**Uses:** Token types and values from lexer
**Sets up:** AST for semantic analysis

---

### 3. Semantic Analysis

**Input:** AST from parser
**Output:** Intermediate Representation (IR) with type information and symbol tables

Semantic analysis transforms the AST into a typed IR while validating the program's meaning. It resolves symbols (variables, functions, types), checks type compatibility, ensures variables are declared before use, and validates function calls match signatures.

**Key responsibilities:**

- Symbol table construction (tracking declarations)
- Type checking and inference
- Name resolution (linking references to declarations)
- Semantic validation (e.g., return type matches, no duplicate declarations)
- Call graph construction (function dependencies)

**Uses:** AST structure from parser
**Sets up:** Typed IR and symbol tables for code generation phases

---

### 4. Control Flow Graph (CFG)

**Input:** IR from semantic analysis
**Output:** Control flow graph with basic blocks

The CFG builder transforms the linear IR into a graph of basic blocks (sequences of instructions with single entry/exit points). Each block contains straight-line code, and edges represent possible control flow (jumps, branches, calls). This explicit graph makes data flow analysis possible.

**Key responsibilities:**

- Basic block identification (splitting on branches/labels)
- Edge construction (connecting blocks via control flow)
- Dominator analysis (which blocks always execute before others)
- Loop detection

**Uses:** IR statements and control flow from semantic analysis
**Sets up:** CFG structure for liveness and register allocation

---

### 5. Liveness Analysis

**Input:** Control Flow Graph
**Output:** Liveness information (which variables are live at each point)

Liveness analysis determines at each program point which variables contain values that will be used later. A variable is "live" if its current value might be read before being overwritten. This is computed via backward data flow analysis through the CFG.

**Key responsibilities:**

- Live-in/live-out sets for each basic block
- Backward data flow analysis (from uses to definitions)
- Iterative fixed-point computation

**Uses:** CFG structure and variable uses/definitions
**Sets up:** Liveness ranges for interference graph construction

---

### 6. Interference Graph

**Input:** Liveness information from CFG
**Output:** Interference graph (variables that need different registers)

The interference graph is constructed from liveness data. Two variables "interfere" if they are both live at the same program point, meaning they need different registers. Nodes represent variables, edges represent interference relationships. This graph becomes the input to register allocation.

**Key responsibilities:**

- Building interference edges from liveness information
- Tracking register preferences/constraints (e.g., Z80's "A" required for arithmetic)
- Handling pre-colored nodes (variables forced into specific registers)

**Uses:** Liveness ranges from liveness analysis
**Sets up:** Interference constraints for register allocator

---

### 7. Register Allocation

**Input:** Interference graph
**Output:** Register assignments for all variables

Register allocation assigns physical Z80 registers (A, B, C, D, E, H, L, or register pairs) to program variables. It's essentially graph coloring: assign registers (colors) such that no two interfering variables get the same register. If there aren't enough registers, some variables are "spilled" to memory.

**Key responsibilities:**

- Graph coloring algorithm (greedy or iterative)
- Handling Z80 register constraints (instruction requirements)
- Register coalescing (eliminating unnecessary moves)
- Spill code generation (load/store to memory when out of registers)

**Uses:** Interference graph and register constraints from instruction descriptors
**Sets up:** Register-allocated IR for instruction selection

---

### 8. Instruction Selection

**Input:** Register-allocated IR
**Output:** Sequence of Z80 instructions

Instruction selection maps high-level IR operations to actual Z80 machine instructions. It performs pattern matching on the IR, choosing the best instruction sequence for each operation while respecting Z80's constraints (e.g., ADD requires accumulator, indirect addressing uses HL).

**Key responsibilities:**

- Pattern matching (IR operations to instruction templates)
- Utilizing instruction descriptors (opcodes, operand types, constraints)
- Handling calling conventions (parameter passing, return values)
- Peephole optimization (local instruction sequences)
- Code generation using register assignments

**Uses:** Instruction descriptors (opcodes, dependencies, properties) and register allocations
**Sets up:** Final machine code for assembly emission

---

## Data Structures

### Instruction Descriptors (`cfg/instruction_descriptor_z80.go`)

Static metadata database describing every Z80 instruction:

- **Opcode** encoding (including prefixed instructions)
- **Category** (load, arithmetic, branch, etc.)
- **Dependencies** (operand types, register constraints, access patterns)
- **Flag effects** (which flags are read/written)
- **Timing** (cycle counts)
- **Properties** (immediate values, memory access, control flow)

Used primarily by instruction selection and register allocation to understand instruction capabilities and constraints.

---

## Supporting Components

### Symbol Tables

Track declarations and their scopes throughout semantic analysis.

### Type System

Defines primitive types (u8, u16, bool) and composite types (structs), used in semantic analysis and code generation.

### Calling Conventions

Define how functions pass parameters and return values (which registers/stack locations).

---
