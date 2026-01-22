package zim

import "zenith/compiler/parser"

// IRNode is the base interface for all IR nodes
type IRNode interface {
	ASTNode() parser.ParserNode // Reference back to original AST node
}

// IRDeclaration represents top-level declarations
type IRDeclaration interface {
	IRNode
}

// IRStatement represents executable statements
type IRStatement interface {
	IRNode
}

// IRExpression represents expressions that produce values
type IRExpression interface {
	IRNode
	Type() Type // All expressions have a resolved type
}

// ============================================================================
// Compilation Unit
// ============================================================================

type IRCompilationUnit struct {
	Declarations []IRDeclaration
	GlobalScope  *SymbolTable
	Types        map[string]Type // User-defined types
	astNode      parser.CompilationUnit
}

func (n *IRCompilationUnit) ASTNode() parser.ParserNode  { return n.astNode }
func (n *IRCompilationUnit) AST() parser.CompilationUnit { return n.astNode }

// ============================================================================
// Declarations
// ============================================================================

// IRVariableDecl represents a variable declaration
type IRVariableDecl struct {
	Symbol      *Symbol
	Initializer IRExpression // nil if no initializer
	Type        Type         // Resolved type
	astNode     parser.VariableDeclaration
}

func (n *IRVariableDecl) ASTNode() parser.ParserNode      { return n.astNode }
func (n *IRVariableDecl) AST() parser.VariableDeclaration { return n.astNode }

// IRFunctionDecl represents a function declaration
type IRFunctionDecl struct {
	Name       string
	Parameters []*Symbol
	ReturnType Type // nil for void
	Body       *IRBlock
	Scope      *SymbolTable
	astNode    parser.FunctionDeclaration
}

func (n *IRFunctionDecl) ASTNode() parser.ParserNode      { return n.astNode }
func (n *IRFunctionDecl) AST() parser.FunctionDeclaration { return n.astNode }

// IRTypeDecl represents a struct type declaration
type IRTypeDecl struct {
	Type    *StructType
	astNode parser.TypeDeclaration
}

func (n *IRTypeDecl) ASTNode() parser.ParserNode  { return n.astNode }
func (n *IRTypeDecl) AST() parser.TypeDeclaration { return n.astNode }

// ============================================================================
// Statements
// ============================================================================

// IRBlock represents a block of statements with its own scope
type IRBlock struct {
	Statements []IRStatement
	Scope      *SymbolTable
	astNode    parser.CodeBlock
}

func (n *IRBlock) ASTNode() parser.ParserNode { return n.astNode }
func (n *IRBlock) AST() parser.CodeBlock      { return n.astNode }

// IRAssignment represents a variable assignment
type IRAssignment struct {
	Target  *Symbol
	Value   IRExpression
	astNode parser.VariableAssignment
}

func (n *IRAssignment) ASTNode() parser.ParserNode     { return n.astNode }
func (n *IRAssignment) AST() parser.VariableAssignment { return n.astNode }

// IRIf represents an if statement
type IRIf struct {
	Condition   IRExpression
	ThenBlock   *IRBlock
	ElsifBlocks []*IRElsif
	ElseBlock   *IRBlock // nil if no else
	astNode     parser.StatementIf
}

func (n *IRIf) ASTNode() parser.ParserNode { return n.astNode }
func (n *IRIf) AST() parser.StatementIf    { return n.astNode }

// IRElsif represents an elsif clause
type IRElsif struct {
	Condition IRExpression
	ThenBlock *IRBlock
	astNode   parser.StatementElsif
}

func (n *IRElsif) ASTNode() parser.ParserNode { return n.astNode }
func (n *IRElsif) AST() parser.StatementElsif { return n.astNode }

// IRFor represents a for loop
type IRFor struct {
	Initializer IRStatement  // nil if not present
	Condition   IRExpression // nil if not present
	Increment   IRExpression // nil if not present
	Body        *IRBlock
	astNode     parser.StatementFor
}

func (n *IRFor) ASTNode() parser.ParserNode { return n.astNode }
func (n *IRFor) AST() parser.StatementFor   { return n.astNode }

// IRSelect represents a select statement (switch)
type IRSelect struct {
	Expression IRExpression
	Cases      []*IRSelectCase
	Else       *IRBlock // nil if no else
	astNode    parser.StatementSelect
}

func (n *IRSelect) ASTNode() parser.ParserNode  { return n.astNode }
func (n *IRSelect) AST() parser.StatementSelect { return n.astNode }

// IRSelectCase represents a case in a select statement
type IRSelectCase struct {
	Value   IRExpression
	Body    *IRBlock
	astNode parser.StatementSelectCase
}

