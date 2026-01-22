package parser

import (
	"zenith/compiler/lexer"
)

// ============================================================================
// compilationUnit: (variable_declaration | function_declaration | type_declaration)*
// ============================================================================

func (ctx *parserContext) compilationUnit() ParserNode {
	mark := ctx.mark()
	children := []ParserNode{}

	for {
		node := ctx.parseOr([]func() ParserNode{
			ctx.variableDeclaration,
			ctx.functionDeclaration,
			ctx.typeDeclaration,
		})
		if node == nil {
			break
		}
		children = append(children, node)
	}

	return &compilationUnit{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// code_block: (statement | expression_statement | function_invocation | variable_declaration | variable_assignment)*
// ============================================================================

func (ctx *parserContext) codeBlock() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenBracesOpen) {
		return nil
	}
	ctx.next(skipEOL) // consume '{'

	children := []ParserNode{}
	for !ctx.is(lexer.TokenBracesClose) && !ctx.is(lexer.TokenEOF) {
		node := ctx.parseOr([]func() ParserNode{
			ctx.statement,
			ctx.variableDeclaration,
			ctx.variableAssignment,
		})
		if node == nil {
			break
		}
		children = append(children, node)
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.error("expected '}' to close code block")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '}'

	return &codeBlock{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// variable_declaration: variable_declaration_type | variable_declaration_inferred
// ============================================================================

// variable_declaration: label type_ref? ('=' expression)?
func (ctx *parserContext) variableDeclaration() ParserNode {
	mark := ctx.mark()

	labelNode := ctx.label()
	if labelNode == nil {
		ctx.gotoMark(mark)
		return nil
	}

	children := []ParserNode{labelNode}

	// Optional type reference
	typeRefNode := ctx.typeReference()
	if typeRefNode != nil {
		children = append(children, typeRefNode)
	}

	// Optional initializer
	if ctx.is(lexer.TokenEquals) {
		ctx.next(skipEOL) // consume '='
		expr := ctx.expression()
		if expr == nil {
			ctx.error("expected expression after '='")
			ctx.gotoMark(mark)
			return nil
		}
		children = append(children, expr)
	}

	// Must have either type or initializer
	if typeRefNode == nil && len(children) < 2 {
		ctx.error("variable declaration must have either type or initializer")
		ctx.gotoMark(mark)
		return nil
	}

	return &variableDeclaration{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// variable_assignment: identifier (operator_arithmetic | operator_bitwise)? '=' expression end
// ============================================================================

func (ctx *parserContext) variableAssignment() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	// Optional compound operator
	if ctx.isAny([]lexer.TokenId{
		lexer.TokenPlus, lexer.TokenMinus, lexer.TokenAsterisk, lexer.TokenSlash,
		lexer.TokenAmpersant, lexer.TokenPipe, lexer.TokenCaret,
	}) {
		ctx.next(skipEOL) // consume operator
	}

	if !ctx.is(lexer.TokenEquals) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '='

	expr := ctx.expression()
	if expr == nil {
		ctx.error("expected expression after '='")
		ctx.gotoMark(mark)
		return nil
	}

	return &variableAssignment{
		parserNodeData: parserNodeData{
			_children: []ParserNode{expr},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// function_declaration: label '(' declaration_fieldlist? ')' type_ref? '{' code_block '}'
// ============================================================================

func (ctx *parserContext) functionDeclaration() ParserNode {
	mark := ctx.mark()

	labelNode := ctx.label()
	if labelNode == nil {
		ctx.gotoMark(mark)
		return nil
	}

	if !ctx.is(lexer.TokenParenOpen) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '('

	children := []ParserNode{labelNode}

	// Optional parameter list
	if !ctx.is(lexer.TokenParenClose) {
		params := ctx.declarationFieldList()
		if params != nil {
			children = append(children, params)
		}
	}

	if !ctx.is(lexer.TokenParenClose) {
		ctx.error("expected ')'")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume ')'

	// Optional return type
	if !ctx.is(lexer.TokenBracesOpen) {
		typeRefNode := ctx.typeReference()
		if typeRefNode != nil {
			children = append(children, typeRefNode)
		}
	}

	bodyNode := ctx.codeBlock()
	if bodyNode == nil {
		ctx.error("expected function body")
		ctx.gotoMark(mark)
		return nil
	}
	children = append(children, bodyNode)

	return &functionDeclaration{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// function_invocation: identifier '(' function_argumentList ')'
// ============================================================================

func (ctx *parserContext) functionInvocation() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	if !ctx.is(lexer.TokenParenOpen) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '('

	children := []ParserNode{}

	// Optional argument list
	if !ctx.is(lexer.TokenParenClose) {
		args := ctx.functionArgumentList()
		if args != nil {
			children = append(children, args)
		}
	}

	if !ctx.is(lexer.TokenParenClose) {
		ctx.error("expected ')'")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume ')'

	return &expressionFunctionInvocation{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// function_argumentList: (expression (',' expression)*)?
func (ctx *parserContext) functionArgumentList() ParserNode {
	mark := ctx.mark()
	children := []ParserNode{}

	expr := ctx.expression()
	if expr == nil {
		return nil
	}
	children = append(children, expr)

	for ctx.is(lexer.TokenComma) {
		ctx.next(skipEOL) // consume ','
		expr := ctx.expression()
		if expr == nil {
			ctx.error("expected expression after ','")
			break
		}
		children = append(children, expr)
	}

	return &functionArgumentList{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// type_declaration: 'struct' identifier type_declaration_fields
// ============================================================================

func (ctx *parserContext) typeDeclaration() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenStruct) {
		return nil
	}
	ctx.next(skipEOL) // consume 'struct'

	if !ctx.is(lexer.TokenIdentifier) {
		ctx.error("expected identifier after 'struct'")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	fields := ctx.typeDeclarationFields()
	if fields == nil {
		ctx.error("expected struct fields")
		ctx.gotoMark(mark)
		return nil
	}

	return &typeDeclaration{
		parserNodeData: parserNodeData{
			_children: []ParserNode{fields},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// type_declaration_fields: '{' declaration_fieldlist '}'
func (ctx *parserContext) typeDeclarationFields() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenBracesOpen) {
		return nil
	}
	ctx.next(skipEOL) // consume '{'

	fields := ctx.declarationFieldList()
	if fields == nil {
		ctx.error("expected field list")
		ctx.gotoMark(mark)
		return nil
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.error("expected '}' to close struct fields")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '}'

	return &typeDeclarationFields{
		parserNodeData: parserNodeData{
			_children: []ParserNode{fields},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// type_ref: identifier ('[' number? ']')?
// ============================================================================

func (ctx *parserContext) typeReference() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	// Optional array syntax
	if ctx.is(lexer.TokenBracketOpen) {
		ctx.next(skipEOL) // consume '['

		// Optional array size
		if ctx.is(lexer.TokenNumber) {
			ctx.next(skipEOL) // consume number
		}

		if !ctx.is(lexer.TokenBracketClose) {
			ctx.error("expected ']'")
			ctx.gotoMark(mark)
			return nil
		}
		ctx.next(skipEOL) // consume ']'
	}

	return &typeRef{
		parserNodeData: parserNodeData{
			_tokens: ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// type_initializer: '{' type_initializer_fieldlist? '}'
// ============================================================================

func (ctx *parserContext) typeInitializer() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenBracesOpen) {
		return nil
	}
	ctx.next(skipEOL) // consume '{'

	children := []ParserNode{}

	// Optional field list
	if !ctx.is(lexer.TokenBracesClose) {
		fields := ctx.typeInitializerFieldList()
		if fields != nil {
			children = append(children, fields)
		}
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.error("expected '}' to close type initializer")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '}'

	return &typeInitializer{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// type_initializer_fieldlist: type_initializer_field (',' type_initializer_field)*
func (ctx *parserContext) typeInitializerFieldList() ParserNode {
	mark := ctx.mark()
	children := []ParserNode{}

	field := ctx.typeInitializerField()
	if field == nil {
		return nil
	}
	children = append(children, field)

	for ctx.is(lexer.TokenComma) {
		ctx.next(skipEOL) // consume ','
		field := ctx.typeInitializerField()
		if field == nil {
			ctx.error("expected field after ','")
			break
		}
		children = append(children, field)
	}

	return &typeInitializerFieldList{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// type_initializer_field: identifier '=' expression
func (ctx *parserContext) typeInitializerField() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	if !ctx.is(lexer.TokenEquals) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '='

	expr := ctx.expression()
	if expr == nil {
		ctx.error("expected expression after '='")
		ctx.gotoMark(mark)
		return nil
	}

	return &typeInitializerField{
		parserNodeData: parserNodeData{
			_children: []ParserNode{expr},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// type_alias: 'type' identifier '=' type_ref end
// ============================================================================

func (ctx *parserContext) typeAlias() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenType) {
		return nil
	}
	ctx.next(skipEOL) // consume 'type'

	if !ctx.is(lexer.TokenIdentifier) {
		ctx.error("expected identifier after 'type'")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	if !ctx.is(lexer.TokenEquals) {
		ctx.error("expected '=' in type alias")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '='

	typeRefNode := ctx.typeReference()
	if typeRefNode == nil {
		ctx.error("expected type reference")
		ctx.gotoMark(mark)
		return nil
	}

	return &typeAlias{
		parserNodeData: parserNodeData{
			_children: []ParserNode{typeRefNode},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// declaration_fieldlist: declaration_field (',' declaration_field)*
// ============================================================================

func (ctx *parserContext) declarationFieldList() ParserNode {
	mark := ctx.mark()
	children := []ParserNode{}

	field := ctx.declarationField()
	if field == nil {
		return nil
	}
	children = append(children, field)

	for ctx.is(lexer.TokenComma) {
		ctx.next(skipEOL) // consume ','
		field := ctx.declarationField()
		if field == nil {
			ctx.error("expected field after ','")
			break
		}
		children = append(children, field)
	}

	return &declarationFieldList{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// declaration_field: label type_ref
func (ctx *parserContext) declarationField() ParserNode {
	mark := ctx.mark()

	labelNode := ctx.label()
	if labelNode == nil {
		return nil
	}

	typeRefNode := ctx.typeReference()
	if typeRefNode == nil {
		ctx.gotoMark(mark)
		return nil
	}

	return &declarationField{
		parserNodeData: parserNodeData{
			_children: []ParserNode{labelNode, typeRefNode},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// statement: statement_if | statement_for | statement_select | statement_expression
// ============================================================================

func (ctx *parserContext) statement() ParserNode {
	return ctx.parseOr([]func() ParserNode{
		ctx.statementIf,
		ctx.statementFor,
		ctx.statementSelect,
		ctx.statementExpression,
	})
}

// ============================================================================
// statement_if: 'if' expression '{' code_block '}' ('elsif' expression '{' code_block '}')* ('else' '{' code_block '}')?
// ============================================================================

func (ctx *parserContext) statementIf() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIf) {
		return nil
	}
	ctx.next(skipEOL) // consume 'if'

	condition := ctx.expression()
	if condition == nil {
		ctx.error("expected condition after 'if'")
		ctx.gotoMark(mark)
		return nil
	}

	thenBlock := ctx.codeBlock()
	if thenBlock == nil {
		ctx.error("expected code block after condition")
		ctx.gotoMark(mark)
		return nil
	}

	children := []ParserNode{condition, thenBlock}

	// Optional elsif clauses
	for ctx.is(lexer.TokenElsif) {
		elsifNode := ctx.statementElsif()
		if elsifNode != nil {
			children = append(children, elsifNode)
		}
	}

	// Optional else clause
	if ctx.is(lexer.TokenElse) {
		ctx.next(skipEOL) // consume 'else'
		elseBlock := ctx.codeBlock()
		if elseBlock == nil {
			ctx.error("expected code block after 'else'")
		} else {
			children = append(children, elseBlock)
		}
	}

	return &statementIf{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// Helper for elsif clause
func (ctx *parserContext) statementElsif() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenElsif) {
		return nil
	}
	ctx.next(skipEOL) // consume 'elsif'

	condition := ctx.expression()
	if condition == nil {
		ctx.error("expected condition after 'elsif'")
		ctx.gotoMark(mark)
		return nil
	}

	block := ctx.codeBlock()
	if block == nil {
		ctx.error("expected code block after condition")
		ctx.gotoMark(mark)
		return nil
	}

	return &statementElsif{
		parserNodeData: parserNodeData{
			_children: []ParserNode{condition, block},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// statement_for: 'for' (statement_for_init ';')? expression (';' expression)? '{' code_block '}'
// ============================================================================

func (ctx *parserContext) statementFor() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenFor) {
		return nil
	}
	ctx.next(skipEOL) // consume 'for'

	children := []ParserNode{}

	// Optional initializer
	if !ctx.is(lexer.TokenSemiColon) {
		init := ctx.parseOr([]func() ParserNode{
			ctx.variableDeclaration,
			ctx.variableAssignment,
		})
		if init != nil {
			children = append(children, init)
			if ctx.is(lexer.TokenSemiColon) {
				ctx.next(skipEOL) // consume ';'
			}
		}
	} else {
		ctx.next(skipEOL) // consume ';' (empty init)
	}

	// Condition (required)
	condition := ctx.expression()
	if condition == nil {
		ctx.error("expected condition in for loop")
		ctx.gotoMark(mark)
		return nil
	}
	children = append(children, condition)

	// Optional increment
	if ctx.is(lexer.TokenSemiColon) {
		ctx.next(skipEOL) // consume ';'
		increment := ctx.expression()
		if increment != nil {
			children = append(children, increment)
		}
	}

	body := ctx.codeBlock()
	if body == nil {
		ctx.error("expected code block in for loop")
		ctx.gotoMark(mark)
		return nil
	}
	children = append(children, body)

	return &statementFor{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// statement_select: 'select' expression '{' statement_select_cases statement_select_else? '}'
// ============================================================================

func (ctx *parserContext) statementSelect() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenSelect) {
		return nil
	}
	ctx.next(skipEOL) // consume 'select'

	expr := ctx.expression()
	if expr == nil {
		ctx.error("expected expression after 'select'")
		ctx.gotoMark(mark)
		return nil
	}

	if !ctx.is(lexer.TokenBracesOpen) {
		ctx.error("expected '{' after select expression")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '{'

	children := []ParserNode{expr}

	// Parse case clauses
	for ctx.is(lexer.TokenCase) {
		caseNode := ctx.statementSelectCase()
		if caseNode == nil {
			break
		}
		children = append(children, caseNode)
	}

	// Optional else clause
	if ctx.is(lexer.TokenElse) {
		elseNode := ctx.statementSelectElse()
		if elseNode != nil {
			children = append(children, elseNode)
		}
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.error("expected '}' to close select statement")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '}'

	return &statementSelect{
		parserNodeData: parserNodeData{
			_children: children,
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// statement_select_cases: 'case' expression '{' code_block '}'
func (ctx *parserContext) statementSelectCase() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenCase) {
		return nil
	}
	ctx.next(skipEOL) // consume 'case'

	expr := ctx.expression()
	if expr == nil {
		ctx.error("expected expression after 'case'")
		ctx.gotoMark(mark)
		return nil
	}

	block := ctx.codeBlock()
	if block == nil {
		ctx.error("expected code block after case")
		ctx.gotoMark(mark)
		return nil
	}

	return &statementSelectCase{
		parserNodeData: parserNodeData{
			_children: []ParserNode{expr, block},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// statement_select_else: 'else' '{' code_block '}'
func (ctx *parserContext) statementSelectElse() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenElse) {
		return nil
	}
	ctx.next(skipEOL) // consume 'else'

	block := ctx.codeBlock()
	if block == nil {
		ctx.error("expected code block after 'else'")
		ctx.gotoMark(mark)
		return nil
	}

	return &statementSelectElse{
		parserNodeData: parserNodeData{
			_children: []ParserNode{block},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// statement_expression: expression_function_invocation end
// ============================================================================

func (ctx *parserContext) statementExpression() ParserNode {
	mark := ctx.mark()

	expr := ctx.expressionFunctionInvocation()
	if expr == nil {
		return nil
	}

	return &statementExpression{
		parserNodeData: parserNodeData{
			_children: []ParserNode{expr},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// expression: entry point - handles precedence by trying rules in order
// ============================================================================

func (ctx *parserContext) expression() ParserNode {
	// Start with lowest precedence (binary logical)
	return ctx.expressionBinaryLogical()
}

// expressionBinaryLogical: handles 'and' | 'or'
func (ctx *parserContext) expressionBinaryLogical() ParserNode {
	left := ctx.expressionBinaryComparison()
	if left == nil {
		return nil
	}

	for ctx.isAny([]lexer.TokenId{lexer.TokenAnd, lexer.TokenOr}) {
		mark := ctx.mark()
		ctx.next(skipEOL) // consume operator

		right := ctx.expressionBinaryComparison()
		if right == nil {
			ctx.error("expected expression after logical operator")
			ctx.gotoMark(mark)
			return nil
		}

		left = &expressionOperatorBinLogical{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					_children: []ParserNode{left, right},
					_tokens:   ctx.fromMark(mark),
				},
			},
		}
	}

	return left
}

// expressionBinaryComparison: handles '=' | '>' | '<' | '>=' | '<=' | '<>'
func (ctx *parserContext) expressionBinaryComparison() ParserNode {
	left := ctx.expressionBinaryBitwise()
	if left == nil {
		return nil
	}

	if ctx.isAny([]lexer.TokenId{
		lexer.TokenEquals, lexer.TokenGreater, lexer.TokenLess,
		lexer.TokenGreaterOrEquals, lexer.TokenLessOrEquals, lexer.TokenNotEquals,
	}) {
		mark := ctx.mark()
		ctx.next(skipEOL) // consume operator

		right := ctx.expressionBinaryBitwise()
		if right == nil {
			ctx.error("expected expression after comparison operator")
			ctx.gotoMark(mark)
			return nil
		}

		return &expressionOperatorBinComparison{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					_children: []ParserNode{left, right},
					_tokens:   ctx.fromMark(mark),
				},
			},
		}
	}

	return left
}

// expressionBinaryBitwise: handles '&' | '|' | '^'
func (ctx *parserContext) expressionBinaryBitwise() ParserNode {
	left := ctx.expressionBinaryArithmetic()
	if left == nil {
		return nil
	}

	for ctx.isAny([]lexer.TokenId{
		lexer.TokenAmpersant, lexer.TokenPipe, lexer.TokenCaret,
	}) {
		mark := ctx.mark()
		ctx.next(skipEOL) // consume operator

		right := ctx.expressionBinaryArithmetic()
		if right == nil {
			ctx.error("expected expression after bitwise operator")
			ctx.gotoMark(mark)
			return nil
		}

		left = &expressionOperatorBinBitwise{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					_children: []ParserNode{left, right},
					_tokens:   ctx.fromMark(mark),
				},
			},
		}
	}

	return left
}

// expressionBinaryArithmetic: handles '+' | '-' | '*' | '/'
func (ctx *parserContext) expressionBinaryArithmetic() ParserNode {
	left := ctx.expressionUnary()
	if left == nil {
		return nil
	}

	for ctx.isAny([]lexer.TokenId{
		lexer.TokenPlus, lexer.TokenMinus, lexer.TokenAsterisk, lexer.TokenSlash,
	}) {
		mark := ctx.mark()
		ctx.next(skipEOL) // consume operator

		right := ctx.expressionUnary()
		if right == nil {
			ctx.error("expected expression after arithmetic operator")
			ctx.gotoMark(mark)
			return nil
		}

		left = &expressionOperatorBinArithmetic{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					_children: []ParserNode{left, right},
					_tokens:   ctx.fromMark(mark),
				},
			},
		}
	}

	return left
}

// expressionUnary: handles unary prefix and postfix operators
func (ctx *parserContext) expressionUnary() ParserNode {
	// Try unary prefix operators: '-' | '+' | '~' | 'not'
	if ctx.isAny([]lexer.TokenId{
		lexer.TokenMinus, lexer.TokenPlus, lexer.TokenTilde, lexer.TokenNot,
	}) {
		mark := ctx.mark()
		ctx.next(skipEOL) // consume operator

		expr := ctx.expressionUnary() // recursive for multiple unary ops
		if expr == nil {
			ctx.error("expected expression after unary operator")
			ctx.gotoMark(mark)
			return nil
		}

		// Determine specific unary type based on operator
		tok := ctx.fromMark(mark)[0]
		switch tok.Id() {
		case lexer.TokenMinus, lexer.TokenPlus:
			return &expressionOperatorUnipreArithmetic{
				expressionOperatorUnaryPrefix: expressionOperatorUnaryPrefix{
					parserNodeData: parserNodeData{
						_children: []ParserNode{expr},
						_tokens:   ctx.fromMark(mark),
					},
				},
			}
		case lexer.TokenTilde:
			return &expressionOperatorUnipreBitwise{
				expressionOperatorUnaryPrefix: expressionOperatorUnaryPrefix{
					parserNodeData: parserNodeData{
						_children: []ParserNode{expr},
						_tokens:   ctx.fromMark(mark),
					},
				},
			}
		case lexer.TokenNot:
			return &expressionOperatorUnipreLogical{
				expressionOperatorUnaryPrefix: expressionOperatorUnaryPrefix{
					parserNodeData: parserNodeData{
						_children: []ParserNode{expr},
						_tokens:   ctx.fromMark(mark),
					},
				},
			}
		}
	}

	// Parse postfix (member access, function call, array index, postfix operators)
	return ctx.expressionPostfix()
}

// expressionPostfix: handles member access, function calls, and postfix operators
func (ctx *parserContext) expressionPostfix() ParserNode {
	left := ctx.expressionPrimary()
	if left == nil {
		return nil
	}

	for {
		mark := ctx.mark()

		// Member access: expression '.' identifier
		if ctx.is(lexer.TokenPeriod) {
			ctx.next(skipEOL) // consume '.'

			if !ctx.is(lexer.TokenIdentifier) {
				ctx.error("expected identifier after '.'")
				ctx.gotoMark(mark)
				break
			}
			ctx.next(skipEOL) // consume identifier

			left = &expressionMemberAccess{
				parserNodeData: parserNodeData{
					_children: []ParserNode{left},
					_tokens:   ctx.fromMark(mark),
				},
			}
			continue
		}

		// Postfix arithmetic: '++' | '--'
		if ctx.isAny([]lexer.TokenId{lexer.TokenIncrement, lexer.TokenDecrement}) {
			ctx.next(skipEOL) // consume operator

			left = &expressionOperatorUnipostArithmetic{
				expressionOperatorUnaryPostfix: expressionOperatorUnaryPostfix{
					parserNodeData: parserNodeData{
						_children: []ParserNode{left},
						_tokens:   ctx.fromMark(mark),
					},
				},
			}
			continue
		}

		// Postfix logical: '?'
		if ctx.is(lexer.TokenQuestion) {
			ctx.next(skipEOL) // consume '?'

			left = &expressionOperatorUnipostLogical{
				expressionOperatorUnaryPostfix: expressionOperatorUnaryPostfix{
					parserNodeData: parserNodeData{
						_children: []ParserNode{left},
						_tokens:   ctx.fromMark(mark),
					},
				},
			}
			continue
		}

		// No more postfix operators
		break
	}

	return left
}

// expressionPrimary: handles base expressions (literals, identifiers, parentheses, etc.)
func (ctx *parserContext) expressionPrimary() ParserNode {
	// Try alternatives in order
	return ctx.parseOr([]func() ParserNode{
		ctx.expressionPrecedence,
		ctx.expressionFunctionInvocation,
		ctx.expressionTypeInitializer,
		ctx.expressionLiteral,
		ctx.expressionIdentifier,
	})
}

// expression_precedence: '(' expression ')'
func (ctx *parserContext) expressionPrecedence() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenParenOpen) {
		return nil
	}
	ctx.next(skipEOL) // consume '('

	expr := ctx.expression()
	if expr == nil {
		ctx.error("expected expression")
		ctx.gotoMark(mark)
		return nil
	}

	if !ctx.is(lexer.TokenParenClose) {
		ctx.error("expected ')'")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume ')'

	return &expressionPrecedence{
		parserNodeData: parserNodeData{
			_children: []ParserNode{expr},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// expression_literal: string | number | bool_literal
func (ctx *parserContext) expressionLiteral() ParserNode {
	mark := ctx.mark()

	if ctx.is(lexer.TokenString) || ctx.is(lexer.TokenNumber) ||
		ctx.is(lexer.TokenTrue) || ctx.is(lexer.TokenFalse) {
		ctx.next(skipEOL) // consume literal
		return &expressionLiteral{
			parserNodeData: parserNodeData{
				_tokens: ctx.fromMark(mark),
			},
		}
	}

	return nil
}

// expressionIdentifier: simple identifier as expression
func (ctx *parserContext) expressionIdentifier() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	return &expression{
		parserNodeData: parserNodeData{
			_tokens: ctx.fromMark(mark),
		},
	}
}

// expression_function_invocation: identifier '(' function_argumentList? ')'
func (ctx *parserContext) expressionFunctionInvocation() ParserNode {
	return ctx.functionInvocation()
}

// expression_type_initializer: type_ref type_initializer
func (ctx *parserContext) expressionTypeInitializer() ParserNode {
	mark := ctx.mark()

	typeRefNode := ctx.typeReference()
	if typeRefNode == nil {
		return nil
	}

	initNode := ctx.typeInitializer()
	if initNode == nil {
		ctx.gotoMark(mark)
		return nil
	}

	return &expressionTypeInitializer{
		parserNodeData: parserNodeData{
			_children: []ParserNode{typeRefNode, initNode},
			_tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// label: identifier ':'
// ============================================================================

func (ctx *parserContext) label() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	if !ctx.is(lexer.TokenColon) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume ':'

	return &label{
		parserNodeData: parserNodeData{
			_tokens: ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// bool_literal: 'true' | 'false'
// ============================================================================

func (ctx *parserContext) boolLiteral() ParserNode {
	mark := ctx.mark()

	if ctx.is(lexer.TokenTrue) || ctx.is(lexer.TokenFalse) {
		ctx.next(skipEOL) // consume bool literal
		return &boolLiteral{
			parserNodeData: parserNodeData{
				_tokens: ctx.fromMark(mark),
			},
		}
	}

	return nil
}

// ============================================================================
// end: eol | eof (removed - EOL now transparent)
// ============================================================================

func (ctx *parserContext) end() bool {
	// EOL handling removed - always return true
	return true
}
