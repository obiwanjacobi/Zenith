# Copilot Code Generation Instructions

Guidelines for generating code in the Zenith compiler project.

Rule #1: don't generate code when not asked to.
Rule #2: after forming a plan for a fix or new feature, ask the user to confirm before generating any code.
---

## Project overview

Zenith is a compiler for a custom language targeting the Z80 CPU, written in Go.
The primary optimisation goals are **code size** and **cycle efficiency** on resource-constrained 8/16-bit hardware.

---

## Repository layout

```
src/
  go.mod                          # module: zenith
  main.go
  compile/                        # compilation pipeline
  compiler/
    common.go                     # generic helpers (OfType, OfTypeInterface)
    diagnostics.go
    lexer/                        # tokeniser
    parser/                       # parser → AST
    zsm/                          # semantic analyser → semantic model
    cfg/                          # CFG builder, instruction selection, register allocation
    emit/                         # assembly emitter
docs/                             # high-level design notes
```

---

## Language facts relevant to code generation

> Full language specification: [`docs/zentih.md`](../docs/zentih.md)

The points below directly influence instruction selection, type sizing, and VR allocation. Do not duplicate information that is already in the spec; refer to it instead.

### Type widths → VR sizes

| Source type | Bytes | VR size |
|---|---|---|
| `u8` / `i8` | 1 | 8-bit |
| `u16` / `i16` | 2 | 16-bit |
| `bit` / `bit[n]` | 1 | 8-bit; boolean result tested via flags, not stored as 0/1 |
| pointer (`T*`) | 2 | 16-bit |
| array (`T[]`) | 2 | 16-bit pointer (element data lives on the stack frame) |
| struct | stack only | never in a VR directly; always accessed by address |

`u24` (3 bytes) has no direct register representation — code generation for it is not yet implemented.

### Arithmetic result widths

The result type is the widest of the two operands, **except** multiplication which always doubles: `u8 × u8 → u16`, `u16 × u16 → u32`. Handle the widened result VR accordingly.

Operator precedence (high → low): **Arithmetic → Bitwise → Comparison → Logical**. Expression trees are visited in this order; do not re-order without parenthesisation in the source.

### Compound types

- **Arrays** — fixed-length arrays are stack-allocated (`StackFrame`). The variable VR holds a 16-bit pointer to the base. Length is compile-time only (no runtime length field in the array itself, despite the conceptual model in the spec).
- **Structs** — always stack-allocated. Never passed by value; always passed and returned as a pointer. Both direct (`s.field`) and pointer (`ptr.field`) access use `.` — the selector decides which addressing mode to emit.
- **Strings** — `u8[]` alias; treated identically to a `u8` array.

### Functions

- Primitive scalars (`u8`, `i8`, `u16`, `i16`) are passed by value in registers (see `CallingConvention`).
- Arrays and structs are passed by reference (pointer in a 16-bit register).
- Structs and arrays cannot be returned from a function (by value); only primitives and pointers can.

### Equality and comparison operators

`=` is *equality* (not assignment); assignment uses `:=` or the statement form. `<>` is not-equal. Comparison results are `bit` (boolean), consumed directly in `BranchMode` — do not materialise as a byte unless the result is stored.

### Virtual Registers (VRs)

Because some CPU targets have compound registers the `VirtualRegister` maintains references to its "parent" VR and "component" VRs. For example, a 16-bit VR that must be allocated to `HL` will have two 8-bit component VRs (for the high and low bytes) linked to it. The instruction selector must maintain these links correctly when creating new VRs for compound values.

---

## Go conventions

- Module path is `zenith`; import sub-packages as `zenith/compiler/zsm`, `zenith/compiler/cfg`, etc.
- Use section divider comments `// ====...====` before logical groups within a file.
- Prefer an **interface + target-specific implementation** pattern:
  - Define e.g. `InstructionSelector` in `instruction_selector.go`.
  - Implement it in `instruction_selector_z80.go` as `instructionSelectorZ80` (unexported struct, exported constructor).
- Test files are co-located with the source they test (`_test.go`, same package).
- Generic helpers live in `compiler/common.go`; `OfType[T]` and `OfTypeInterface[T, I]` avoid interface-assertion boilerplate.
- Do **not** use named return values; always return explicitly.
- Errors must be wrapped with context: `fmt.Errorf("context: %w", err)`.

