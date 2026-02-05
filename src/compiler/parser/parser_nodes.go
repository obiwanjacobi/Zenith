package parser

import (
	"reflect"
	"strconv"
	"zenith/compiler"
	"zenith/compiler/lexer"
)

type ParserNode interface {
	Children() []ParserNode
	Tokens() []lexer.Token
	Errors() []ParserError
}

// Base parser node data structure
type parserNodeData struct {
	_children []ParserNode
	_tokens   []lexer.Token
	_errors   []ParserError
}

func (n *parserNodeData) Children() []ParserNode {
	return n._children
}

func (n *parserNodeData) Tokens() []lexer.Token {
	return n._tokens
}

func (n *parserNodeData) Errors() []ParserError {
	return n._errors
}

func (n *parserNodeData) tokensOf(tokenId lexer.TokenId) []lexer.Token {
	result := make([]lexer.Token, 0)
	for i := 0; i < len(n._tokens); i++ {
		if n._tokens[i].Id() == tokenId {
			result = append(result, n._tokens[i])
		}
	}
	return result
}
func (n *parserNodeData) childrenOf(t reflect.Type) []interface{} {
	result := make([]interface{}, 0)
	for i := 0; i < len(n._children); i++ {
		child := n._children[i]
		if reflect.TypeOf(child).Implements(t) {
			result = append(result, child)
		}
	}
	return result
}

// ============================================================================
// compilationUnit: (variable_declaration | function_declaration | type_declaration)*
// ============================================================================

type CompilationUnit interface {
	ParserNode
	Declarations() []ParserNode
}

type compilationUnit struct {
	parserNodeData
}

func (n *compilationUnit) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *compilationUnit) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *compilationUnit) Declarations() []ParserNode {
	return n.parserNodeData._children
}

// ============================================================================
// code_block: (statement | expression_statement | function_invocation | variable_declaration | variable_assignment)*
// ============================================================================

type CodeBlock interface {
	ParserNode
	Statements() []ParserNode
}

type codeBlock struct {
	parserNodeData
}

func (n *codeBlock) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *codeBlock) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *codeBlock) Statements() []ParserNode {
	return n.parserNodeData._children
}

// ============================================================================
// variable_declaration: label type_ref? ('=' expression)?
// ============================================================================

type VariableDeclaration interface {
	ParserNode
	Label() Label
	TypeRef() TypeRef
	Initializer() Expression
}

type variableDeclaration struct {
	parserNodeData
}

func (n *variableDeclaration) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *variableDeclaration) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *variableDeclaration) Label() Label {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[Label]())
	if len(children) > 0 {
		return children[0].(Label)
	}
	return nil
}

func (n *variableDeclaration) TypeRef() TypeRef {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[TypeRef]())
	if len(children) > 0 {
		return children[0].(TypeRef)
	}
	return nil
}

func (n *variableDeclaration) Initializer() Expression {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[Expression]())
	if len(children) > 0 {
		return children[0].(Expression)
	}
	return nil
}

// ============================================================================
// variable_assignment: identifier (operator_arithmetic | operator_bitwise)? '=' expression
// ============================================================================

type VariableAssignment interface {
	ParserNode
	Identifier() lexer.Token
	Operator() lexer.Token
	Expression() Expression
}

type variableAssignment struct {
	parserNodeData
}

func (n *variableAssignment) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *variableAssignment) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *variableAssignment) Identifier() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

func (n *variableAssignment) Operator() lexer.Token {
	// Return compound operator token if present
	for _, token := range n.parserNodeData._tokens {
		switch token.Id() {
		case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenAsterisk, lexer.TokenSlash,
			lexer.TokenAmpersant, lexer.TokenPipe, lexer.TokenCaret:
			return token
		}
	}
	return nil
}

func (n *variableAssignment) Expression() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

// ============================================================================
// function_declaration: label '(' declaration_fieldlist? ')' type_ref? '{' code_block '}'
// ============================================================================

type FunctionDeclaration interface {
	ParserNode
	Label() Label
	Parameters() DeclarationFieldList
	ReturnType() TypeRef
	Body() CodeBlock
}

type functionDeclaration struct {
	parserNodeData
}

