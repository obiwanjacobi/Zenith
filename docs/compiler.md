# Compiler

Details on the compiler construction.

- What constructs prevent from generating efficient code?
- Use cycle count to decide what registers (IX/IY!) and instructions to use.
- Optimize for size, speed or both.
	- Allow indicating performance critical sections
- Enable low level IO addressing and peripheral support (PIO/SIO/CTC/DMA)
- https://retrocomputing.stackexchange.com/questions/6095/why-do-c-to-z80-compilers-produce-poor-code
- https://retrocomputing.stackexchange.com/questions/15004/what-languages-are-better-fit-for-generating-efficient-code-for-popular-8-bit-cp

## Function Parameters and Return value

- Registers: use as much as possible for parameter passing and return value
- minimize stack usage
- reduce register shuffling
- optimize small function call overhead (inline expansion)
- overflow parameters onto the stack -or- a central place in memory (not used for interrupts).

### Interrupt handling

- do not use IX/IY
- use the alternate register set (reentrant interrupts? NMI?)

## Memory Management

- What memory management feature are required?
- Do we allow dynamic memory allocation? (how to make it safe(r))
- allow segmented bounds checking (parts of the code can be optimized when they're debugged and finished)
- debug-fill allocated memory.
- support/abstract custom memory bank switching
- Explicit Memory model
- Memory Segmentation
  - Support for 16bit address space (default)
  - Far/near pointers + relative pointers (handles?)
  - align data placement with bank segmentation
  - Integrate custom Bank/memory switching mechanism (tagging functions that do the switching and are called by the compiler when needed)
  - Allow code to be mapped to specific memory regions (banks)
- Memory Access Patterns
  - Direct mem access
  - Pointer arithmetic (minimal overhead)
  - Compile-time mem layout optimization
- Memory safety constraints
  - directed bounds and safety checks
  - explicit end efficient mem (de)allocation
