package zim

import (
	"fmt"
	"zenith/compiler/lexer"
	"zenith/compiler/parser"
)

// SemanticAnalyzer performs semantic analysis on the AST and builds the IR
type SemanticAnalyzer struct {
	globalScope  *SymbolTable
	currentScope *SymbolTable
	types        map[string]Type
	errors       []error
}

// NewSemanticAnalyzer creates a new semantic analyzer
func NewSemanticAnalyzer() *SemanticAnalyzer {
	sa := &SemanticAnalyzer{
		types:  make(map[string]Type),
		errors: make([]error, 0),
	}
	return sa
}

// Analyze performs semantic analysis on the AST and returns the IR
func (sa *SemanticAnalyzer) Analyze(ast parser.CompilationUnit) (*IRCompilationUnit, []error) {
	// Initialize global scope
	sa.globalScope = NewSymbolTable(nil)
	sa.currentScope = sa.globalScope
	sa.initBuiltinTypes()

	// Pass 1: Register all top-level declarations (types, functions, globals)
	// This allows forward references to work
	for _, decl := range ast.Declarations() {
		sa.registerDeclaration(decl)
	}

	// Pass 2: Build IR with full type checking and resolution
	irDecls := make([]IRDeclaration, 0, len(ast.Declarations()))
	for _, decl := range ast.Declarations() {
		irDecl := sa.processDeclaration(decl)
		if irDecl != nil {
			irDecls = append(irDecls, irDecl)
		}
	}

	return &IRCompilationUnit{
		Declarations: irDecls,
		GlobalScope:  sa.globalScope,
		Types:        sa.types,
		astNode:      ast,
	}, sa.errors
}

// ============================================================================
// Built-in Types Initialization
// ============================================================================

func (sa *SemanticAnalyzer) initBuiltinTypes() {
	sa.types["u8"] = U8Type
	sa.types["u16"] = U16Type
	sa.types["i8"] = I8Type
	sa.types["i16"] = I16Type
	sa.types["d8"] = D8Type
	sa.types["d16"] = D16Type
	sa.types["bool"] = BoolType
}

// ============================================================================
// Pass 1: Declaration Registration
// ============================================================================

func (sa *SemanticAnalyzer) registerDeclaration(node parser.ParserNode) {
	switch n := node.(type) {
	case parser.VariableDeclaration:
		// Only register if it has an explicit type (not inferred)
		if typeRef := n.TypeRef(); typeRef != nil {
			sa.registerVariable(n.Label().Name(), typeRef)
		}
		// Inferred types will be resolved in pass 2
	case parser.FunctionDeclaration:
		sa.registerFunction(n)
	case parser.TypeDeclaration:
		sa.registerType(n)
	default:
		sa.error(fmt.Sprintf("unknown declaration type: %T", node))
	}
}

func (sa *SemanticAnalyzer) registerVariable(name string, typeRef parser.TypeRef) {
	typ := sa.resolveTypeRef(typeRef)
	if typ == nil {
		return // Error already reported
	}

	symbol := &Symbol{
		Name:   name,
		Type:   typ,
		Offset: 0, // Will be computed during layout phase
	}

	if !sa.currentScope.Add(symbol) {
		sa.error(fmt.Sprintf("symbol '%s' already declared in this scope", name))
	}
}

func (sa *SemanticAnalyzer) registerFunction(node parser.FunctionDeclaration) {
	// Parse parameter types
	paramTypes := []Type{}
	if params := node.Parameters(); params != nil {
		for _, field := range params.Fields() {
			typ := sa.resolveTypeRef(field.TypeRef())
			if typ != nil {
				paramTypes = append(paramTypes, typ)
			}
		}
	}

	// Parse return type
	var returnType Type
	if retTypeRef := node.ReturnType(); retTypeRef != nil {
		returnType = sa.resolveTypeRef(retTypeRef)
	}

	funcType := NewFunctionType(paramTypes, returnType)
	symbol := &Symbol{
		Name:   node.Label().Name(),
		Type:   funcType,
		Offset: 0,
	}

	if !sa.currentScope.Add(symbol) {
		sa.error(fmt.Sprintf("function '%s' already declared", node.Label().Name()))
	}
}