func (n *functionDeclaration) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *functionDeclaration) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *functionDeclaration) Label() Label {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[Label]())
	if len(children) > 0 {
		return children[0].(Label)
	}
	return nil
}

func (n *functionDeclaration) Parameters() DeclarationFieldList {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[DeclarationFieldList]())
	if len(children) > 0 {
		return children[0].(DeclarationFieldList)
	}
	return nil
}

func (n *functionDeclaration) ReturnType() TypeRef {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[TypeRef]())
	if len(children) > 0 {
		return children[0].(TypeRef)
	}
	return nil
}

func (n *functionDeclaration) Body() CodeBlock {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[CodeBlock]())
	if len(children) > 0 {
		return children[0].(CodeBlock)
	}
	return nil
}

// ============================================================================
// function_argumentList: (expression (',' expression)*)?
// ============================================================================

type FunctionArgumentList interface {
	ParserNode
	Arguments() []Expression
}

type functionArgumentList struct {
	parserNodeData
}

func (n *functionArgumentList) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *functionArgumentList) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *functionArgumentList) Arguments() []Expression {
	return compiler.OfType[Expression](n.parserNodeData._children)
}

// ============================================================================
// type_declaration: 'struct' identifier type_declaration_fields
// ============================================================================

type TypeDeclaration interface {
	ParserNode
	Name() lexer.Token
	Fields() TypeDeclarationFields
}

type typeDeclaration struct {
	parserNodeData
}

func (n *typeDeclaration) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *typeDeclaration) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *typeDeclaration) Name() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

func (n *typeDeclaration) Fields() TypeDeclarationFields {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(TypeDeclarationFields)
	}
	return nil
}

// ============================================================================
// type_declaration_fields: '{' declaration_fieldlist '}'
// ============================================================================

type TypeDeclarationFields interface {
	ParserNode
	Fields() DeclarationFieldList
}

type typeDeclarationFields struct {
	parserNodeData
}

func (n *typeDeclarationFields) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *typeDeclarationFields) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *typeDeclarationFields) Fields() DeclarationFieldList {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(DeclarationFieldList)
	}
	return nil
}

// ============================================================================
// type_ref: identifier ('[' number? ']')?
// ============================================================================

type TypeRef interface {
	ParserNode
	TypeName() lexer.Token
	ArraySize() lexer.Token
	IsArray() bool
}

type typeRef struct {
	parserNodeData
}

func (n *typeRef) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *typeRef) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *typeRef) TypeName() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

func (n *typeRef) ArraySize() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenNumber)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

func (n *typeRef) IsArray() bool {
	tokens := n.parserNodeData.tokensOf(lexer.TokenBracketOpen)
	return len(tokens) > 0
}

// ============================================================================
// type_initializer: '{' type_initializer_fieldlist? '}'
// ============================================================================

type TypeInitializer interface {
	ParserNode
	Fields() TypeInitializerFieldList
}

type typeInitializer struct {
	parserNodeData
}

func (n *typeInitializer) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *typeInitializer) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *typeInitializer) Fields() TypeInitializerFieldList {
	if len(n.parserNodeData._children) > 0 {
		if fields, ok := n.parserNodeData._children[0].(TypeInitializerFieldList); ok {
			return fields
		}
	}
	return nil
}

// ============================================================================
// type_initializer_fieldlist: type_initializer_field (',' type_initializer_field)*
// ============================================================================

type TypeInitializerFieldList interface {
	ParserNode
	Fields() []TypeInitializerField
}

type typeInitializerFieldList struct {
	parserNodeData
}

func (n *typeInitializerFieldList) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *typeInitializerFieldList) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *typeInitializerFieldList) Fields() []TypeInitializerField {
	return compiler.OfType[TypeInitializerField](n.parserNodeData._children)
}

// ============================================================================
// type_initializer_field: identifier '=' expression
// ============================================================================

type TypeInitializerField interface {
	ParserNode
	Identifier() lexer.Token
	Expression() Expression
}

type typeInitializerField struct {
	parserNodeData
}

func (n *typeInitializerField) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *typeInitializerField) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *typeInitializerField) Identifier() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

func (n *typeInitializerField) Expression() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

// ============================================================================
// type_alias: 'type' identifier '=' type_ref
// ============================================================================

