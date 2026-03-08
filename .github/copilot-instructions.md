# Copilot Code Generation Instructions

Guidelines for generating code in the Zenith compiler project.

Rule #1: don't generate code when not asked to.
Rule #2: after forming a plan for a fix or new feature, ask the user to confirm before generating any code.
Rule #3: Do not overthink the implementation. If the user asks for a fix or new feature, generate the most straightforward code that accomplishes the task. Add a comment if it is not the most optimal or elegant solution with what could be improved.

---

## Project overview

Zenith is a compiler for a custom language targeting 8-bit CPUs (Z80, with 6502 and others planned), written in Go.
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
    cfg/                          # CFG builder + TAC lowering (target-independent)
    cfg/z80/                      # Z80 instruction selector, register allocator, emitter
    cfg/6502/                     # (future) 6502 instruction selector etc.
    emit/                         # assembly emitter helpers
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

VRs are **immutable once assigned** — each expression result allocates a new VR with an `AllowedSet` encoding the Z80 instruction's register requirement. Distinct Go types model each kind of value:

- `TempVR` — an unallocated virtual register with an `AllowedSet` of candidate physical registers.
- `PhysVR` — a VR that has been assigned a physical register after allocation.
- `ImmVR` — a compile-time constant (never register-allocated).
- `StackVR` — a value permanently backed by a stack slot.

Register composition (`HL = H + L`) is a **hardware fact** encoded in the register descriptor (`Register.Composition`). To access a sub-register byte, allocate a new `TempVR` constrained to that sub-register (e.g. `{L}` for the low byte of HL). There are no parent/child pointer fields on VR types.

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

### SSA-style immutable VR model

Every expression result allocates a **new** VR. A VR's physical register assignment is **terminal** — once assigned, it is never changed. Concretely:

- Instruction selection never mutates a VR that has already been used as an operand elsewhere.
- No aliasing: two instructions holding the same VR pointer always agree on its state.
- Variable reassignments lower to an explicit `LD` copy into a new VR. The peephole pass eliminates any resulting no-op copies.

Explicit copy instructions are emitted at control-flow merge points (if/for join nodes) to unify values arriving from different paths. There are no phi (φ) nodes — copies serve the same purpose without requiring a phi-elimination step.

Source-level variables are tracked via `InstructionSelectionContext.symbolToVReg`, which maps each `Symbol` to its **current** live VR. On reassignment, `symbolToVReg` is updated to point to the new VR; the old VR is left unchanged.

### Three-level IR inside BasicBlock

`BasicBlock` holds three instruction lists that are populated in sequence:
- `SemInstructions []zsm.SemStatement` — high-level semantic IR from the CFG builder.
- `TAC []TACInstruction` — target-independent three-address code, produced by TAC lowering.
- `MachineInstructions []MachineInstruction` — target-specific instructions produced by the backend instruction selector.

Each stage populates its own list; do not conflate them or mix instructions across levels.

### CFG preserves structured control flow

The CFG is built directly from structured source constructs (`if`/`elsif`/`for`/`select`). There is no lowering to unstructured gotos first. Block labels (`LabelIfThen`, `LabelForBody`, etc.) document the structural intent and are used to generate correct jump targets.

### TAC lowering (target-independent)

TAC lowering translates `SemStatement` nodes into `TACInstruction`s. This step is **shared across all targets** — no target-specific knowledge belongs here.

TAC is typed by operand size (`ADD8`, `ADD16`, etc.) because 8-bit targets require completely different instruction sequences for different widths. Key TAC opcodes:

| Opcode | Meaning |
|---|---|
| `ADD8 / ADD16` | Integer addition |
| `SUB8 / SUB16` | Integer subtraction |
| `MUL8 / MUL16` | Multiplication (via runtime helper on Z80/6502) |
| `DIV8 / DIV16` | Division (via runtime helper) |
| `AND8 / OR8 / XOR8` | Bitwise ops |
| `LOAD8 / LOAD16` | Load from address + constant offset |
| `STORE8 / STORE16` | Store to address + constant offset |
| `LOAD_IDX8 / LOAD_IDX16` | Indexed load (dynamic index) |
| `STORE_IDX8 / STORE_IDX16` | Indexed store (dynamic index) |
| `INIT_SEQ` | Sequential store of literal values into contiguous memory (see below) |
| `COPY` | Move/copy a value (used at merge points in place of φ nodes) |
| `CALL` | Function call with explicit arg list and result temp |
| `BRANCH_IF` | Conditional branch to a block label |
| `JUMP` | Unconditional branch |
| `RETURN` | Function return with optional value |
| `STACK_ADDR` | Compute address of a stack slot |

