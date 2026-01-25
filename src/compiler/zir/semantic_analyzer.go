package zir

import (
	"fmt"
	"zenith/compiler/lexer"
	"zenith/compiler/parser"
)

// SemanticAnalyzer performs semantic analysis on the AST and builds the IR
type SemanticAnalyzer struct {
	globalScope     *SymbolTable
	currentScope    *SymbolTable
	currentFunction string // Track which function we're analyzing
	callGraph       *CallGraph
	errors          []*IRError
}

// NewSemanticAnalyzer creates a new semantic analyzer
func NewSemanticAnalyzer() *SemanticAnalyzer {
	sa := &SemanticAnalyzer{
		callGraph: NewCallGraph(),
		errors:    make([]*IRError, 0),
	}
	return sa
}

// Analyze performs semantic analysis on the AST and returns the IR
func (sa *SemanticAnalyzer) Analyze(ast parser.CompilationUnit) (*IRCompilationUnit, []*IRError) {
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
		CallGraph:    sa.callGraph,
		astNode:      ast,
	}, sa.errors
}

// ============================================================================
// Built-in Types Initialization
// ============================================================================

func (sa *SemanticAnalyzer) initBuiltinTypes() {
	// Add builtin types as type symbols in global scope
	builtins := map[string]Type{
		"u8":   U8Type,
		"u16":  U16Type,
		"i8":   I8Type,
		"i16":  I16Type,
		"d8":   D8Type,
		"d16":  D16Type,
		"bool": BoolType,
	}
	for name, typ := range builtins {
		sa.globalScope.Add(&Symbol{
			Name: name,
			Kind: SymbolType,
			Type: typ,
		})
	}
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
		sa.error(fmt.Sprintf("unknown declaration type: %T", node), node)
	}
}