type TypeAlias interface {
	ParserNode
	Name() lexer.Token
	AliasedType() TypeRef
}

type typeAlias struct {
	parserNodeData
}

func (n *typeAlias) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *typeAlias) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *typeAlias) Name() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

func (n *typeAlias) AliasedType() TypeRef {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(TypeRef)
	}
	return nil
}

// ============================================================================
// declaration_fieldlist: declaration_field (',' declaration_field)*
// ============================================================================

type DeclarationFieldList interface {
	ParserNode
	Fields() []DeclarationField
}

type declarationFieldList struct {
	parserNodeData
}

func (n *declarationFieldList) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *declarationFieldList) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *declarationFieldList) Fields() []DeclarationField {
	return compiler.OfType[DeclarationField](n.parserNodeData._children)
}

// ============================================================================
// declaration_field: label type_ref
// ============================================================================

type DeclarationField interface {
	ParserNode
	Label() Label
	TypeRef() TypeRef
}

type declarationField struct {
	parserNodeData
}

func (n *declarationField) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *declarationField) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *declarationField) Label() Label {
	return n.parserNodeData._children[0].(Label)
}

func (n *declarationField) TypeRef() TypeRef {
	return n.parserNodeData._children[1].(TypeRef)
}

// ============================================================================
// statement: statement_if | statement_for | statement_select | statement_expression
// ============================================================================

type Statement interface {
	ParserNode
}

type statement struct {
	parserNodeData
}

func (n *statement) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statement) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

// ============================================================================
// statement_if: 'if' expression '{' code_block '}' ('elsif' expression '{' code_block '}')* ('else' '{' code_block '}')?
// ============================================================================

type StatementIf interface {
	ParserNode
	Condition() Expression
	ThenBlock() CodeBlock
	ElsifClauses() []StatementElsif
	ElseBlock() CodeBlock
}

type statementIf struct {
	parserNodeData
}

func (n *statementIf) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementIf) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementIf) Condition() Expression {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[Expression]())
	if len(children) > 0 {
		return children[0].(Expression)
	}
	return nil
}

func (n *statementIf) ThenBlock() CodeBlock {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[CodeBlock]())
	if len(children) > 0 {
		return children[0].(CodeBlock)
	}
	return nil
}

func (n *statementIf) ElsifClauses() []StatementElsif {
	return compiler.OfType[StatementElsif](n.parserNodeData._children)
}

func (n *statementIf) ElseBlock() CodeBlock {
	// The else block is distinct from the main then block
	blocks := compiler.OfType[CodeBlock](n.parserNodeData._children)
	if len(blocks) > 1 {
		return blocks[len(blocks)-1]
	}
	return nil
}

// ============================================================================
// elsif clause helper (part of statement_if)
// ============================================================================

type StatementElsif interface {
	ParserNode
	Condition() Expression
	ThenBlock() CodeBlock
}

type statementElsif struct {
	parserNodeData
}

func (n *statementElsif) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementElsif) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementElsif) Condition() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

func (n *statementElsif) ThenBlock() CodeBlock {
	if len(n.parserNodeData._children) > 1 {
		return n.parserNodeData._children[1].(CodeBlock)
	}
	return nil
}

// ============================================================================
// statement_for: 'for' (statement_for_init ';')? expression (';' expression)? '{' code_block '}'
// ============================================================================

type StatementFor interface {
	ParserNode
	Initializer() ParserNode
	Condition() Expression
	Increment() Expression
	Body() CodeBlock
}

type statementFor struct {
	parserNodeData
}

func (n *statementFor) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementFor) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementFor) Initializer() ParserNode {
	// First child if it's not an Expression
	if len(n.parserNodeData._children) > 0 {
		child := n.parserNodeData._children[0]
		exprChildren := n.parserNodeData.childrenOf(reflect.TypeFor[Expression]())
		// Check if first child is an expression
		if len(exprChildren) > 0 && exprChildren[0] == child {
			return nil
		}
		return child
	}
	return nil
}

func (n *statementFor) Condition() Expression {
	expressions := compiler.OfType[Expression](n.parserNodeData._children)
	if len(expressions) > 0 {
		return expressions[0]
	}
	return nil
}