**TAC is not always maximally flat.** When flattening would destroy information that targets need to generate optimal code, preserve it as a compound TAC node. The key example is `INIT_SEQ`.

**`INIT_SEQ`** represents sequential initialisation of contiguous memory (array and struct literals):
```
INIT_SEQ base=t1, element_size=1, values=[#1, #2, #3]
```
This tells the backend: compute the base address once, then store each value at successive offsets. On Z80 this maps to `LD (HL), n; INC HL` repeated — no per-element address recomputation. On 6502 it maps to `STA base,X` with X incrementing. The backend must not flatten `INIT_SEQ` into independent STORE instructions.

For individual element stores (e.g. `arr[i] := v`), use `STORE_IDX` — these are already atomic at source level.

**Decision rule:** if the source construct encodes information (e.g. "these stores are sequential from a known base") that would be destroyed by full flattening and is needed by at least one target to avoid redundant address computation, use a compound TAC node.

All variable reassignments lower to `COPY dst, src`. The peephole pass eliminates any resulting self-copies after register allocation.

### Lazy stack frame; deferred prologue/epilogue

Only arrays and structs receive stack slots (allocated during `allocateFrameSlots`). Scalars live entirely in VRs. The entry and exit `BasicBlock`s are reserved placeholders; prologue and epilogue instructions are only emitted into them *after* instruction selection has determined whether a stack frame is needed at all.

### Constraint propagation + linear scan

Register allocation is a **single-pass linear scan** over the instruction sequence. There is no retry loop, no second fallback pass, and no interference graph.

1. **Constraint propagation (copy coalescing):** Before the scan, walk copy instructions (`LD vrDst, vrSrc`) and propagate constraints — if one side is constrained to a single register and the other is unconstrained, the unconstrained VR inherits the same `AllowedSet`. This pre-colors VRs through copy chains without a separate analysis phase.
2. **Linear scan:** Walk instructions forward. `TempVR`s with a single-register `AllowedSet` are pre-colored. Unconstrained `TempVR`s are assigned greedily to any available register of the correct size not currently occupied by a live VR.
3. **Spill:** If no register is available, spill the longest-lived unconstrained `TempVR` to a stack slot and insert a reload before its next use.

Compound register topology is respected automatically: assigning a 16-bit VR to `HL` marks both `H` and `L` as occupied, using `Register.Composition` from the register descriptor. No per-VR tracking is needed.

### BranchMode avoids materialising booleans

When a boolean-producing expression (comparison, logical-and/or) is consumed directly by a branch, the `ExprContext` is set to `BranchMode`. The instruction selector emits a conditional jump directly (`JP cc, nn` or `JR cc, e`) without ever storing a 0/1 byte in a register. Only fall back to `ValueMode` when the boolean result genuinely needs to be stored.

### Instruction descriptors are the source of truth for machine facts

All architectural facts (opcode encoding, cycle counts, byte size, flag effects, operand kinds) live in `InstrDescriptor` variables in `instructions_z80.go`. Liveness analysis and the emitter rely on `AffectedFlags` / `DependentFlags` being accurate. Never hard-code these facts in instruction selection logic — always define or reuse a descriptor.

### Generate-then-clean (peephole)

Instruction selection emits **correct, straightforward code**. It does not try to avoid redundant moves or no-op copies — that is the sole responsibility of the peephole pass.

After register allocation, the peephole pass makes one forward scan and removes local patterns:
- `LD r, r` — self-move (result of two VRs pre-colored to the same register).
- `LD r1, r2; LD r2, r1` — back-and-forth pair.
- Other 1–3 instruction patterns documented in `emit/z80.md`.

Do **not** add complexity to `Select*` methods to avoid generating these patterns. All optimisation logic belongs exclusively in the peephole pass.

---

## Compiler pipeline