func (sa *SemanticAnalyzer) registerType(node parser.TypeDeclaration) {
	name := node.Name().Text()

	// Build struct fields
	fields := []*StructField{}
	if fieldList := node.Fields(); fieldList != nil {
		for _, field := range fieldList.Fields().Fields() {
			fieldType := sa.resolveTypeRef(field.TypeRef())
			if fieldType != nil {
				fields = append(fields, &StructField{
					Name: field.Label().Name(),
					Type: fieldType,
				})
			}
		}
	}

	structType := NewStructType(name, fields)
	sa.types[name] = structType
}

// ============================================================================
// Pass 2: IR Building with Type Checking
// ============================================================================

func (sa *SemanticAnalyzer) processDeclaration(node parser.ParserNode) IRDeclaration {
	switch n := node.(type) {
	case parser.VariableDeclaration:
		return sa.processVarDecl(n)
	case parser.FunctionDeclaration:
		return sa.processFunctionDecl(n)
	case parser.TypeDeclaration:
		return sa.processTypeDecl(n)
	default:
		sa.error(fmt.Sprintf("unknown declaration type: %T", node))
		return nil
	}
}

func (sa *SemanticAnalyzer) processVarDecl(node parser.VariableDeclaration) *IRVariableDecl {
	name := node.Label().Name()
	typeRef := node.TypeRef()
	initExpr := node.Initializer()

	var symbol *Symbol
	var initializer IRExpression

	if typeRef != nil {
		// Explicit type: lookup symbol registered in pass 1
		symbol = sa.currentScope.Lookup(name)
		if symbol == nil {
			sa.error(fmt.Sprintf("internal error: symbol '%s' not found", name))
			return nil
		}

		// Process optional initializer
		if initExpr != nil {
			initializer = sa.processExpression(initExpr)
			if initializer == nil {
				return nil
			}
			// TODO: Check that initializer type matches variable type
		}
	} else {
		// Inferred type: initializer is mandatory
		if initExpr == nil {
			sa.error(fmt.Sprintf("variable '%s' without type must have initializer", name))
			return nil
		}

		// Process initializer to infer type
		initializer = sa.processExpression(initExpr)
		if initializer == nil {
			return nil
		}

		// Create symbol with inferred type
		symbol = &Symbol{
			Name:   name,
			Type:   initializer.Type(),
			Offset: 0,
		}
		if !sa.currentScope.Add(symbol) {
			sa.error(fmt.Sprintf("symbol '%s' already declared in this scope", name))
			return nil
		}
	}

	return &IRVariableDecl{
		Symbol:      symbol,
		Initializer: initializer,
		astNode:     node,
	}
}

func (sa *SemanticAnalyzer) processFunctionDecl(node parser.FunctionDeclaration) *IRFunctionDecl {
	name := node.Label().Name()
	symbol := sa.currentScope.Lookup(name)
	if symbol == nil {
		sa.error(fmt.Sprintf("internal error: function '%s' not found", name))
		return nil
	}

	// Create new scope for function
	funcScope := NewSymbolTable(sa.currentScope)
	sa.pushScope(funcScope)
	defer sa.popScope()

	// Add parameters to function scope
	parameters := []*Symbol{}
	if params := node.Parameters(); params != nil {
		for _, field := range params.Fields() {
			paramType := sa.resolveTypeRef(field.TypeRef())
			paramSymbol := &Symbol{
				Name:   field.Label().Name(),
				Type:   paramType,
				Offset: 0,
			}
			funcScope.Add(paramSymbol)
			parameters = append(parameters, paramSymbol)
		}
	}

	// Process function body
	body := sa.processBlock(node.Body())

	// Get return type
	var returnType Type
	if retTypeRef := node.ReturnType(); retTypeRef != nil {
		returnType = sa.resolveTypeRef(retTypeRef)
	}

	return &IRFunctionDecl{
		Name:       name,
		Parameters: parameters,
		ReturnType: returnType,
		Body:       body,
		Scope:      funcScope,
		astNode:    node,
	}
}

func (sa *SemanticAnalyzer) processTypeDecl(node parser.TypeDeclaration) *IRTypeDecl {
	name := node.Name().Text()
	structType := sa.types[name].(*StructType)

	return &IRTypeDecl{
		Type:    structType,
		astNode: node,
	}
}

// ============================================================================
// Statement Processing
// ============================================================================