func (n *statementFor) Increment() Expression {
	expressions := compiler.OfType[Expression](n.parserNodeData._children)
	if len(expressions) > 1 {
		return expressions[1]
	}
	return nil
}

func (n *statementFor) Body() CodeBlock {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[CodeBlock]())
	if len(children) > 0 {
		return children[0].(CodeBlock)
	}
	return nil
}

// ============================================================================
// statement_select: 'select' expression '{' statement_select_cases statement_select_else? '}'
// ============================================================================

type StatementSelect interface {
	ParserNode
	Expression() Expression
	Cases() []StatementSelectCase
	Else() StatementSelectElse
}

type statementSelect struct {
	parserNodeData
}

func (n *statementSelect) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementSelect) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementSelect) Expression() Expression {
	children := n.parserNodeData.childrenOf(reflect.TypeFor[Expression]())
	if len(children) > 0 {
		return children[0].(Expression)
	}
	return nil
}

func (n *statementSelect) Cases() []StatementSelectCase {
	return compiler.OfType[StatementSelectCase](n.parserNodeData._children)
}

func (n *statementSelect) Else() StatementSelectElse {
	// Use concrete type to avoid matching statementSelectCase
	elseNodes := compiler.OfType[*statementSelectElse](n.parserNodeData._children)
	if len(elseNodes) > 0 {
		return elseNodes[0]
	}
	return nil
}

// ============================================================================
// statement_select_cases: 'case' expression '{' code_block '}'
// ============================================================================

type StatementSelectCase interface {
	ParserNode
	Expression() Expression
	Body() CodeBlock
}

type statementSelectCase struct {
	parserNodeData
}

func (n *statementSelectCase) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementSelectCase) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementSelectCase) Expression() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

func (n *statementSelectCase) Body() CodeBlock {
	if len(n.parserNodeData._children) > 1 {
		return n.parserNodeData._children[1].(CodeBlock)
	}
	return nil
}

// ============================================================================
// statement_select_else: 'else' '{' code_block '}'
// ============================================================================

type StatementSelectElse interface {
	ParserNode
	Body() CodeBlock
}

type statementSelectElse struct {
	parserNodeData
}

func (n *statementSelectElse) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementSelectElse) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementSelectElse) Body() CodeBlock {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(CodeBlock)
	}
	return nil
}

// ============================================================================
// statement_expression: expression_function_invocation
// ============================================================================

type StatementExpression interface {
	ParserNode
	Expression() Expression
}

type statementExpression struct {
	parserNodeData
}

func (n *statementExpression) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementExpression) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementExpression) Expression() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

// ============================================================================
// statement_return
// ============================================================================

type StatementReturn interface {
	ParserNode
	Value() Expression
}

type statementReturn struct {
	parserNodeData
}

func (n *statementReturn) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *statementReturn) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *statementReturn) Value() Expression {
	if len(n.parserNodeData._children) > 0 && n.parserNodeData._children[0] != nil {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

// ============================================================================
// expression (base interface for all expression types)
// ============================================================================

type ExpressionKind int

const (
	ExprPrecedence ExpressionKind = iota
	ExprMemberAccess
	ExprSubscript
	ExprBinaryArithmetic
	ExprBinaryBitwise
	ExprBinaryComparison
	ExprBinaryLogical
	ExprUnaryPrefixArithmetic
	ExprUnaryPrefixBitwise
	ExprUnaryPrefixLogical
	ExprUnaryPostfixArithmetic
	ExprUnaryPostfixLogical
	ExprFunctionInvocation
	ExprTypeInitializer
	ExprLiteral
	ExprIdentifier
)

type Expression interface {
	ParserNode
	ExpressionKind() ExpressionKind
}

type expression struct {
	parserNodeData
}

func (n *expression) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expression) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expression) ExpressionKind() ExpressionKind {
	return ExprIdentifier // default/fallback
}

// ============================================================================
// expression_precedence: '(' expression ')'
// ============================================================================

type ExpressionPrecedence interface {
	Expression
	Inner() Expression
}

type expressionPrecedence struct {
	parserNodeData
}

func (n *expressionPrecedence) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionPrecedence) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionPrecedence) ExpressionKind() ExpressionKind {
	return ExprPrecedence
}

func (n *expressionPrecedence) Inner() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