---

## Design principles

These principles are inferred from the implementation and must be respected when generating new code.

### Non-SSA VirtualRegister model

The compiler does **not** use Static Single Assignment (SSA) form. Each source-level variable maps to exactly one `VirtualRegister` for its entire lifetime (tracked via `InstructionSelectionContext.symbolToVReg`). Reassignments generate an explicit `LD r, r` move into the *existing* VR — they do **not** create a new VR or insert phi (φ) nodes. There are no phi nodes anywhere in the system. Do not introduce phi nodes, value renaming on reassignment, or SSA-style numbering.

### Two-level IR inside BasicBlock

`BasicBlock` holds two instruction lists simultaneously:
- `Instructions []zsm.SemStatement` — the semantic IR, populated by the CFG builder.
- `MachineInstructions []MachineInstruction` — generated by instruction selection.

Both live in the same block; instruction selection translates one into the other in-place. Do not conflate them.

### CFG preserves structured control flow

The CFG is built directly from structured source constructs (`if`/`elsif`/`for`/`select`). There is no lowering to unstructured gotos first. Block labels (`LabelIfThen`, `LabelForBody`, etc.) document the structural intent and are used to generate correct jump targets.

### Move-based assignment

All variable reassignments are lowered to explicit `LD r, r` move instructions at instruction-selection time. The optimizer/emitter may later eliminate no-op or redundant moves via pattern matching (see `emit/z80.md`). Do not try to "reuse" the result VR of an expression as the variable VR — always emit a `SelectMove`.

### Lazy stack frame; deferred prologue/epilogue

Only arrays and structs receive stack slots (allocated during `allocateFrameSlots`). Scalars live entirely in VRs. The entry and exit `BasicBlock`s are reserved placeholders; prologue and epilogue instructions are only emitted into them *after* instruction selection has determined whether a stack frame is needed at all.

### Constraint-first, two-pass register allocation

Register allocation is graph-colouring (Chaitin-style) over an interference graph derived from liveness analysis. Key points:
- VRs with a non-empty `AllowedSet` encode Z80 architectural constraints (e.g. `[A]` for arithmetic, `[HL]` for 16-bit results). These are processed first (`ConstrainedFirst` strategy).
- If any VRs remain unallocated after the coloring pass, a second pass (`ResolveUnallocated`) inserts move instructions at each instruction site to satisfy remaining constraints. This is the expected fallback, not an error.
- Multiple coloring strategies are tried in sequence (`ConstrainedFirst → ResultFirst → OperandFirst`) before falling back to the second pass.

### BranchMode avoids materialising booleans

When a boolean-producing expression (comparison, logical-and/or) is consumed directly by a branch, the `ExprContext` is set to `BranchMode`. The instruction selector emits a conditional jump directly (`JP cc, nn` or `JR cc, e`) without ever storing a 0/1 byte in a register. Only fall back to `ValueMode` when the boolean result genuinely needs to be stored.

### Instruction descriptors are the source of truth for machine facts

All architectural facts (opcode encoding, cycle counts, byte size, flag effects, operand kinds) live in `InstrDescriptor` variables in `instructions_z80.go`. Liveness analysis and the emitter rely on `AffectedFlags` / `DependentFlags` being accurate. Never hard-code these facts in instruction selection logic — always define or reuse a descriptor.

---

## Compiler pipeline

```
Source text
  → lexer/tokenizer.go   (tokens)
  → parser/parser.go     (AST nodes)
  → zsm/sem_analyzer.go  (semantic model, SemXxx nodes)
  → cfg/cfg.go           (Control Flow Graph, BasicBlock)
  → cfg/instruction_selection.go  (VirtualRegister machine instructions)
  → cfg/liveness.go      (liveness analysis)
  → cfg/interference.go  (interference graph)
  → cfg/register_allocation.go (graph-coloring allocation → physical registers)
  → emit/                (assembly text output)
```

---

## CFG and instruction selection

