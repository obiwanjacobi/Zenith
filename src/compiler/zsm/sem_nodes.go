package zsm

import (
	"fmt"
	"zenith/compiler/parser"
)

// SemNode is the base interface for all semantic model nodes
type SemNode interface {
	ASTNode() parser.ParserNode // Reference back to original AST node
}

// SemDeclaration represents top-level declarations
type SemDeclaration interface {
	SemNode
}

// SemStatement represents executable statements
type SemStatement interface {
	SemNode
}

// SemExpression represents expressions that produce values
type SemExpression interface {
	SemNode
	Type() Type // All expressions have a resolved type
}

// ============================================================================
// Compilation Unit
// ============================================================================

type SemCompilationUnit struct {
	Declarations []SemDeclaration
	GlobalScope  *SymbolTable
	CallGraph    *CallGraph // Function call relationships
	astNode      parser.CompilationUnit
}

func (n *SemCompilationUnit) ASTNode() parser.ParserNode  { return n.astNode }
func (n *SemCompilationUnit) AST() parser.CompilationUnit { return n.astNode }

// ============================================================================
// Declarations
// ============================================================================

// SemVariableDecl represents a variable declaration
type SemVariableDecl struct {
	Symbol      *Symbol
	Initializer SemExpression // nil if no initializer
	TypeInfo    Type          // Resolved type
	astNode     parser.VariableDeclaration
}

func (n *SemVariableDecl) ASTNode() parser.ParserNode      { return n.astNode }
func (n *SemVariableDecl) AST() parser.VariableDeclaration { return n.astNode }

// SemFunctionDecl represents a function declaration
type SemFunctionDecl struct {
	Name       string
	Parameters []*Symbol
	ReturnType Type // nil for void
	Body       *SemBlock
	Scope      *SymbolTable
	astNode    parser.FunctionDeclaration
}

func (n *SemFunctionDecl) ASTNode() parser.ParserNode      { return n.astNode }
func (n *SemFunctionDecl) AST() parser.FunctionDeclaration { return n.astNode }

// SemTypeDecl represents a struct type declaration
type SemTypeDecl struct {
	TypeInfo *StructType
	astNode  parser.TypeDeclaration
}

func (n *SemTypeDecl) ASTNode() parser.ParserNode  { return n.astNode }
func (n *SemTypeDecl) AST() parser.TypeDeclaration { return n.astNode }

// ============================================================================
// Statements
// ============================================================================

// SemBlock represents a block of statements
type SemBlock struct {
	Statements []SemStatement
	astNode    parser.CodeBlock
}

func (n *SemBlock) ASTNode() parser.ParserNode { return n.astNode }
func (n *SemBlock) AST() parser.CodeBlock      { return n.astNode }

// SemAssignment represents a variable assignment
type SemAssignment struct {
	Target  *Symbol
	Value   SemExpression
	astNode parser.VariableAssignment
}

func (n *SemAssignment) ASTNode() parser.ParserNode     { return n.astNode }
func (n *SemAssignment) AST() parser.VariableAssignment { return n.astNode }

// SemIf represents an if statement
type SemIf struct {
	Condition   SemExpression
	ThenBlock   *SemBlock
	ElsifBlocks []*SemElsif
	ElseBlock   *SemBlock // nil if no else
	astNode     parser.StatementIf
}

func (n *SemIf) ASTNode() parser.ParserNode { return n.astNode }
func (n *SemIf) AST() parser.StatementIf    { return n.astNode }

// SemElsif represents an elsif clause
type SemElsif struct {
	Condition SemExpression
	ThenBlock *SemBlock
	astNode   parser.StatementElsif
}

func (n *SemElsif) ASTNode() parser.ParserNode { return n.astNode }
func (n *SemElsif) AST() parser.StatementElsif { return n.astNode }

// SemFor represents a for loop
type SemFor struct {
	Initializer SemStatement  // nil if not present
	Condition   SemExpression // nil if not present
	Increment   SemExpression // nil if not present
	Body        *SemBlock
	astNode     parser.StatementFor
}

func (n *SemFor) ASTNode() parser.ParserNode { return n.astNode }
func (n *SemFor) AST() parser.StatementFor   { return n.astNode }

// SemSelect represents a select statement (switch)
type SemSelect struct {
	Expression SemExpression
	Cases      []*SemSelectCase
	Else       *SemBlock // nil if no else
	astNode    parser.StatementSelect
}

func (n *SemSelect) ASTNode() parser.ParserNode  { return n.astNode }
func (n *SemSelect) AST() parser.StatementSelect { return n.astNode }

// SemSelectCase represents a case in a select statement
type SemSelectCase struct {
	Value   SemExpression
	Body    *SemBlock
	astNode parser.StatementSelectCase
}

func (n *SemSelectCase) ASTNode() parser.ParserNode      { return n.astNode }
func (n *SemSelectCase) AST() parser.StatementSelectCase { return n.astNode }

// SemExpressionStmt represents an expression used as a statement
type SemExpressionStmt struct {
	Expression SemExpression
	astNode    parser.StatementExpression
}

func (n *SemExpressionStmt) ASTNode() parser.ParserNode      { return n.astNode }
func (n *SemExpressionStmt) AST() parser.StatementExpression { return n.astNode }