// ============================================================================
// expression_member_access: expression '.' identifier
// ============================================================================

type ExpressionMemberAccess interface {
	Expression
	Object() Expression
	Member() lexer.Token
}

type expressionMemberAccess struct {
	parserNodeData
}

func (n *expressionMemberAccess) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionMemberAccess) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionMemberAccess) ExpressionKind() ExpressionKind {
	return ExprMemberAccess
}

func (n *expressionMemberAccess) Object() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

func (n *expressionMemberAccess) Member() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

// ============================================================================
// expression_subscript: expression '[' expression ']'
// ============================================================================

type ExpressionSubscript interface {
	Expression
	Array() Expression
	Index() Expression
}

type expressionSubscript struct {
	parserNodeData
}

func (n *expressionSubscript) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionSubscript) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionSubscript) ExpressionKind() ExpressionKind {
	return ExprSubscript
}

func (n *expressionSubscript) Array() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

func (n *expressionSubscript) Index() Expression {
	if len(n.parserNodeData._children) > 1 {
		return n.parserNodeData._children[1].(Expression)
	}
	return nil
}

// ============================================================================
// expression_operator_binary (base for all binary operators)
// ============================================================================

type ExpressionOperatorBinary interface {
	Expression
	Left() Expression
	Right() Expression
	Operator() lexer.Token
}

type expressionOperatorBinary struct {
	parserNodeData
}

func (n *expressionOperatorBinary) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionOperatorBinary) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionOperatorBinary) Left() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

func (n *expressionOperatorBinary) Right() Expression {
	if len(n.parserNodeData._children) > 1 {
		return n.parserNodeData._children[1].(Expression)
	}
	return nil
}

func (n *expressionOperatorBinary) Operator() lexer.Token {
	if len(n.parserNodeData._tokens) > 0 {
		return n.parserNodeData._tokens[0]
	}
	return nil
}

// ============================================================================
// expression_operator_bin_arithmetic: expression operator_arithmetic expression
// ============================================================================

type ExpressionOperatorBinArithmetic interface {
	ExpressionOperatorBinary
}

type expressionOperatorBinArithmetic struct {
	expressionOperatorBinary
}

func (n *expressionOperatorBinArithmetic) Children() []ParserNode {
	return n.expressionOperatorBinary.Children()
}

func (n *expressionOperatorBinArithmetic) Tokens() []lexer.Token {
	return n.expressionOperatorBinary.Tokens()
}

func (n *expressionOperatorBinArithmetic) Left() Expression {
	return n.expressionOperatorBinary.Left()
}

func (n *expressionOperatorBinArithmetic) Right() Expression {
	return n.expressionOperatorBinary.Right()
}

func (n *expressionOperatorBinArithmetic) Operator() lexer.Token {
	return n.expressionOperatorBinary.Operator()
}

func (n *expressionOperatorBinArithmetic) ExpressionKind() ExpressionKind {
	return ExprBinaryArithmetic
}

// ============================================================================
// expression_operator_bin_bitwise: expression operator_bitwise expression
// ============================================================================

type ExpressionOperatorBinBitwise interface {
	ExpressionOperatorBinary
}

type expressionOperatorBinBitwise struct {
	expressionOperatorBinary
}

func (n *expressionOperatorBinBitwise) Children() []ParserNode {
	return n.expressionOperatorBinary.Children()
}

func (n *expressionOperatorBinBitwise) Tokens() []lexer.Token {
	return n.expressionOperatorBinary.Tokens()
}

func (n *expressionOperatorBinBitwise) Left() Expression {
	return n.expressionOperatorBinary.Left()
}

func (n *expressionOperatorBinBitwise) Right() Expression {
	return n.expressionOperatorBinary.Right()
}

func (n *expressionOperatorBinBitwise) Operator() lexer.Token {
	return n.expressionOperatorBinary.Operator()
}

func (n *expressionOperatorBinBitwise) ExpressionKind() ExpressionKind {
	return ExprBinaryBitwise
}

// ============================================================================
// expression_operator_bin_comparison: expression ('=' | '>' | '<' | '>=' | '<=' | '<>') expression
// ============================================================================

type ExpressionOperatorBinComparison interface {
	ExpressionOperatorBinary
}

