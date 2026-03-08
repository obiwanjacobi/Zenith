# Zenith Compiler — Known Issues / TODO

---

## Parser

### ~~Bug 1: `expressionTypeInitializer` greedily captures `if`-body~~ ✅ Fixed

Added `inCondition bool` to `parserContext` and set it around every condition `expression()` call
(`statementIf`, `statementElsif`, `statementFor`). `expressionTypeInitializer` returns nil
immediately when `inCondition` is true, so `flag {body}` is never mistaken for a struct literal
inside a branch condition. Ordering in `expressionPrimary` restored (type-init before identifier)
so struct literals continue to work everywhere else.

---

## Semantic Analyser

### ~~Bug 2: `ExpressionPrecedence` not handled~~ ✅ Fixed

Added `case parser.ExpressionPrecedence:` to `processExpression` in
`compiler/zsm/sem_analyzer.go`, delegating to `sa.processExpression(n.Inner())`.

### Bug 3: Check return type of `ret` statements and function decl

---

## Machine Instructions Z80

- `SelectReturn` should receive the exit-block and emit a JP not a RET.
- block if.merge does not seem to be used.

---

## Planned work

- ~~After fixing parser Bug #1, rewrite `TestTACLowering_LogicalNotBranchInversion`~~ ✅ Both
  blockers fixed — test re-enabled and passing.