func (sa *SemanticAnalyzer) registerVariable(name string, typeRef parser.TypeRef) {
	typ := sa.resolveTypeRef(typeRef)
	if typ == nil {
		return // Error already reported
	}

	symbol := &Symbol{
		Name:   name,
		Kind:   SymbolVariable,
		Type:   typ,
		Offset: 0, // Will be computed during layout phase
	}

	if !sa.currentScope.Add(symbol) {
		sa.error(fmt.Sprintf("symbol '%s' already declared in this scope", name), typeRef)
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
		Kind:   SymbolFunction,
		Type:   funcType,
		Offset: 0,
	}

	if !sa.currentScope.Add(symbol) {
		sa.error(fmt.Sprintf("function '%s' already declared", node.Label().Name()), node)
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

	// Add type as a symbol
	sa.currentScope.Add(&Symbol{
		Name: name,
		Kind: SymbolType,
		Type: structType,
	})
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
		sa.error(fmt.Sprintf("unknown declaration type: %T", node), node)
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
		typeName := typeRef.TypeName().Text()
		symbol = sa.currentScope.Lookup(typeName)
		if symbol == nil {
			sa.error(fmt.Sprintf("symbol '%s' not found", typeName), node)
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

		symbol = &Symbol{
			Name:   name,
			Kind:   SymbolVariable,
			Type:   symbol.Type,
			Offset: 0,
		}

		// globals have been registered already
		if !sa.currentScope.IsGlobal() {
			// Create symbol with inferred type

			if !sa.currentScope.Add(symbol) {
				sa.error(fmt.Sprintf("symbol '%s' already declared in this scope", name), node)
				return nil
			}
		}
	} else {
		// Inferred type: initializer is mandatory
		if initExpr == nil {
			sa.error(fmt.Sprintf("internal error: variable '%s' without type must have initializer", name), node)
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
			Kind:   SymbolVariable,
			Type:   initializer.Type(),
			Offset: 0,
		}
		if !sa.currentScope.Add(symbol) {
			sa.error(fmt.Sprintf("symbol '%s' already declared in this scope", name), node)
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
		sa.error(fmt.Sprintf("internal error: function '%s' not found", name), node)
		return nil
	}

	// Track current function for call graph
	prevFunc := sa.currentFunction
	sa.currentFunction = name
	sa.callGraph.AddFunction(name)
	defer func() { sa.currentFunction = prevFunc }()

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
				Kind:   SymbolVariable,
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
	symbol := sa.currentScope.Lookup(name)
	if symbol == nil || symbol.Kind != SymbolType {
		sa.error(fmt.Sprintf("internal error: type '%s' not found", name), node)
		return nil
	}
	typ := symbol.Type

	structType, ok := typ.(*StructType)
	if !ok {
		sa.error(fmt.Sprintf("internal error: type '%s' is not a struct type", name), node)
		return nil
	}

	return &IRTypeDecl{
		Type:    structType,
		astNode: node,
	}
}

// ============================================================================
// Statement Processing
// ============================================================================

func (sa *SemanticAnalyzer) processBlock(node parser.CodeBlock) *IRBlock {
	// Use current scope (function scope) - no new scope for blocks

	statements := []IRStatement{}
	for _, stmt := range node.Statements() {
		irStmt := sa.processStatement(stmt)
		if irStmt != nil {
			statements = append(statements, irStmt)
		}
	}

	return &IRBlock{
		Statements: statements,
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
	case parser.StatementReturn:
		return sa.processReturn(n)
	default:
		sa.error(fmt.Sprintf("unknown statement type: %T", node), node)
		return nil
	}
}

func (sa *SemanticAnalyzer) processAssignment(node parser.VariableAssignment) *IRAssignment {
	name := node.Identifier().Text()
	symbol := sa.currentScope.Lookup(name)
	if symbol == nil {
		sa.error(fmt.Sprintf("undefined variable '%s'", name), node)
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

	// Process elsif clauses
	elsifBlocks := []*IRElsif{}
	for _, elsifNode := range node.ElsifClauses() {
		elsifCondition := sa.processExpression(elsifNode.Condition())
		elsifThenBlock := sa.processBlock(elsifNode.ThenBlock())
		elsifBlocks = append(elsifBlocks, &IRElsif{
			Condition: elsifCondition,
			ThenBlock: elsifThenBlock,
			astNode:   elsifNode,
		})
	}

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
	// Create a new scope for the for loop
	loopScope := &SymbolTable{
		parent:  sa.currentScope,
		symbols: make(map[string]*Symbol),
	}
	sa.pushScope(loopScope)
	defer sa.popScope()

	var initializer IRStatement
	if init := node.Initializer(); init != nil {
		// Initializer can be a variable declaration or an expression
		initializer = sa.processStatement(init)

		// Mark variable as counter if it's a declaration
		if varDecl, ok := initializer.(*IRVariableDecl); ok {
			sa.updateVariableUsage(varDecl.Symbol, VariableUsageCounter)
		}
	}

	var condition IRExpression
	if cond := node.Condition(); cond != nil {
		condition = sa.processExpression(cond)
		// Variables in condition are likely counters
		sa.trackVariableUsageInExpression(condition, VariableUsageCounter)
	}

	var increment IRExpression
	if inc := node.Increment(); inc != nil {
		increment = sa.processExpression(inc)
		// Variables in increment are counters
		sa.trackVariableUsageInExpression(increment, VariableUsageCounter)
	}

	var body *IRBlock
	if bodyNode := node.Body(); bodyNode != nil {
		body = sa.processBlock(bodyNode)
	}

	return &IRFor{
		Initializer: initializer,
		Condition:   condition,
		Increment:   increment,
		Body:        body,
		astNode:     node,
	}
}

func (sa *SemanticAnalyzer) processSelect(node parser.StatementSelect) *IRSelect {
	// Process the select expression
	expr := sa.processExpression(node.Expression())
	if expr == nil {
		return nil
	}

	// Process cases
	cases := []*IRSelectCase{}
	for _, caseNode := range node.Cases() {
		// Process case value
		caseValue := sa.processExpression(caseNode.Expression())
		if caseValue == nil {
			continue
		}

		// Process case body
		caseBody := sa.processBlock(caseNode.Body())

		cases = append(cases, &IRSelectCase{
			Value:   caseValue,
			Body:    caseBody,
			astNode: caseNode,
		})
	}

	// Process optional else clause
	var elseBody *IRBlock
	if elseNode := node.Else(); elseNode != nil {
		elseBody = sa.processBlock(elseNode.Body())
	}

	return &IRSelect{
		Expression: expr,
		Cases:      cases,
		Else:       elseBody,
		astNode:    node,
	}
}

func (sa *SemanticAnalyzer) processExpressionStmt(node parser.StatementExpression) *IRExpressionStmt {
	expr := sa.processExpression(node.Expression())
	return &IRExpressionStmt{
		Expression: expr,
		astNode:    node,
	}
}

func (sa *SemanticAnalyzer) processReturn(node parser.StatementReturn) *IRReturn {
	var value IRExpression
	if node.Value() != nil {
		value = sa.processExpression(node.Value())
	}

	// TODO: Check that the return value type is compatible with the function's declared return type

	return &IRReturn{
		Value:   value,
		astNode: node,
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
	case parser.ExpressionOperatorUnaryPrefix:
		return sa.processUnaryPrefixOp(n)
	case parser.ExpressionFunctionInvocation:
		return sa.processFunctionCall(n)
	case parser.ExpressionMemberAccess:
		return sa.processMemberAccess(n)
	case parser.ExpressionTypeInitializer:
		return sa.processTypeInitializer(n)
	case parser.ExpressionIdentifier:
		return sa.processIdentifier(n)
	default:
		sa.error(fmt.Sprintf("unknown expression type: %T", node), node)
		return nil
	}
}

func (sa *SemanticAnalyzer) processLiteral(node parser.ExpressionLiteral) *IRConstant {
	token := node.Value()

	var value interface{}
	var typ Type

	switch token.Id() {
	case lexer.TokenNumber:
		value = node.Number()
		// Determine type based on value range
		numVal := node.Number()
		if numVal < 0 {
			if numVal >= -128 {
				typ = I8Type
			} else {
				typ = I16Type
			}
		} else {
			if numVal <= 255 {
				typ = U8Type
			} else {
				typ = U16Type
			}
		}
	case lexer.TokenString:
		value = node.String()
		// String is u8[] array
		typ = NewArrayType(U8Type, len(node.String()))
	case lexer.TokenTrue, lexer.TokenFalse:
		value = token.Id() == lexer.TokenTrue
		typ = BoolType
	default:
		sa.error(fmt.Sprintf("unknown literal type: %s", token.Text()), node)
		return nil
	}

	return &IRConstant{
		Value:   value,
		typ:     typ,
		astNode: node,
	}
}

// processIdentifier handles identifier expressions (variable/parameter references)
func (sa *SemanticAnalyzer) processIdentifier(node parser.ExpressionIdentifier) *IRSymbolRef {
	// Get the identifier token directly from the node
	token := node.Identifier()
	if token == nil {
		sa.error("identifier expression has no identifier token", node)
		return nil
	}

	name := token.Text()
	symbol := sa.currentScope.Lookup(name)
	if symbol == nil {
		sa.error(fmt.Sprintf("undefined identifier '%s'", name), node)
		return nil
	}

	return &IRSymbolRef{
		Symbol:  symbol,
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processUnaryPrefixOp(node parser.ExpressionOperatorUnaryPrefix) IRExpression {
	operand := sa.processExpression(node.Operand())
	if operand == nil {
		return nil
	}

	opToken := node.Operator().Id()

	// Handle unary minus with constant folding for literals
	if opToken == lexer.TokenMinus {
		if constant, ok := operand.(*IRConstant); ok {
			if numVal, ok := constant.Value.(int); ok {
				negatedVal := -numVal
				var typ Type
				if negatedVal >= -128 {
					typ = I8Type
				} else {
					typ = I16Type
				}
				return &IRConstant{
					Value:   negatedVal,
					typ:     typ,
					astNode: node,
				}
			}
		}
	}

	// TODO: Handle other unary operators (unary plus, bitwise not, logical not)
	sa.error(fmt.Sprintf("unary operator %s not yet implemented", node.Operator().Text()), node)
	return nil
}

func (sa *SemanticAnalyzer) processBinaryOp(node parser.ExpressionOperatorBinary, opToken lexer.TokenId) *IRBinaryOp {
	left := sa.processExpression(node.Left())
	right := sa.processExpression(node.Right())

	if left == nil || right == nil {
		return nil
	}

	// Map token to operator
	op := sa.mapBinaryOperator(opToken)

	// Track variable usage for arithmetic operations
	if sa.isArithmeticOperator(op) {
		sa.trackVariableUsageInExpression(left, VariableUsageArithmetic)
		sa.trackVariableUsageInExpression(right, VariableUsageArithmetic)
	}

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
		sa.error(fmt.Sprintf("undefined function '%s'", name), node)
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

	// Record call in call graph
	if sa.currentFunction != "" {
		sa.callGraph.AddCall(sa.currentFunction, name)
	}

	return &IRFunctionCall{
		Function:  symbol,
		Arguments: args,
		typ:       returnType,
		astNode:   node,
	}
}

func (sa *SemanticAnalyzer) processMemberAccess(node parser.ExpressionMemberAccess) *IRMemberAccess {
	// Process the object expression
	object := sa.processExpression(node.Object())
	if object == nil {
		return nil
	}

	// Get the member name
	memberToken := node.Member()
	if memberToken == nil {
		sa.error("member access has no member name", node)
		return nil
	}
	memberName := memberToken.Text()

	// Get the struct type from the object
	structType, ok := object.Type().(*StructType)
	if !ok {
		sa.error(fmt.Sprintf("cannot access member '%s' on non-struct type", memberName), node)
		return nil
	}

	// Find the field in the struct
	var field *StructField
	for _, f := range structType.Fields() {
		if f.Name == memberName {
			field = f
			break
		}
	}

	if field == nil {
		sa.error(fmt.Sprintf("struct '%s' has no field '%s'", structType.Name(), memberName), node)
		return nil
	}

	return &IRMemberAccess{
		Object:  &object,
		Field:   field,
		typ:     field.Type,
		astNode: node,
	}
}

func (sa *SemanticAnalyzer) processTypeInitializer(node parser.ExpressionTypeInitializer) *IRTypeInitializer {
	// Get the type reference
	typeRef := node.TypeRef()
	if typeRef == nil {
		sa.error("type initializer has no type reference", node)
		return nil
	}

	// Resolve the type
	typ := sa.resolveTypeRef(typeRef)
	if typ == nil {
		return nil
	}

	// Ensure it's a struct type
	structType, ok := typ.(*StructType)
	if !ok {
		sa.error(fmt.Sprintf("cannot initialize non-struct type '%s'", typeRef.TypeName().Text()), node)
		return nil
	}

	// Process field initializers
	fieldInits := []*IRFieldInit{}
	if initializer := node.Initializer(); initializer != nil {
		if fieldList := initializer.Fields(); fieldList != nil {
			for _, fieldNode := range fieldList.Fields() {
				fieldName := fieldNode.Identifier().Text()

				// Find the field in the struct
				var structField *StructField
				for _, f := range structType.Fields() {
					if f.Name == fieldName {
						structField = f
						break
					}
				}

				if structField == nil {
					sa.error(fmt.Sprintf("struct '%s' has no field '%s'", structType.Name(), fieldName), fieldNode)
					continue
				}

				// Process the field value expression
				valueExpr := sa.processExpression(fieldNode.Expression())
				if valueExpr == nil {
					continue
				}

				// TODO: Type check that valueExpr type matches structField type

				fieldInits = append(fieldInits, &IRFieldInit{
					Field: structField,
					Value: valueExpr,
				})
			}
		}
	}

	return &IRTypeInitializer{
		StructType: structType,
		Fields:     fieldInits,
		typ:        structType,
		astNode:    node,
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

func (sa *SemanticAnalyzer) resolveTypeRef(typeRef parser.TypeRef) Type {
	if typeRef == nil {
		return nil
	}

	typeName := typeRef.TypeName().Text()
	symbol := sa.currentScope.Lookup(typeName)
	if symbol == nil || symbol.Kind != SymbolType {
		sa.error(fmt.Sprintf("undefined type '%s'", typeName), typeRef)
		return nil
	}
	typ := symbol.Type

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
		// Note: We can't pass a node here because we don't have access to it
		sa.error(fmt.Sprintf("unknown binary operator: %d", token), nil)
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

// updateVariableUsage updates the usage pattern for a variable symbol
// Only updates if the current usage is more specific than the existing one
func (sa *SemanticAnalyzer) updateVariableUsage(symbol *Symbol, usage VariableUsage) {
	if symbol == nil || symbol.Kind != SymbolVariable {
		return
	}

	// Only update if current usage is General (unspecified)
	// Once a specific usage is set, keep it
	if symbol.Usage == VariableUsageGeneral {
		symbol.Usage = usage
	}
}

// trackVariableUsageInExpression recursively tracks how variables are used in expressions
func (sa *SemanticAnalyzer) trackVariableUsageInExpression(expr IRExpression, usage VariableUsage) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *IRSymbolRef:
		sa.updateVariableUsage(e.Symbol, usage)
	case *IRBinaryOp:
		sa.trackVariableUsageInExpression(e.Left, usage)
		sa.trackVariableUsageInExpression(e.Right, usage)
	case *IRUnaryOp:
		sa.trackVariableUsageInExpression(e.Operand, usage)
	case *IRFunctionCall:
		for _, arg := range e.Arguments {
			sa.trackVariableUsageInExpression(arg, usage)
		}
	case *IRMemberAccess:
		if e.Object != nil {
			sa.trackVariableUsageInExpression(*e.Object, VariableUsagePointer)
		}
	}
}

// isArithmeticOperator checks if an operator is arithmetic
func (sa *SemanticAnalyzer) isArithmeticOperator(op BinaryOperator) bool {
	switch op {
	case OpAdd, OpSubtract, OpMultiply, OpDivide,
		OpBitwiseAnd, OpBitwiseOr, OpBitwiseXor:
		return true
	default:
		return false
	}
}

func (sa *SemanticAnalyzer) error(msg string, node parser.ParserNode) {
	sa.errors = append(sa.errors, NewIRError(msg, node))
}