type expressionOperatorBinComparison struct {
	expressionOperatorBinary
}

func (n *expressionOperatorBinComparison) Children() []ParserNode {
	return n.expressionOperatorBinary.Children()
}

func (n *expressionOperatorBinComparison) Tokens() []lexer.Token {
	return n.expressionOperatorBinary.Tokens()
}

func (n *expressionOperatorBinComparison) Left() Expression {
	return n.expressionOperatorBinary.Left()
}

func (n *expressionOperatorBinComparison) Right() Expression {
	return n.expressionOperatorBinary.Right()
}

func (n *expressionOperatorBinComparison) Operator() lexer.Token {
	return n.expressionOperatorBinary.Operator()
}

func (n *expressionOperatorBinComparison) ExpressionKind() ExpressionKind {
	return ExprBinaryComparison
}

// ============================================================================
// expression_operator_bin_logical: expression ('and' | 'or') expression
// ============================================================================

type ExpressionOperatorBinLogical interface {
	ExpressionOperatorBinary
}

type expressionOperatorBinLogical struct {
	expressionOperatorBinary
}

func (n *expressionOperatorBinLogical) Children() []ParserNode {
	return n.expressionOperatorBinary.Children()
}

func (n *expressionOperatorBinLogical) Tokens() []lexer.Token {
	return n.expressionOperatorBinary.Tokens()
}

func (n *expressionOperatorBinLogical) Left() Expression {
	return n.expressionOperatorBinary.Left()
}

func (n *expressionOperatorBinLogical) Right() Expression {
	return n.expressionOperatorBinary.Right()
}

func (n *expressionOperatorBinLogical) Operator() lexer.Token {
	return n.expressionOperatorBinary.Operator()
}

func (n *expressionOperatorBinLogical) ExpressionKind() ExpressionKind {
	return ExprBinaryLogical
}

// ============================================================================
// expression_operator_unaryprefix (base for unary prefix operators)
// ============================================================================

type ExpressionOperatorUnaryPrefix interface {
	Expression
	Operand() Expression
	Operator() lexer.Token
}

type expressionOperatorUnaryPrefix struct {
	parserNodeData
}

func (n *expressionOperatorUnaryPrefix) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionOperatorUnaryPrefix) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionOperatorUnaryPrefix) Operand() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

func (n *expressionOperatorUnaryPrefix) Operator() lexer.Token {
	if len(n.parserNodeData._tokens) > 0 {
		return n.parserNodeData._tokens[0]
	}
	return nil
}

// ============================================================================
// expression_operator_unipre_arithmetic: ('-' | '+') expression
// ============================================================================

type ExpressionOperatorUnipreArithmetic interface {
	ExpressionOperatorUnaryPrefix
}

type expressionOperatorUnipreArithmetic struct {
	expressionOperatorUnaryPrefix
}

func (n *expressionOperatorUnipreArithmetic) Children() []ParserNode {
	return n.expressionOperatorUnaryPrefix.Children()
}

func (n *expressionOperatorUnipreArithmetic) Tokens() []lexer.Token {
	return n.expressionOperatorUnaryPrefix.Tokens()
}

func (n *expressionOperatorUnipreArithmetic) Operand() Expression {
	return n.expressionOperatorUnaryPrefix.Operand()
}

func (n *expressionOperatorUnipreArithmetic) Operator() lexer.Token {
	return n.expressionOperatorUnaryPrefix.Operator()
}

func (n *expressionOperatorUnipreArithmetic) ExpressionKind() ExpressionKind {
	return ExprUnaryPrefixArithmetic
}

// ============================================================================
// expression_operator_unipre_bitwise: '~' expression
// ============================================================================

type ExpressionOperatorUnipreBitwise interface {
	ExpressionOperatorUnaryPrefix
}

type expressionOperatorUnipreBitwise struct {
	expressionOperatorUnaryPrefix
}

func (n *expressionOperatorUnipreBitwise) Children() []ParserNode {
	return n.expressionOperatorUnaryPrefix.Children()
}

func (n *expressionOperatorUnipreBitwise) Tokens() []lexer.Token {
	return n.expressionOperatorUnaryPrefix.Tokens()
}

