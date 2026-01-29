# Zenith Semantic Model

## Post Parser Rules

- rules not covered by parser grammar
  - for-init variable declarions/assignment
  - expression operand precedence check
    - use of differen operators require use of grouping ()

## Type Checking

- builtin types
- determine type of expression
- resolve type-ref
  - respect scopes (no shadowing)
  - undefined symbols

## Function Inlining

- explicit inlining (compiler directive)
- implicit inlining (optimization)
  - determined by compiler based on rules (TBD)