// SemReturn represents a return statement
type SemReturn struct {
	Value   SemExpression // nil if no return value
	astNode parser.StatementReturn
}

func (n *SemReturn) ASTNode() parser.ParserNode  { return n.astNode }
func (n *SemReturn) AST() parser.StatementReturn { return n.astNode }

// ============================================================================
// Expressions
// ============================================================================

// SemConstant represents a constant literal value
type SemConstant struct {
	Value    interface{} // int, string, bool
	TypeInfo Type
	astNode  parser.Expression
}

func (n *SemConstant) ASTNode() parser.ParserNode { return n.astNode }
func (n *SemConstant) AST() parser.Expression     { return n.astNode }
func (n *SemConstant) Type() Type                 { return n.TypeInfo }

// SemSymbolRef represents a reference to a symbol (variable, parameter)
type SemSymbolRef struct {
	Symbol  *Symbol
	astNode parser.Expression
}

func (n *SemSymbolRef) ASTNode() parser.ParserNode { return n.astNode }
func (n *SemSymbolRef) AST() parser.Expression     { return n.astNode }
func (n *SemSymbolRef) Type() Type                 { return n.Symbol.Type }

// SemBinaryOp represents binary operations (arithmetic, comparison, logical, bitwise)
type SemBinaryOp struct {
	Op       BinaryOperator
	Left     SemExpression
	Right    SemExpression
	TypeInfo Type
	astNode  parser.ExpressionOperatorBinary
}

func (n *SemBinaryOp) ASTNode() parser.ParserNode           { return n.astNode }
func (n *SemBinaryOp) AST() parser.ExpressionOperatorBinary { return n.astNode }
func (n *SemBinaryOp) Type() Type                           { return n.TypeInfo }

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

// SemUnaryOp represents unary operations
type SemUnaryOp struct {
	Op       UnaryOperator
	Operand  SemExpression
	TypeInfo Type
	astNode  parser.ExpressionOperatorUnaryPrefix
}

func (n *SemUnaryOp) ASTNode() parser.ParserNode                { return n.astNode }
func (n *SemUnaryOp) AST() parser.ExpressionOperatorUnaryPrefix { return n.astNode }
func (n *SemUnaryOp) Type() Type                                { return n.TypeInfo }

type UnaryOperator int

const (
	OpNegate UnaryOperator = iota
	OpLogicalNot
	OpBitwiseNot
)

// SemFunctionCall represents a function call
type SemFunctionCall struct {
	Function  *Symbol
	Arguments []SemExpression
	TypeInfo  Type
	astNode   parser.ExpressionFunctionInvocation
}

func (n *SemFunctionCall) ASTNode() parser.ParserNode               { return n.astNode }
func (n *SemFunctionCall) AST() parser.ExpressionFunctionInvocation { return n.astNode }
func (n *SemFunctionCall) Type() Type                               { return n.TypeInfo }

// SemMemberAccess represents accessing a struct field
type SemMemberAccess struct {
	Object   *SemExpression
	Field    *StructField
	TypeInfo Type
	astNode  parser.ExpressionMemberAccess
}

func (n *SemMemberAccess) ASTNode() parser.ParserNode         { return n.astNode }
func (n *SemMemberAccess) AST() parser.ExpressionMemberAccess { return n.astNode }
func (n *SemMemberAccess) Type() Type                         { return n.TypeInfo }

// SemSubscript represents array subscripting (indexing)
type SemSubscript struct {
	Array    SemExpression
	Index    SemExpression
	TypeInfo Type
	astNode  parser.ExpressionSubscript
}

func (n *SemSubscript) ASTNode() parser.ParserNode        { return n.astNode }
func (n *SemSubscript) AST() parser.ExpressionSubscript   { return n.astNode }
func (n *SemSubscript) Type() Type                        { return n.TypeInfo }

// SemTypeInitializer represents struct initialization
type SemTypeInitializer struct {
	StructType *StructType
	Fields     []*SemFieldInit
	TypeInfo   Type
	astNode    parser.ExpressionTypeInitializer
}

func (n *SemTypeInitializer) ASTNode() parser.ParserNode            { return n.astNode }
func (n *SemTypeInitializer) AST() parser.ExpressionTypeInitializer { return n.astNode }
func (n *SemTypeInitializer) Type() Type                            { return n.TypeInfo }

// SemFieldInit represents a field initialization in a struct literal
type SemFieldInit struct {
	Field *StructField
	Value SemExpression
}

func DumpSemanticModel(semCU *SemCompilationUnit) {
	fmt.Println("========== Semantic Model ===========")
	fmt.Printf("Semantic Compilation Unit with %d declarations\n", len(semCU.Declarations))
	for _, decl := range semCU.Declarations {
		switch d := decl.(type) {
		case *SemFunctionDecl:
			fmt.Printf("  Function: %s (params=%d)\n",
				d.Name, len(d.Parameters))
		case *SemVariableDecl:
			fmt.Printf("  Variable: %s\n", d.Symbol.Name)
		case *SemTypeDecl:
			fmt.Printf("  Type: %s\n", d.TypeInfo.Name())
		default:
			fmt.Printf("  Unknown: %T\n", decl)
		}
	}
	fmt.Println()
}