func (n *expressionOperatorUnipreBitwise) Operand() Expression {
	return n.expressionOperatorUnaryPrefix.Operand()
}

func (n *expressionOperatorUnipreBitwise) Operator() lexer.Token {
	return n.expressionOperatorUnaryPrefix.Operator()
}

func (n *expressionOperatorUnipreBitwise) ExpressionKind() ExpressionKind {
	return ExprUnaryPrefixBitwise
}

// ============================================================================
// expression_operator_unipre_logical: 'not' expression
// ============================================================================

type ExpressionOperatorUnipreLogical interface {
	ExpressionOperatorUnaryPrefix
}

type expressionOperatorUnipreLogical struct {
	expressionOperatorUnaryPrefix
}

func (n *expressionOperatorUnipreLogical) Children() []ParserNode {
	return n.expressionOperatorUnaryPrefix.Children()
}

func (n *expressionOperatorUnipreLogical) Tokens() []lexer.Token {
	return n.expressionOperatorUnaryPrefix.Tokens()
}

func (n *expressionOperatorUnipreLogical) Operand() Expression {
	return n.expressionOperatorUnaryPrefix.Operand()
}

func (n *expressionOperatorUnipreLogical) Operator() lexer.Token {
	return n.expressionOperatorUnaryPrefix.Operator()
}

func (n *expressionOperatorUnipreLogical) ExpressionKind() ExpressionKind {
	return ExprUnaryPrefixLogical
}

// ============================================================================
// expression_operator_unarypostfix (base for unary postfix operators)
// ============================================================================

type ExpressionOperatorUnaryPostfix interface {
	Expression
	Operand() Expression
	Operator() lexer.Token
}

type expressionOperatorUnaryPostfix struct {
	parserNodeData
}

func (n *expressionOperatorUnaryPostfix) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionOperatorUnaryPostfix) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionOperatorUnaryPostfix) Operand() Expression {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(Expression)
	}
	return nil
}

func (n *expressionOperatorUnaryPostfix) Operator() lexer.Token {
	if len(n.parserNodeData._tokens) > 0 {
		return n.parserNodeData._tokens[0]
	}
	return nil
}

// ============================================================================
// expression_operator_unipost_arithmetic: expression ('++' | '--')
// ============================================================================

type ExpressionOperatorUnipostArithmetic interface {
	ExpressionOperatorUnaryPostfix
}

type expressionOperatorUnipostArithmetic struct {
	expressionOperatorUnaryPostfix
}

func (n *expressionOperatorUnipostArithmetic) Children() []ParserNode {
	return n.expressionOperatorUnaryPostfix.Children()
}

func (n *expressionOperatorUnipostArithmetic) Tokens() []lexer.Token {
	return n.expressionOperatorUnaryPostfix.Tokens()
}

func (n *expressionOperatorUnipostArithmetic) Operand() Expression {
	return n.expressionOperatorUnaryPostfix.Operand()
}

func (n *expressionOperatorUnipostArithmetic) Operator() lexer.Token {
	return n.expressionOperatorUnaryPostfix.Operator()
}

func (n *expressionOperatorUnipostArithmetic) ExpressionKind() ExpressionKind {
	return ExprUnaryPostfixArithmetic
}

// ============================================================================
// expression_operator_unipost_logical: expression '?'
// ============================================================================

type ExpressionOperatorUnipostLogical interface {
	ExpressionOperatorUnaryPostfix
}

type expressionOperatorUnipostLogical struct {
	expressionOperatorUnaryPostfix
}

func (n *expressionOperatorUnipostLogical) Children() []ParserNode {
	return n.expressionOperatorUnaryPostfix.Children()
}

func (n *expressionOperatorUnipostLogical) Tokens() []lexer.Token {
	return n.expressionOperatorUnaryPostfix.Tokens()
}

func (n *expressionOperatorUnipostLogical) Operand() Expression {
	return n.expressionOperatorUnaryPostfix.Operand()
}

func (n *expressionOperatorUnipostLogical) Operator() lexer.Token {
	return n.expressionOperatorUnaryPostfix.Operator()
}

func (n *expressionOperatorUnipostLogical) ExpressionKind() ExpressionKind {
	return ExprUnaryPostfixLogical
}