- All instructions operate on `VirtualRegister` values — never reference physical registers directly in selection logic. Look up the pre-defined register-set slices in `calling_convention_z80.go` and use the right width (8-bit or 16-bit) for the source type being compiled.
- Constrain a VR's allowed register set to the **minimum** the instruction genuinely requires. Over-constraining causes unnecessary spills; under-constraining produces incorrect code.
- `VirtualRegister` has an `AllowedSet` field and `ParentVR`/`ComponentVRs` links for 16-bit pairs — consult the existing code to understand the pattern before adding new VRs.
- Emit machine instructions via the selector's emit helpers (look at existing `Select*` methods for the pattern). Do not construct instructions outside of selector methods.
- Every new opcode **must** have a corresponding descriptor (in `cfg/instructions_z80.go`) with accurate flag-effect fields. Liveness analysis reads those fields — getting them wrong silently breaks register allocation.
- `ExprContext` carries the evaluation mode through the expression tree. Always propagate it:
  - **ValueMode** — result must land in a VR.
  - **BranchMode** — emit a conditional jump directly; do **not** materialise a boolean byte in a register.
- When a `TargetSymbol` is set on the context, use it to name or constrain the result VR — consult existing uses of `WithSymbol` for the pattern.
- For multiply/divide, Z80 has no hardware instructions — delegate to runtime helper calls. See existing `SelectMultiply` / `SelectDivide` for the established helper-call pattern.

---

## Z80-specific rules

- **8-bit arithmetic** always uses `A` as the accumulator. Load the operand into a VR constrained to `A` first, then emit the operation.
- **16-bit arithmetic** uses `HL` as the primary register. Clear carry before subtraction-with-carry (`SBC HL, rr`).
- **Z80 has no multiply or divide instructions.** Delegate to runtime helpers — see existing `SelectMultiply` and `SelectDivide` for the calling pattern and helper names.
- Prefer **`JR`** (relative jump) over `JP` (absolute jump) for short-range branches — it is one byte smaller.
- Use **`DJNZ`** for counted loops when the counter fits in `B`.
- Do **not** use IX or IY in interrupt handlers; use the alternate register set instead.
- Opcode prefixes are encoded in the high byte of the opcode constant (`CB`, `ED`, `DD`, `FD` prefixes). Consult the existing opcode definitions before adding new ones.

---

## Register allocation

Register allocation uses **graph colouring**. Key constraints to respect when writing instruction selection:

- Set `AllowedSet` as narrow as the instruction genuinely requires — over-constraining causes unnecessary spills, under-constraining produces wrong code.
- A second pass handles VRs that remain unallocated after the primary coloring pass by inserting moves — this is the expected fallback, not an error condition.
- When splitting a 16-bit result into 8-bit components, use the established helper in the selector rather than allocating the component VRs independently. Consult existing 16-bit operations for the pattern.

---

## Calling convention

A `CallingConvention` interface abstracts parameter and return-value placement. Key rules:

- Never hard-code which physical register receives a parameter or return value — always query the calling convention interface.
- Parameters in registers where possible; overflow spills to the stack.
- Consult `calling_convention.go` (interface) and `calling_convention_z80.go` (Z80 implementation) for the API.

---

## Liveness and interference

- Liveness analysis works at the `VirtualRegister` ID level; CPU flags are tracked as implicit VRs.
- When adding a new machine instruction type, implement `GetResult()` and `GetOperands()` correctly — liveness and the interference graph depend on them to determine which VRs are live simultaneously.

---

## Testing

- Run all tests: `go test ./...` from `src/`.
- Test function names: `Test<Phase>_<Scenario>` (e.g. `TestInstructionSelection_Add8Bit`).
- Prefer **table-driven tests** with a `tests []struct{ name string; input ...; want ... }` slice.
- Assertion helper: use `t.Errorf` (not `t.Fatalf`) unless a failure makes subsequent assertions meaningless.

---

## Documentation

- Design notes live in Markdown files next to the source they describe (e.g. `emit/z80.md`, `zsm/zsm.md`, `compiler/cfg/` has none yet).
- High-level architecture decisions live in `docs/compiler.md`.
- Inline code comments use the `// ====...====` section-divider style for major groups.
- Do **not** create a new Markdown file to document individual changes or generated code.