```
Source text
  → lexer/tokenizer.go        (tokens)
  → parser/parser.go          (AST nodes)
  → zsm/sem_analyzer.go       (semantic model, SemXxx nodes)
  → cfg/cfg.go                (Control Flow Graph, BasicBlock with SemInstructions)
  → cfg/tac_lowering.go       (TAC lowering — shared, target-independent)
  → cfg/<target>/selector.go  (instruction selection: TAC → MachineInstructions + TempVRs)
  → cfg/<target>/liveness.go  (live range computation)
  → cfg/<target>/regalloc.go  (constraint propagation + linear scan → PhysVRs)
  → cfg/<target>/peephole.go  (remove redundant moves, self-copies, etc.)
  → emit/<target>/            (assembly text output)
```

The split at `cfg/<target>/` is the target boundary. Everything above it is shared.

---

## TAC lowering

TAC lowering (`cfg/tac_lowering.go`) is the only place that reads `SemStatement` nodes. It is target-independent and must contain no Z80- or 6502-specific logic.

- Each `SemExpression` sub-tree is reduced to a sequence of TAC instructions; each intermediate value gets a fresh named temp.
- Compound source constructs that encode sequential-memory information (array literals, struct literals) must emit `INIT_SEQ`, **not** a sequence of `STORE` instructions — backends rely on this to avoid recomputing addresses per element.
- At control-flow merge points (end of `if` branches, start of `for` bodies) emit explicit `COPY dst, src` instructions to unify values. There are no φ nodes.
- `BranchMode`: when a boolean expression feeds directly into a `BRANCH_IF`, emit the `BRANCH_IF` with the condition temp; do not emit a `COPY` to a separate boolean temp.

## Target instruction selection

The target selector (`cfg/<target>/selector.go`) reads `TACInstruction`s and emits `MachineInstruction`s with `TempVR` operands.

- All instructions operate on VR values — never reference physical registers directly in selection logic. Look up the pre-defined register-set slices in the target's calling convention file.
- Every handler must return a **new** `TempVR` for its result. Never mutate a VR received as an operand — VRs are immutable once assigned.
- Constrain a VR's `AllowedSet` to exactly what the instruction requires. Over-constraining wastes registers; under-constraining produces wrong code.
- Sub-registers (e.g. `H`, `L`) are just `TempVR`s with a single-entry `AllowedSet`. The relationship to the parent register (`HL`) is encoded in `Register.Composition` — no parent/child fields on VR types.
- Emit machine instructions via the selector's emit helpers. Do not construct instructions outside of selector methods.
- Every new opcode **must** have a corresponding descriptor with accurate flag-effect fields. Liveness analysis reads those fields.
- For multiply/divide on Z80, there are no hardware instructions — delegate to runtime helper calls (`__mul8`, `__mul16`, `__div8`, `__div16`).

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

Register allocation uses **constraint propagation + linear scan**. Key rules:

- Set `AllowedSet` to exactly what the instruction genuinely requires. Over-constraining wastes registers; under-constraining produces wrong code.
- The allocator makes a **single forward pass**. There is no retry, no second fallback pass, and no interference graph construction.
- Compound register allocation (`HL` occupies both `H` and `L`) is handled automatically via `Register.Composition` — no per-VR tracking needed.
- Spills produce a `StackVR` with a reload inserted before its first use. The stack frame size is updated accordingly.
- To access a sub-register byte of a 16-bit result, allocate a new `TempVR` constrained to the sub-register (e.g. `{L}`). After allocation both will resolve to the correct physical register — the peephole pass removes any resulting self-moves.

---

## Calling convention

A `CallingConvention` interface abstracts parameter and return-value placement. Key rules:

- Never hard-code which physical register receives a parameter or return value — always query the calling convention interface.
- Parameters in registers where possible; overflow spills to the stack.
- Consult `calling_convention.go` (interface) and `calling_convention_z80.go` (Z80 implementation) for the API.

---

## Liveness analysis

Liveness analysis computes **live ranges** (first definition to last use) for each `TempVR`. These drive the linear-scan allocator's spill decisions.

- When adding a new machine instruction type, implement `GetResult()` and `GetOperands()` correctly — liveness depends on them to determine which VRs are simultaneously live.
- CPU flags are tracked implicitly via `AffectedFlags` / `DependentFlags` in the instruction descriptor. Do not add explicit flag VRs.

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
