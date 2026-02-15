# Zentih Language

The goal is to create a language that understands the Z80's unique architectural constraints while providing developers with a clean, efficient programming interface that generates optimal machine code.

## Types

### Primitives

`u` - unsigned
`i` - signed
`d` - decimal (BCD)

| Type  | Size | Desciption   |
| ----- | ---- | ------------ |
| `u8`  | 1    | 0-255        |
| `i8`  | 1    | -128-127     |
| `u16` | 2    | 0-65535      |
| `i16` | 2    | -32768-32767 |
| `u24` | 3    | ?            |
| `d8`  | 1    | BCD: 0-99    |
| `d16` | 2    | BCD: 0-9999  |

#### Literals

Default type for a numerical literal is the smallest unsigned type that will fit the value.
Unless the value is negative, then it is the smallest signed type.
If the target it is assigned to is explicitly typed, that type will be used, unless it is incompatible then an error is generated.

`x := 42`       u8
`x := 420`      u16
`x := -128`     i16
`x:u16 = 42`    u16 typed target
`x = i8(42)`    i8  conversion (from u8)

If the literal does not fit in a primitve type a compiler error is generated.
Use a conversion function or a explicitly typed target.

### Array

An array is stored as a ponter and a length (capacity) in memory.

Type syntax: `<type>[<len>]`

```c
arr: u8[3]
```

Indexing syntax: `<arr>[<index>]`

```c
arr: u8[3]
arr[0] = 0

l = arr.length  // l=3
```

Static instantiation syntax: `arr:u8[3] = (1, 2, 3)`

Implementation:

```c
// variable length struct
struct Array
    len: u16    // length of array (length+2 = total size)
    arr[]       // allocated array elements
```

The type of the array elements is tracked during compile time - no runtime type info.
The compiler will generate the correct indexing code taking the element size into account.

#### Slice

> TBD

A slice is a reference to a part of an array.

The start is inclusive, the end is exclusive.

Syntax: `slice := arr[1,4]` from second-till fourth elelement.

Implementation:

```c
// fixed length struct
struct Slice
    // TBD: small slices (u8)? Or large slices (u16)?
    len: u8|u16?
    ptr: u8* // points to the start of the slice into the array elements
```

The type of the array elements is tracked during compile time and is based on the array that is pointed to.
The compiler will generate the correct indexing code taking the element size into account.

> TBD

Not sure if we allow slices to reinterpret the array-element type.
This would allow a `arr: u8[]` to be sliced to a `slice: u16[]` for instance.
-What if the originating array does not fit exactly into the slice (odd number of elements)?

#### String

A string is an array of (ascii) characters.

```c
str: u8[] = "String"

illegal: u16[] = "Invalid Assignment"   // error: must be u8
```

### Pointer

Type syntax: `<type>*`

```c
u8*
any*        //  void pointer
```

Ref syntax: `&<var>`

```c
&val
```

Deref syntax: `*<var>`

```c
*ptr
```

Null pointer: `ptr u8* = nil`

#### Function Pointers

Syntax: `<fn>`

```c
myFn: (p1: u8, p2: i16) u8
fnPtr: fn(u8)u8* = myFn
```

> TBD: function pointer type?

### Struct

A grouping of named (and typed) data elements.

Syntax: `struct <name> { <fields> }`

```c
struct data { cnt: u8, arr: u8[5] }
```

Construction Syntax:

```c
instance := { cnt = 42, arr = "hello" }`   // inferred type (matched on field names and data types)
instance : data = { cnt = 42, arr = "hello" }
```

Nesting Structs:

```c
struct Address {}
struct Person { address: Address }

instance: Person = { address = { ... } }
```

Struct Pointers:

```c
instance.cnt++    // direct struct instance access
ptr := &instance  // make a ptr to instance
ptr.cnt++         // struct access via ptr
```

Accessing a struct instance directly or via a pointer always uses `.`.

> TBD: anonymous structs?

### Bit

Also adds the `true` and `false` keywords.

Syntax: `bit` or `bit[4]`

```c
b0: bit = true              // bool
b1: bit[1] = true           // bool / 0b1
b2: bit[2] = 0b10
b4: bit[4] = 0b1011

// equivalent
b4[2] = true
b4[2] = 1

if b4[2] {
    // b4[2] == true
}
```

Allows manipulating any number of bits. A bool is just a `bit-1`. The `true` and `false` keywords still apply to any bit.

### Alias

All types (incl. `struct`s) can be aliased.

Syntax: `type <alias> = <type>`

---

## Functions

'<params>' and '<ret> are optional (void).

Syntax: `<label> (<params>) <ret> { <fn body> }`

> TBD: syntax of public label (sum)

```c
sum: (x: u8, y: u8) u16 { ret x + y }
```

Invocation syntax: `result := sum(101, 42)`

### Parameters and Return

- Primitive types can be passed by value (param and return) - except u24?.
- Structs cannot be passed by value (param and return).
- Structs and Arrays (of Structs) cannot be returned from a function.
- Arrays/structs are passed by ref as param.

### Conversions

Syntax: `<type>(<value>)`

```c
x:= i16(42)
```

### Special Functions

For Z80 instructions like RST0-RST38 and Interrupts.

Use a tag to indicate special use.

```c
#address 0x20
reset20: () { ... }
```

There are also compiler-intrinsic functions. See [Compiler](#intrinsics) for more info.

---

## Values and Variables

A variable is variable.
A value is constant.

Variable syntax: `x: u8`    default value
Constant value syntax: `const x: u8 = 42`  must be initialzed

Constant values are not stored in memory but are managed during compilation.

Global variables are stored in the 'global' memory.
Memory layout configuration dictates where that is and how big the space is.

Variables used inside functions are kept in registers as much as possible or stored on stack.

---

## Flow Control

### Loops

The 'init' and 'next' parts are optional (like in Go).
For loop syntax: `for <init>; <condition>; <next> { <body> }`

```c
for i:=0; i < 3; i++ { ... }
```

```c
for i < 3 { i++ }   // while loop
```

### Conditional Branching

#### If, Elsif and Else

No '()' are required around condition.

Syntax: `if <condition> { ... } elsif <condition> { ... } else { ... }`

```c
a := 42
if a = 42 {
    ...
} elsif a == 0 {
    ...
} else {
    ...
}
```

#### Select-Case

Compiled to a jump table?

Syntax:

```c
select <variable>
{
    case <value>:
    else:
}
```

No `break` keyword is needed. There is no fall-through in the `select`-`case` statement.

---

## Symbols

Comment syntax: `// <text>` rest of the line is comment

