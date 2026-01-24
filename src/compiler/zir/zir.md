# Zenith Intermediate Represenation

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

## Flow Analysis

- call flow graph
- data flow analysis

## Register Allocation

- calling convention
  - use registers as much as possible (fixed convention)
  - spill-over onto the stack
    - cleanup on return?
  - internal/public or just one?

- virtual registers
- 3-address code (is this applicable to Z80 assembly?)
- graph coloring (the flow of registers against use of vars)

## Model Optimization

- intermediate model level
  - ??
- low level (call flow graph)
  - optimization rules when patterns are detected

## Emit Phase Preparation

here or in emit?

- emit assmbly (text)
  - which flavor (assembler target)?
- emit instructions
  - instruction encoding