func (n *IRSelectCase) ASTNode() parser.ParserNode      { return n.astNode }
func (n *IRSelectCase) AST() parser.StatementSelectCase { return n.astNode }

// IRExpressionStmt represents an expression used as a statement
type IRExpressionStmt struct {
	Expression IRExpression
	astNode    parser.StatementExpression
}

func (n *IRExpressionStmt) ASTNode() parser.ParserNode      { return n.astNode }
func (n *IRExpressionStmt) AST() parser.StatementExpression { return n.astNode }

// ============================================================================
// Expressions
// ============================================================================

// IRConstant represents a constant literal value
type IRConstant struct {
	Value   interface{} // int, string, bool
	typ     Type
	astNode parser.ExpressionLiteral
}

func (n *IRConstant) ASTNode() parser.ParserNode    { return n.astNode }
func (n *IRConstant) AST() parser.ExpressionLiteral { return n.astNode }
func (n *IRConstant) Type() Type                    { return n.typ }

// IRSymbolRef represents a reference to a symbol (variable, parameter)
type IRSymbolRef struct {
	Symbol  *Symbol
	astNode parser.Expression
}

func (n *IRSymbolRef) ASTNode() parser.ParserNode { return n.astNode }
func (n *IRSymbolRef) AST() parser.Expression     { return n.astNode }
func (n *IRSymbolRef) Type() Type                 { return n.Symbol.Type }

// IRBinaryOp represents binary operations (arithmetic, comparison, logical, bitwise)
type IRBinaryOp struct {
	Op      BinaryOperator
	Left    IRExpression
	Right   IRExpression
	typ     Type
	astNode parser.ExpressionOperatorBinary
}

func (n *IRBinaryOp) ASTNode() parser.ParserNode           { return n.astNode }
func (n *IRBinaryOp) AST() parser.ExpressionOperatorBinary { return n.astNode }
func (n *IRBinaryOp) Type() Type                           { return n.typ }

type BinaryOperator int

const (
	// Arithmetic
	OpAdd BinaryOperator = iota
	OpSubtract
	OpMultiply
	OpDivide
	// Bitwise
	OpBitwiseAnd
	OpBitwiseOr
	OpBitwiseXor
	// Comparison
	OpEqual
	OpNotEqual
	OpLessThan
	OpLessEqual
	OpGreaterThan
	OpGreaterEqual
	// Logical
	OpLogicalAnd
	OpLogicalOr
)

// IRUnaryOp represents unary operations
type IRUnaryOp struct {
	Op      UnaryOperator
	Operand IRExpression
	typ     Type
	astNode parser.ExpressionOperatorUnaryPrefix
}

func (n *IRUnaryOp) ASTNode() parser.ParserNode                { return n.astNode }
func (n *IRUnaryOp) AST() parser.ExpressionOperatorUnaryPrefix { return n.astNode }
func (n *IRUnaryOp) Type() Type                                { return n.typ }

type UnaryOperator int

const (
	OpNegate UnaryOperator = iota
	OpNot
	OpBitwiseNot
)

// IRFunctionCall represents a function call
type IRFunctionCall struct {
	Function  *Symbol
	Arguments []IRExpression
	typ       Type
	astNode   parser.ExpressionFunctionInvocation
}

func (n *IRFunctionCall) ASTNode() parser.ParserNode               { return n.astNode }
func (n *IRFunctionCall) AST() parser.ExpressionFunctionInvocation { return n.astNode }
func (n *IRFunctionCall) Type() Type                               { return n.typ }

// IRMemberAccess represents accessing a struct field
type IRMemberAccess struct {
	Object  *IRExpression
	Field   *StructField
	typ     Type
	astNode parser.ExpressionMemberAccess
}

func (n *IRMemberAccess) ASTNode() parser.ParserNode         { return n.astNode }
func (n *IRMemberAccess) AST() parser.ExpressionMemberAccess { return n.astNode }
func (n *IRMemberAccess) Type() Type                         { return n.typ }

// IRTypeInitializer represents struct initialization
type IRTypeInitializer struct {
	StructType *StructType
	Fields     []*IRFieldInit
	typ        Type
	astNode    parser.ExpressionTypeInitializer
}

func (n *IRTypeInitializer) ASTNode() parser.ParserNode            { return n.astNode }
func (n *IRTypeInitializer) AST() parser.ExpressionTypeInitializer { return n.astNode }
func (n *IRTypeInitializer) Type() Type                            { return n.typ }

// IRFieldInit represents a field initialization in a struct literal
type IRFieldInit struct {
	Field *StructField
	Value IRExpression
}
