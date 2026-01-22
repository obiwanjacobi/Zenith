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
| `i16` | 2    | -32786-32785 |
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
`x = i8(42)`    i8  conversion

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
```

Static instantiation syntax: `arr:u8[3] = (1, 2, 3)`

#### String

A string is an array of (ascii) characters.

```c
str: u8[] = "String"
```

#### Slice

> TBD

A slice is a reference to a part of an array.

The start is inclusive, the end is exclusive.

Syntax: `slice := arr[1,4]` from second-till fourth elelement.

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

### Boolean

> TBD

Also adds the `true` and `false` keywords.

Syntax: `bool`

```c
b: bool = true
```

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
- 

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

---

## Values and Variables

A variable is variable.
A value is constant.

Variable syntax: `x: u8`    default value
Constant value syntax: `const x: u8 = 42`  must be initialzed

Constant values are not stored in memory.

```c

```

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

#### Switch

> TBD: `select`-`case`?

Compiled to a jump table.

Syntax:

```c
switch <variable>
{
    case <value>:
    else:
}
```

No `break` keyword is needed. There is no fall-through in the `switch`-`case` statement.

---

## Symbols

Comment syntax: `// <text>` rest of the line is comment

Public Label Syntax: `label:`

Private Label Syntax: `.label`

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
| `*`      | Multiplication*      |
| `/`      | Division*            |
| `++`     | Increment            |
| `--`     | Decrement            |

*) Implemented in software.

The result type is the same as the biggest operand type unless the target assignment type is bigger.

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
| `>>>`    | arithmetic shift right   |
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

> TBD: Do we need to include (carry,zero) flags? Or are the comparison operators enough?

#### Logical

| Operator | Description |
| -------- | ----------- |
| `!`      | Not         |
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
| `#`      | Compiler directive / intrinsic / tags   |

## Keywords

Other than the ones already discussed.

| Keyword       | Description                |
| ------------- | -------------------------- |
| `in`          | IO input `r:u8 = in 0x32`  |
| `out`         | IO output `out 0x32, a`    |
| `ret`         | Return statement           |
| `brk`         | Break out of a scope       |
| `brk` <label> | Break out of scope 'label' |
| `cnt`         | Skip current iteration     |
| `cnt` <label> | Skip current iteration of <label> |
| `goto`        | ??                         |

> TBD: are 'in' and 'out' compiler intrinsics?

## Files

Multiple files can be compiled in parallel.

### Modules

A modules is named a collection of file where exported symbols can be (re)used by other code or modules.

#### Import Export

All [public lables](#symbols) are exported.

To import a symbol from a module use the qualified name: `<module>.<symbol>`

## Compiler

### Directives

All directives start with a `#`.

| Directive          | Description                            |
| ------------------ | -------------------------------------- |
| #if <const>        | Conditional compilation                |
| #elsif <const>     | Conditional compilation                |
| #else              | Conditional compilation                |
| #end               | Ends a compilation block               |
| #asm               | Inline assembly block (#end)           |
| #address <address> | Puts a symbol at a specific address    |
| #callconv <call>   | Select calling convention for function |

### Intrinsics

All intrinsics start with a `@`.

| Intrinsic                  | Description                     |
| -------------------------- | ------------------------------- |
| @movemem(src, dst, u/d, r) | LDI/LDIR/LDD/LDDR               |
| @findmem(src, f, u/d, r)   | CPI/CPIR/CPD/CPDR               |
| @carry(false/true/not)     | Clear, set or toggle carry flag |

- BCD/DAA?

### Configuration

The compiler can be configured to suit the hardware that is being coded for best.

| Setting   | Description                                                 |
| -------   | ----------------------------------------------------------- |
| output    | What output file to generate (asm (what flavor?), hex, elf) |
| z80-undoc | use z80 undocumented assembly instructions                  |

> TBD:

- Memory Layout (where is rom, ram - how big)
- Memory Bank Switching
