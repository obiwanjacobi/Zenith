package zir

// Type represents a resolved type in the IR
type Type interface {
	Name() string
	Size() int // Size in bytes for Z80
}

// PrimitiveType represents built-in types like u8, i8, d8, bool
type PrimitiveType struct {
	name string
	size int
}

func (t *PrimitiveType) Name() string { return t.name }
func (t *PrimitiveType) Size() int    { return t.size }

// ArrayType represents fixed-size arrays
type ArrayType struct {
	elementType Type
	length      int // 0 for unsized arrays
}

func (t *ArrayType) Name() string {
	if t.length > 0 {
		return t.elementType.Name() + "[" + string(rune(t.length)) + "]"
	}
	return t.elementType.Name() + "[]"
}

func (t *ArrayType) Size() int {
	if t.length > 0 {
		return t.elementType.Size() * t.length
	}
	return 2 // Pointer size for unsized arrays
}

func (t *ArrayType) ElementType() Type { return t.elementType }
func (t *ArrayType) Length() int       { return t.length }

// StructType represents user-defined struct types
type StructType struct {
	name   string
	fields []*StructField
	size   int // Computed from fields
}

type StructField struct {
	Name   string
	Type   Type
	Offset int // Byte offset within struct
}

func (t *StructType) Name() string           { return t.name }
func (t *StructType) Size() int              { return t.size }
func (t *StructType) Fields() []*StructField { return t.fields }
func (t *StructType) Field(name string) *StructField {
	for _, f := range t.fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// PointerType represents pointer types (u8*, etc.)
type PointerType struct {
	pointeeType Type
}

func (t *PointerType) Name() string {
	return t.pointeeType.Name() + "*"
}

func (t *PointerType) Size() int {
	return 2 // Pointers are always 2 bytes on Z80
}

func (t *PointerType) PointeeType() Type { return t.pointeeType }

// FunctionType represents function signatures (for function pointers)
type FunctionType struct {
	parameters []Type
	returnType Type // nil for void
}

func (t *FunctionType) Name() string {
	// Could generate a signature string if needed
	return "function"
}

func (t *FunctionType) Size() int {
	return 2 // Function pointer size
}

func (t *FunctionType) Parameters() []Type { return t.parameters }
func (t *FunctionType) ReturnType() Type   { return t.returnType }

// Built-in primitive types
var (
	// Unsigned types
	U8Type  = &PrimitiveType{"u8", 1}
	U16Type = &PrimitiveType{"u16", 2}

	// Signed types
	I8Type  = &PrimitiveType{"i8", 1}
	I16Type = &PrimitiveType{"i16", 2}

	// BCD (Binary Coded Decimal) types
	D8Type  = &PrimitiveType{"d8", 1}
	D16Type = &PrimitiveType{"d16", 2}

	// Boolean type
	BoolType = &PrimitiveType{"bool", 1}
)

// NewArrayType creates a new array type
func NewArrayType(elementType Type, length int) *ArrayType {
	return &ArrayType{
		elementType: elementType,
		length:      length,
	}
}

// NewPointerType creates a new pointer type
func NewPointerType(pointeeType Type) *PointerType {
	return &PointerType{
		pointeeType: pointeeType,
	}
}

// NewStructType creates a new struct type with computed field offsets
func NewStructType(name string, fields []*StructField) *StructType {
	offset := 0
	for _, field := range fields {
		field.Offset = offset
		offset += field.Type.Size()
	}
	return &StructType{
		name:   name,
		fields: fields,
		size:   offset,
	}
}

// NewFunctionType creates a new function type (for function pointers)
func NewFunctionType(parameters []Type, returnType Type) *FunctionType {
	return &FunctionType{
		parameters: parameters,
		returnType: returnType,
	}
}