func (sa *SemanticAnalyzer) processBlock(node parser.CodeBlock) *IRBlock {
	// Create new scope for block
	blockScope := NewSymbolTable(sa.currentScope)
	sa.pushScope(blockScope)
	defer sa.popScope()

	statements := []IRStatement{}
	for _, stmt := range node.Statements() {
		irStmt := sa.processStatement(stmt)
		if irStmt != nil {
			statements = append(statements, irStmt)
		}
	}

	return &IRBlock{
		Statements: statements,
		Scope:      blockScope,
		astNode:    node,
	}
}

func (sa *SemanticAnalyzer) processStatement(node parser.ParserNode) IRStatement {
	switch n := node.(type) {
	case parser.VariableDeclaration:
		return sa.processVarDecl(n)
	case parser.VariableAssignment:
		return sa.processAssignment(n)
	case parser.StatementIf:
		return sa.processIf(n)
	case parser.StatementFor:
		return sa.processFor(n)
	case parser.StatementSelect:
		return sa.processSelect(n)
	case parser.StatementExpression:
		return sa.processExpressionStmt(n)
	default:
		sa.error(fmt.Sprintf("unknown statement type: %T", node))
		return nil
	}
}

func (sa *SemanticAnalyzer) processAssignment(node parser.VariableAssignment) *IRAssignment {
	name := node.Identifier().Text()
	symbol := sa.currentScope.Lookup(name)
	if symbol == nil {
		sa.error(fmt.Sprintf("undefined variable '%s'", name))
		return nil
	}

	value := sa.processExpression(node.Expression())
	if value == nil {
		return nil
	}

	// TODO: Check type compatibility

	return &IRAssignment{
		Target:  symbol,
		Value:   value,
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processIf(node parser.StatementIf) *IRIf {
	condition := sa.processExpression(node.Condition())
	thenBlock := sa.processBlock(node.ThenBlock())

	// TODO: Process elsif clauses
	elsifBlocks := []*IRElsif{}

	var elseBlock *IRBlock
	if eb := node.ElseBlock(); eb != nil {
		elseBlock = sa.processBlock(eb)
	}

	return &IRIf{
		Condition:   condition,
		ThenBlock:   thenBlock,
		ElsifBlocks: elsifBlocks,
		ElseBlock:   elseBlock,
		astNode:     node,
	}
}

func (sa *SemanticAnalyzer) processFor(node parser.StatementFor) *IRFor {
	// TODO: Implement
	return &IRFor{
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processSelect(node parser.StatementSelect) *IRSelect {
	// TODO: Implement
	return &IRSelect{
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processExpressionStmt(node parser.StatementExpression) *IRExpressionStmt {
	expr := sa.processExpression(node.Expression())
	return &IRExpressionStmt{
		Expression: expr,
		astNode:    node,
	}
}

// ============================================================================
// Expression Processing
// ============================================================================

func (sa *SemanticAnalyzer) processExpression(node parser.Expression) IRExpression {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case parser.ExpressionLiteral:
		return sa.processLiteral(n)
	case parser.ExpressionOperatorBinary:
		return sa.processBinaryOp(n, n.Operator().Id())
	case parser.ExpressionFunctionInvocation:
		return sa.processFunctionCall(n)
	case parser.ExpressionMemberAccess:
		return sa.processMemberAccess(n)
	case parser.ExpressionTypeInitializer:
		return sa.processTypeInitializer(n)
	default:
		// Try to extract identifier from generic Expression
		if expr, ok := node.(interface{ ExpressionKind() parser.ExpressionKind }); ok {
			if expr.ExpressionKind() == parser.ExprIdentifier {
				return sa.processIdentifier(node)
			}
		}
		sa.error(fmt.Sprintf("unknown expression type: %T", node))
		return nil
	}
}

func (sa *SemanticAnalyzer) processLiteral(node parser.ExpressionLiteral) *IRConstant {
	token := node.Value()

	var value interface{}
	var typ Type

	switch token.Id() {
	case lexer.TokenNumber:
		// TODO: Parse number and determine type (u8, u16, etc.)
		value = 0
		typ = U8Type
	case lexer.TokenString:
		value = token.Text()
		// String is u8[] array
		typ = NewArrayType(U8Type, len(token.Text()))
	case lexer.TokenTrue, lexer.TokenFalse:
		value = token.Id() == lexer.TokenTrue
		typ = BoolType
	default:
		sa.error(fmt.Sprintf("unknown literal type: %s", token.Text()))
		return nil
	}

	return &IRConstant{
		Value:   value,
		typ:     typ,
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processIdentifier(node parser.Expression) *IRSymbolRef {
	// TODO: Extract identifier name from expression
	// This is a bit tricky with the current AST structure
	name := "" // Extract from tokens

	symbol := sa.currentScope.Lookup(name)
	if symbol == nil {
		sa.error(fmt.Sprintf("undefined identifier '%s'", name))
		return nil
	}

	return &IRSymbolRef{
		Symbol:  symbol,
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processBinaryOp(node parser.ExpressionOperatorBinary, opToken lexer.TokenId) *IRBinaryOp {
	left := sa.processExpression(node.Left())
	right := sa.processExpression(node.Right())

	if left == nil || right == nil {
		return nil
	}

	// Map token to operator
	op := sa.mapBinaryOperator(opToken)

	// Determine result type
	// TODO: Implement proper type inference/coercion
	resultType := left.Type()

	return &IRBinaryOp{
		Op:      op,
		Left:    left,
		Right:   right,
		typ:     resultType,
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processFunctionCall(node parser.ExpressionFunctionInvocation) *IRFunctionCall {
	name := node.FunctionName().Text()
	symbol := sa.currentScope.Lookup(name)
	if symbol == nil {
		sa.error(fmt.Sprintf("undefined function '%s'", name))
		return nil
	}

	// Process arguments
	args := []IRExpression{}
	if argList := node.Arguments(); argList != nil {
		for _, arg := range argList.Arguments() {
			irArg := sa.processExpression(arg)
			if irArg != nil {
				args = append(args, irArg)
			}
		}
	}

	// TODO: Type check arguments against function signature

	// Get return type from function type
	funcType := symbol.Type.(*FunctionType)
	returnType := funcType.ReturnType()

	return &IRFunctionCall{
		Function:  symbol,
		Arguments: args,
		typ:       returnType,
		astNode:   node,
	}
}

func (sa *SemanticAnalyzer) processMemberAccess(node parser.ExpressionMemberAccess) *IRMemberAccess {
	// TODO: Implement
	return nil
}

func (sa *SemanticAnalyzer) processTypeInitializer(node parser.ExpressionTypeInitializer) *IRTypeInitializer {
	// TODO: Implement
	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

func (sa *SemanticAnalyzer) resolveTypeRef(typeRef parser.TypeRef) Type {
	if typeRef == nil {
		return nil
	}

	typeName := typeRef.TypeName().Text()
	typ := sa.types[typeName]
	if typ == nil {
		sa.error(fmt.Sprintf("undefined type '%s'", typeName))
		return nil
	}

	// Handle array types
	if typeRef.IsArray() {
		length := 0
		if sizeToken := typeRef.ArraySize(); sizeToken != nil {
			// TODO: Parse array size
			length = 0 // Placeholder
		}
		return NewArrayType(typ, length)
	}

	return typ
}

func (sa *SemanticAnalyzer) mapBinaryOperator(token lexer.TokenId) BinaryOperator {
	switch token {
	case lexer.TokenPlus:
		return OpAdd
	case lexer.TokenMinus:
		return OpSubtract
	case lexer.TokenAsterisk:
		return OpMultiply
	case lexer.TokenSlash:
		return OpDivide
	case lexer.TokenAmpersant:
		return OpBitwiseAnd
	case lexer.TokenPipe:
		return OpBitwiseOr
	case lexer.TokenCaret:
		return OpBitwiseXor
	case lexer.TokenEquals:
		return OpEqual
	case lexer.TokenNotEquals:
		return OpNotEqual
	case lexer.TokenLess:
		return OpLessThan
	case lexer.TokenLessOrEquals:
		return OpLessEqual
	case lexer.TokenGreater:
		return OpGreaterThan
	case lexer.TokenGreaterOrEquals:
		return OpGreaterEqual
	case lexer.TokenAnd:
		return OpLogicalAnd
	case lexer.TokenOr:
		return OpLogicalOr
	default:
		sa.error(fmt.Sprintf("unknown binary operator: %d", token))
		return OpAdd // Default
	}
}

func (sa *SemanticAnalyzer) pushScope(scope *SymbolTable) {
	sa.currentScope = scope
}

func (sa *SemanticAnalyzer) popScope() {
	if sa.currentScope.parent != nil {
		sa.currentScope = sa.currentScope.parent
	}
}

func (sa *SemanticAnalyzer) error(msg string) {
	sa.errors = append(sa.errors, fmt.Errorf("%s", msg))
}
