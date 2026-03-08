# Zenith Compiler — Known Issues / TODO

---

## Parser

### Bug 1: `expressionTypeInitializer` greedily captures `if`-body

**File:** `compiler/parser/parser_rules.go` — `expressionTypeInitializer` / `expressionPrimary`

**Symptom:** An `if` condition that ends with a bare identifier (e.g. `if not flag`) causes a parse
error: *"expected code block after condition"*.

**Root cause:** `expressionPrimary` tries `expressionTypeInitializer` as one of its alternatives.
`expressionTypeInitializer` calls `typeReference()` first, which succeeds on any identifier, then
calls `typeInitializer()` which looks for `{` — and finds the opening brace of the `if`-body.  
The entire block is consumed as though it were a struct-literal initializer, leaving no `{` for the
`if` rule to find.

**Affected syntax:**
```
if not flag {       // 'flag {' parsed as struct literal → error
    x = 1
}
```

**Fix direction:** In `expressionPrimary`, move `expressionTypeInitializer` after
`expressionIdentifier` (or after all other primaries), or add a look-ahead guard in
`expressionTypeInitializer` that requires the trailing `{` to look like a struct literal (e.g. must
be at the same line, or must be preceded by a known type name rather than a value-context
identifier).

**Blocked test:** `TestTACLowering_LogicalNotBranchInversion` in
`compiler/cfg/tac_lowering_test.go`

---

## Semantic Analyser

### ~~Bug 2: `ExpressionPrecedence` not handled~~ ✅ Fixed

Added `case parser.ExpressionPrecedence:` to `processExpression` in
`compiler/zsm/sem_analyzer.go`, delegating to `sa.processExpression(n.Inner())`.

### Bug 3: Check return type of `ret` statements and function decl

---

## Planned work

- After fixing parser Bug #1, rewrite `TestTACLowering_LogicalNotBranchInversion` to use
  `if not (x < 10) {}` directly (no intermediate `flag: bit` variable). The test verifies that
  `not (comparison)` emits a fused `TacBranchCond` (then/else swapped) rather than a
  `TacUnary(BNOT/NEG)`.