Label Syntax: `label:`

Qualified Name: `<module>.<symbol>`

## Expressions

Precedence:

- Arithmetic
- Bitwise
- Comparison
- Logical

The compiler will try to result (parts of) expressions at compile-time as much as possible.

### Operators

#### Arithmetic

| Operator | Description          |
| -------- | -------------------- |
| `+`      | Addition             |
| `-`      | Subtraction          |
| `+c`     | Addition /w carry    |
| `-c`     | Subtraction /w carry |
| `*`      | Multiplication       |
| `/`      | Division             |
| `%`      | Modulo               |
| `++`     | Increment            |
| `--`     | Decrement            |

The result type is the same as the biggest operand type unless the target assignment type is bigger. The result type for Multiplication is always double-the-operands.

```c
x:u8 = 101
y:u8 = 42
z:u16 = x + y
```

#### Bitwise

| Operator | Description              |
| -------- | ------------------------ |
| `&`      | And                      |
| `\|`     | Or                       |
| `~`      | Negate/Invert            |
| `^`      | Exclusive Or             |
| `>>`     | Logical shift right      |
| `>>>`    | Arithmetic shift right   |
| `<<`     | Shift left               |
| `>\|`    | Roll right               |
| `\|<`    | Roll left                |
| `>\|c`   | Roll right through carry |
| `\|<c`   | Roll left through carry  |

The result type is the same as the biggest operand type unless the target assignment type is bigger.

#### Comparison

| Operator | Description                 |
| -------- | --------------------------- |
| `=`      | Equals                      |
| `<>`     | Not Equals                  |
| `>`      | Greater                     |
| `<`      | Lesser                      |
| `>=`     | Greater or Equal            |
| `<=`     | Lesser or Equal             |
| `<f>?`   | Test a flag (c, z, s, n, p) |

The result type is a `bool`.

#### Logical

| Operator | Description |
| -------- | ----------- |
| `not`    | Not         |
| `and`    | and         |
| `or`     | Or          |
| `?`      | Bool*       |

> *) TBD: Boolean operator, if applicable. Makes conditional branches easier.

#### Other

All arithmetic (except `++` and `--`) and bitwise operators can be used in this form:

`a += 3`
`a |= 0x80`

| Operator | Description                             |
| -------- | --------------------------------------- |
| `=`      | Assignment                              |
| `()`     | Operator Precedence, List instantiation |
| `{}`     | Scope Block, Object construction        |
| `#`      | Compiler directive / tags               |
| `@`      | Compiler intrinsic                      |

## Keywords

Other than the ones already discussed.

| Keyword       | Description                |
| ------------- | -------------------------- |
| `ret`         | Return statement           |
| `brk`         | Break out of a scope       |
| `brk` <label> | Break out of scope 'label' |
| `cnt`         | Skip current iteration     |
| `cnt` <label> | Skip current iteration of <label> |
| `goto`        | ??                         |

## Files

Multiple files can be compiled in parallel.

Their syntax trees will be combined before semantic analysis. This means they all share the same namespace.

### Modules

A modules is named a collection of file where exported symbols can be (re)used by other code or modules.

#### Import / Export

All [lables](#symbols) are exported.

To import a symbol from a module use the qualified name: `<module>.<symbol>`

## Compiler

### Directives

All directives start with a `#`.

| Directive          | Description                            |
| ------------------ | -------------------------------------- |
| #if <const>        | Conditional compilation                |
| #ifn <const>       | Conditional compilation                |
| #elsif <const>     | Conditional compilation                |
| #elsifn <const>    | Conditional compilation                |
| #else              | Conditional compilation                |
| #end               | Ends a compilation block               |
| #asm               | Inline assembly block (#end)           |
| #address <address> | Puts a symbol at a specific address    |
| #callconv <call>   | Select calling convention for function |

### Intrinsics

All intrinsics start with a `@`.

| Intrinsic                  | Description                     |
| -------------------------- | ------------------------------- |
| `@movemem(src, dst, u/d, r)` | LDI/LDIR/LDD/LDDR             |
| `@findmem(src, f, u/d, r)`   | CPI/CPIR/CPD/CPDR             |
| `@carry(false/true/not)`     | Clear, set or toggle carry flag |
| `@in`                        | IO input: IN                  |
| `@out`                       | IO output: OUT                |
| `@len(any[])`                | Returns the length of an array type |

> TBD: naming. Perhaps `@memory_move()` and `@memory_find()` etc. is better?

- Provide prolog/epilog 'macros' for working with the calling conventions for custom asm code.

### Configuration

The compiler can be configured to suit the hardware that is being coded for best.

| Setting   | Description                                                 |
| -------   | ----------------------------------------------------------- |
| output    | What output file to generate (asm (what flavor?), hex, elf) |

> TBD:

- Memory Layout (where is rom, ram - how big)
- Memory Bank Switching