// ============================================================================
// expression_function_invocation: identifier '(' function_argumentList? ')'
// ============================================================================

type ExpressionFunctionInvocation interface {
	Expression
	FunctionName() lexer.Token
	Arguments() FunctionArgumentList
}

type expressionFunctionInvocation struct {
	parserNodeData
}

func (n *expressionFunctionInvocation) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionFunctionInvocation) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionFunctionInvocation) ExpressionKind() ExpressionKind {
	return ExprFunctionInvocation
}

func (n *expressionFunctionInvocation) FunctionName() lexer.Token {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return nil
}

func (n *expressionFunctionInvocation) Arguments() FunctionArgumentList {
	if len(n.parserNodeData._children) > 0 {
		if args, ok := n.parserNodeData._children[0].(FunctionArgumentList); ok {
			return args
		}
	}
	return nil
}

// ============================================================================
// expression_type_initializer: type_ref type_initializer
// ============================================================================

type ExpressionTypeInitializer interface {
	Expression
	TypeRef() TypeRef
	Initializer() TypeInitializer
}

type expressionTypeInitializer struct {
	parserNodeData
}

func (n *expressionTypeInitializer) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionTypeInitializer) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionTypeInitializer) ExpressionKind() ExpressionKind {
	return ExprTypeInitializer
}

func (n *expressionTypeInitializer) TypeRef() TypeRef {
	if len(n.parserNodeData._children) > 0 {
		return n.parserNodeData._children[0].(TypeRef)
	}
	return nil
}

func (n *expressionTypeInitializer) Initializer() TypeInitializer {
	if len(n.parserNodeData._children) > 1 {
		return n.parserNodeData._children[1].(TypeInitializer)
	}
	return nil
}

// ============================================================================
// expression_literal: string | number | bool_literal
// ============================================================================

type ExpressionLiteral interface {
	Expression
	Value() lexer.Token
	Number() int
	String() string
}

type expressionLiteral struct {
	parserNodeData
}

func (n *expressionLiteral) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionLiteral) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionLiteral) ExpressionKind() ExpressionKind {
	return ExprLiteral
}

func (n *expressionLiteral) Value() lexer.Token {
	if len(n.parserNodeData._tokens) > 0 {
		return n.parserNodeData._tokens[0]
	}
	return nil
}

func (n *expressionLiteral) Number() int {
	if token := n.Value(); token != nil && token.Id() == lexer.TokenNumber {
		if num, err := strconv.ParseInt(token.Text(), 0, 64); err == nil {
			return int(num)
		}
	}
	return 0
}

func (n *expressionLiteral) String() string {
	if token := n.Value(); token != nil && token.Id() == lexer.TokenString {
		return token.Text()
	}
	return ""
}

// ============================================================================
// expression_identifier: identifier
// ============================================================================

type ExpressionIdentifier interface {
	Expression
	Identifier() lexer.Token
}

type expressionIdentifier struct {
	parserNodeData
}

func (n *expressionIdentifier) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *expressionIdentifier) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *expressionIdentifier) ExpressionKind() ExpressionKind {
	return ExprIdentifier
}

func (n *expressionIdentifier) Identifier() lexer.Token {
	if len(n.parserNodeData._tokens) > 0 {
		return n.parserNodeData._tokens[0]
	}
	return nil
}

// ============================================================================
// label: identifier ':'
// ============================================================================

type Label interface {
	ParserNode
	Name() string
}

type label struct {
	parserNodeData
}

func (n *label) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *label) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *label) Name() string {
	tokens := n.parserNodeData.tokensOf(lexer.TokenIdentifier)
	if len(tokens) > 0 {
		return tokens[0].Text()
	}
	return ""
}

// ============================================================================
// bool_literal: 'true' | 'false'
// ============================================================================

type BoolLiteral interface {
	ParserNode
	Value() bool
}

type boolLiteral struct {
	parserNodeData
}

func (n *boolLiteral) Children() []ParserNode {
	return n.parserNodeData.Children()
}

func (n *boolLiteral) Tokens() []lexer.Token {
	return n.parserNodeData.Tokens()
}

func (n *boolLiteral) Value() bool {
	tokens := n.parserNodeData.tokensOf(lexer.TokenTrue)
	return len(tokens) > 0
}
