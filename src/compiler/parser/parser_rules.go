package parser

import (
	"zenith/compiler"
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
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// code_block: (statement | expression_statement | function_invocation | variable_declaration | variable_assignment)*
// ============================================================================

func (ctx *parserContext) codeBlock() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenBracesOpen) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '{'

	children := []ParserNode{}
	errors := make([]*compiler.Diagnostic, 0)
	for !ctx.is(lexer.TokenBracesClose) && !ctx.is(lexer.TokenEOF) {
		node := ctx.parseOr([]func() ParserNode{
			ctx.variableDeclaration,
			ctx.variableAssignment,
			// leave statement last.
			ctx.statement,
		})
		if node == nil {
			// not an error, empty block is valid
			break
		}
		children = append(children, node)
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.appendError(&errors, "expected '}' to close code block")
	} else {
		ctx.next(skipEOL) // consume '}'
	}
	return &codeBlock{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
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
		if expr != nil {
			children = append(children, expr)
		}
	}

	// Must have either type or initializer
	if typeRefNode == nil && len(children) < 2 {
		ctx.gotoMark(mark)
		return nil
	}

	return &variableDeclaration{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
		},
	}
}

// ============================================================================
// variable_assignment: identifier (operator_arithmetic | operator_bitwise)? '=' expression end
// Note: Also supports subscript expressions like arr[i] = value
// ============================================================================

func (ctx *parserContext) variableAssignment() ParserNode {
	mark := ctx.mark()

	// Parse lvalue using existing expression postfix logic
	// This handles identifier, subscripts, and member access
	lvalue := ctx.expressionPostfix()
	if lvalue == nil {
		ctx.gotoMark(mark)
		return nil
	}

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

	rvalue := ctx.expression()
	if rvalue == nil {
		ctx.gotoMark(mark)
		return nil
	}

	return &variableAssignment{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: []ParserNode{lvalue, rvalue},
			tokens:   ctx.fromMark(mark),
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
	errors := make([]*compiler.Diagnostic, 0)

	// Optional parameter list
	if !ctx.is(lexer.TokenParenClose) {
		params := ctx.declarationFieldList()
		if params != nil {
			children = append(children, params)
		}
	}

	if !ctx.is(lexer.TokenParenClose) {
		ctx.appendError(&errors, "expected ')'")
	} else {
		ctx.next(skipEOL) // consume ')'
	}

	// Optional return type
	if !ctx.is(lexer.TokenBracesOpen) {
		typeRefNode := ctx.typeReference()
		if typeRefNode != nil {
			children = append(children, typeRefNode)
		}
	}

	bodyNode := ctx.codeBlock()
	if bodyNode == nil {
		ctx.appendError(&errors, "expected function body")
	}
	children = append(children, bodyNode)

	return &functionDeclaration{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// function_invocation: identifier '(' function_argumentList ')'
// ============================================================================

func (ctx *parserContext) functionInvocation() ParserNode {
	mark := ctx.mark()

	var isIntrinsic bool
	if ctx.is(lexer.TokenAtSign) {
		isIntrinsic = true
		ctx.next(skipEOL) // consume '@'
	}

	if !ctx.is(lexer.TokenIdentifier) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	if !ctx.is(lexer.TokenParenOpen) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '('

	children := []ParserNode{}
	errors := make([]*compiler.Diagnostic, 0)

	// Optional argument list
	if !ctx.is(lexer.TokenParenClose) {
		args := ctx.functionArgumentList()
		if args != nil {
			children = append(children, args)
		}
	}

	if !ctx.is(lexer.TokenParenClose) {
		ctx.appendError(&errors, "expected ')'")
	} else {
		ctx.next(skipEOL) // consume ')'
	}

	return &expressionFunctionInvocation{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
		isIntrinsic: isIntrinsic,
	}
}

// function_argumentList: (expression (',' expression)*)?
func (ctx *parserContext) functionArgumentList() ParserNode {
	mark := ctx.mark()
	children := []ParserNode{}

	expr := ctx.expression()
	if expr == nil {
		ctx.gotoMark(mark)
		return nil
	}
	children = append(children, expr)

	errors := make([]*compiler.Diagnostic, 0)
	for ctx.is(lexer.TokenComma) {
		ctx.next(skipEOL) // consume ','
		expr := ctx.expression()
		if expr == nil {
			ctx.appendError(&errors, "expected expression after ','")
			break
		}
		children = append(children, expr)
	}

	return &functionArgumentList{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// type_declaration: 'struct' identifier type_declaration_fields
// ============================================================================

func (ctx *parserContext) typeDeclaration() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenStruct) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'struct'

	errors := make([]*compiler.Diagnostic, 0)
	if !ctx.is(lexer.TokenIdentifier) {
		ctx.appendError(&errors, "expected identifier after 'struct'")
	} else {
		ctx.next(skipEOL) // consume identifier
	}

	children := []ParserNode{}
	fields := ctx.typeDeclarationFields()
	if fields == nil {
		ctx.appendError(&errors, "expected struct fields")
	} else {
		children = append(children, fields)
	}

	return &typeDeclaration{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// type_declaration_fields: '{' declaration_fieldlist '}'
func (ctx *parserContext) typeDeclarationFields() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenBracesOpen) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '{'

	errors := make([]*compiler.Diagnostic, 0)
	children := []ParserNode{}
	fields := ctx.declarationFieldList()
	if fields == nil {
		ctx.appendError(&errors, "expected field list")
	} else {
		children = append(children, fields)
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.appendError(&errors, "expected '}' to close struct fields")
	} else {
		ctx.next(skipEOL) // consume '}'
	}

	return &typeDeclarationFields{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// type_ref: identifier ('[' number? ']')?
// ============================================================================

func (ctx *parserContext) typeReference() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	errors := make([]*compiler.Diagnostic, 0)
	// Optional array syntax
	if ctx.is(lexer.TokenBracketOpen) {
		ctx.next(skipEOL) // consume '['

		// Optional array size
		if ctx.is(lexer.TokenNumber) {
			ctx.next(skipEOL) // consume number
		}

		if !ctx.is(lexer.TokenBracketClose) {
			ctx.appendError(&errors, "expected ']'")
		} else {
			ctx.next(skipEOL) // consume ']'
		}
	}

	// pointer types (e.g. u8*) can be denoted with a trailing '*'
	if ctx.is(lexer.TokenAsterisk) {
		ctx.next(skipEOL) // consume '*'
	}

	return &typeRef{
		parserNodeData: parserNodeData{
			source: ctx.source,
			tokens: ctx.fromMark(mark),
			errors: errors,
		},
	}
}

// ============================================================================
// type_initializer: '{' type_initializer_fieldlist? '}'
// ============================================================================

func (ctx *parserContext) typeInitializer() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenBracesOpen) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume '{'

	children := []ParserNode{}
	errors := make([]*compiler.Diagnostic, 0)

	// Optional field list
	if !ctx.is(lexer.TokenBracesClose) {
		fields := ctx.typeInitializerFieldList()
		if fields != nil {
			children = append(children, fields)
		}
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.appendError(&errors, "expected '}' to close type initializer")
	} else {
		ctx.next(skipEOL) // consume '}'
	}

	return &typeInitializer{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// type_initializer_fieldlist: type_initializer_field (',' type_initializer_field)*
func (ctx *parserContext) typeInitializerFieldList() ParserNode {
	mark := ctx.mark()
	children := []ParserNode{}

	field := ctx.typeInitializerField()
	if field == nil {
		ctx.gotoMark(mark)
		return nil
	}
	children = append(children, field)

	errors := make([]*compiler.Diagnostic, 0)
	for ctx.is(lexer.TokenComma) {
		ctx.next(skipEOL) // consume ','
		field := ctx.typeInitializerField()
		if field == nil {
			ctx.appendError(&errors, "expected field after ','")
			break
		}
		children = append(children, field)
	}

	return &typeInitializerFieldList{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// type_initializer_field: identifier '=' expression
func (ctx *parserContext) typeInitializerField() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	errors := make([]*compiler.Diagnostic, 0)
	children := []ParserNode{}

	if !ctx.is(lexer.TokenEquals) {
		ctx.appendError(&errors, "expected '=' in type initializer field")
	}
	ctx.next(skipEOL) // consume '='

	expr := ctx.expression()
	if expr == nil {
		ctx.appendError(&errors, "expected expression after '=")
	} else {
		children = append(children, expr)
	}

	return &typeInitializerField{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// array_initializer: '[' (expression (',' expression)*)? ']'
// ============================================================================

func (ctx *parserContext) arrayInitializer() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenBracketOpen) {
		return nil
	}
	ctx.next(skipEOL) // consume '['

	children := []ParserNode{}
	errors := make([]*compiler.Diagnostic, 0)

	// Check for empty array: []
	if ctx.is(lexer.TokenBracketClose) {
		ctx.next(skipEOL) // consume ']'
		return &arrayInitializer{
			parserNodeData: parserNodeData{
				source:   ctx.source,
				children: children,
				tokens:   ctx.fromMark(mark),
				errors:   errors,
			},
		}
	}

	// Parse first expression
	expr := ctx.expression()
	if expr == nil {
		ctx.gotoMark(mark)
		return nil
	}
	children = append(children, expr)

	for ctx.is(lexer.TokenComma) {
		ctx.next(skipEOL) // consume ','

		// Allow trailing comma before ']'
		if ctx.is(lexer.TokenBracketClose) {
			break
		}

		expr := ctx.expression()
		if expr == nil {
			ctx.appendError(&errors, "expected expression after ','")
			break
		}
		children = append(children, expr)
	}

	if !ctx.is(lexer.TokenBracketClose) {
		ctx.appendError(&errors, "expected ']' to close array initializer")
	} else {
		ctx.next(skipEOL) // consume ']'
	}

	return &arrayInitializer{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// type_alias: 'type' identifier '=' type_ref end
// ============================================================================

func (ctx *parserContext) typeAlias() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenType) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'type'

	errors := make([]*compiler.Diagnostic, 0)
	if !ctx.is(lexer.TokenIdentifier) {
		ctx.appendError(&errors, "expected identifier after 'type'")
	} else {
		ctx.next(skipEOL) // consume identifier
	}

	if !ctx.is(lexer.TokenEquals) {
		ctx.appendError(&errors, "expected '=' in type alias")
	} else {
		ctx.next(skipEOL) // consume '='
	}

	children := []ParserNode{}
	typeRefNode := ctx.typeReference()
	if typeRefNode == nil {
		ctx.appendError(&errors, "expected type reference")
	} else {
		children = append(children, typeRefNode)
	}

	return &typeAlias{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
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
		ctx.gotoMark(mark)
		return nil
	}
	children = append(children, field)

	errors := make([]*compiler.Diagnostic, 0)
	for ctx.is(lexer.TokenComma) {
		ctx.next(skipEOL) // consume ','
		field := ctx.declarationField()
		if field == nil {
			ctx.appendError(&errors, "expected field after ','")
			break
		}
		children = append(children, field)
	}

	// Check if there's another field without a comma (common error)
	if ctx.is(lexer.TokenIdentifier) {
		ctx.appendError(&errors, "expected ',' between fields")
	}

	return &declarationFieldList{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// declaration_field: label type_ref
func (ctx *parserContext) declarationField() ParserNode {
	mark := ctx.mark()

	labelNode := ctx.label()
	if labelNode == nil {
		ctx.gotoMark(mark)
		return nil
	}

	typeRefNode := ctx.typeReference()
	if typeRefNode == nil {
		ctx.gotoMark(mark)
		return nil
	}

	return &declarationField{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: []ParserNode{labelNode, typeRefNode},
			tokens:   ctx.fromMark(mark),
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
		ctx.statementReturn,
		ctx.statementExpression,
	})
}

// ============================================================================
// statement_if: 'if' expression '{' code_block '}' ('elsif' expression '{' code_block '}')* ('else' '{' code_block '}')?
// ============================================================================

func (ctx *parserContext) statementIf() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIf) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'if'

	errors := make([]*compiler.Diagnostic, 0)
	children := []ParserNode{}
	condition := ctx.expression()
	if condition == nil {
		ctx.appendError(&errors, "expected condition after 'if'")
	} else {
		children = append(children, condition)
	}

	thenBlock := ctx.codeBlock()
	if thenBlock == nil {
		ctx.appendError(&errors, "expected code block after condition")
	} else {
		children = append(children, thenBlock)
	}

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
			ctx.appendError(&errors, "expected code block after 'else'")
		} else {
			children = append(children, elseBlock)
		}
	}

	return &statementIf{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// Helper for elsif clause
func (ctx *parserContext) statementElsif() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenElsif) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'elsif'

	errors := make([]*compiler.Diagnostic, 0)
	children := []ParserNode{}
	condition := ctx.expression()
	if condition == nil {
		ctx.appendError(&errors, "expected condition after 'elsif'")
	} else {
		children = append(children, condition)
	}

	block := ctx.codeBlock()
	if block == nil {
		ctx.appendError(&errors, "expected code block after condition")
	} else {
		children = append(children, block)
	}

	return &statementElsif{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// statement_for: 'for' (statement_for_init ';')? expression (';' expression)? '{' code_block '}'
// ============================================================================

func (ctx *parserContext) statementFor() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenFor) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'for'

	children := []ParserNode{}
	errors := make([]*compiler.Diagnostic, 0)

	// Optional initializer
	if !ctx.is(lexer.TokenSemiColon) {
		// Try variable declaration first
		init := ctx.variableDeclaration()
		if init != nil {
			// Check that variable declaration has an initializer
			if varDecl, ok := init.(VariableDeclaration); ok {
				if varDecl.Initializer() == nil {
					ctx.appendError(&errors, "variable declaration in for-loop initialization must have an initializer")
				}
			}
			children = append(children, init)
		} else {
			// Try variable assignment
			init = ctx.variableAssignment()
			if init != nil {
				children = append(children, init)
			}
		}

		if init != nil {
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
		ctx.appendError(&errors, "expected condition in for loop")
	} else {
		children = append(children, condition)
	}

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
		ctx.appendError(&errors, "expected code block in for loop")
	} else {
		children = append(children, body)
	}

	return &statementFor{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// statement_select: 'select' expression '{' statement_select_cases statement_select_else? '}'
// ============================================================================

func (ctx *parserContext) statementSelect() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenSelect) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'select'

	errors := make([]*compiler.Diagnostic, 0)
	children := []ParserNode{}
	expr := ctx.expression()
	if expr == nil {
		ctx.appendError(&errors, "expected expression after 'select'")
	} else {
		children = append(children, expr)
	}

	if !ctx.is(lexer.TokenBracesOpen) {
		ctx.appendError(&errors, "expected '{' after select expression")
	} else {
		ctx.next(skipEOL) // consume '{'
	}

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

	// ensure at least one case or else clause
	if (len(compiler.OfTypeInterface[*statementSelectCase, StatementSelectCase](children)) == 0) &&
		len(compiler.OfTypeInterface[*statementSelectElse, StatementSelectElse](children)) == 0 {
		ctx.appendError(&errors, "expected at least one 'case' or 'else' clause in select statement")
	}

	if !ctx.is(lexer.TokenBracesClose) {
		ctx.appendError(&errors, "expected '}' to close select statement")
	} else {
		ctx.next(skipEOL) // consume '}'
	}

	return &statementSelect{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// statement_select_cases: 'case' expression '{' code_block '}'
func (ctx *parserContext) statementSelectCase() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenCase) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'case'

	errors := make([]*compiler.Diagnostic, 0)
	children := []ParserNode{}
	expr := ctx.expression()
	if expr == nil {
		ctx.appendError(&errors, "expected expression after 'case'")
	} else {
		children = append(children, expr)
	}

	block := ctx.codeBlock()
	if block == nil {
		ctx.appendError(&errors, "expected code block after case")
	} else {
		children = append(children, block)
	}

	return &statementSelectCase{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// statement_select_else: 'else' '{' code_block '}'
func (ctx *parserContext) statementSelectElse() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenElse) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'else'

	errors := make([]*compiler.Diagnostic, 0)
	children := []ParserNode{}
	block := ctx.codeBlock()
	if block == nil {
		ctx.appendError(&errors, "expected code block after 'else'")
	} else {
		children = append(children, block)
	}

	return &statementSelectElse{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: children,
			tokens:   ctx.fromMark(mark),
			errors:   errors,
		},
	}
}

// ============================================================================
// statement_return: 'ret' expression?
// ============================================================================

func (ctx *parserContext) statementReturn() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenReturn) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume 'ret'

	// Optional expression
	expr := ctx.expression()

	return &statementReturn{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: []ParserNode{expr},
			tokens:   ctx.fromMark(mark),
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
		ctx.gotoMark(mark)
		return nil
	}

	return &statementExpression{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: []ParserNode{expr},
			tokens:   ctx.fromMark(mark),
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
			// Can't parse right side - rewind to before operator
			ctx.gotoMark(mark)
			return nil
		}

		left = &expressionOperatorBinLogical{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					source:   ctx.source,
					children: []ParserNode{left, right},
					tokens:   ctx.fromMark(mark),
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
			// Can't parse right side - rewind to before operator
			ctx.gotoMark(mark)
			return nil
		}

		return &expressionOperatorBinComparison{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					source:   ctx.source,
					children: []ParserNode{left, right},
					tokens:   ctx.fromMark(mark),
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
			// Can't parse right side - rewind to before operator
			ctx.gotoMark(mark)
			return nil
		}

		left = &expressionOperatorBinBitwise{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					source:   ctx.source,
					children: []ParserNode{left, right},
					tokens:   ctx.fromMark(mark),
				},
			},
		}
	}

	return left
}

// expressionBinaryArithmetic: handles '+' | '-' | '*' | '/' | '%'
func (ctx *parserContext) expressionBinaryArithmetic() ParserNode {
	left := ctx.expressionUnary()
	if left == nil {
		return nil
	}

	for ctx.isAny([]lexer.TokenId{
		lexer.TokenPlus, lexer.TokenMinus, lexer.TokenAsterisk, lexer.TokenSlash, lexer.TokenPercent,
	}) {
		mark := ctx.mark()
		ctx.next(skipEOL) // consume operator

		right := ctx.expressionUnary()
		if right == nil {
			// Can't parse right side - rewind to before operator
			ctx.gotoMark(mark)
			return nil
		}

		left = &expressionOperatorBinArithmetic{
			expressionOperatorBinary: expressionOperatorBinary{
				parserNodeData: parserNodeData{
					source:   ctx.source,
					children: []ParserNode{left, right},
					tokens:   ctx.fromMark(mark),
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
			//ctx.error("expected expression after unary operator")
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
						source:   ctx.source,
						children: []ParserNode{expr},
						tokens:   ctx.fromMark(mark),
						errors:   make([]*compiler.Diagnostic, 0),
					},
				},
			}
		case lexer.TokenTilde:
			return &expressionOperatorUnipreBitwise{
				expressionOperatorUnaryPrefix: expressionOperatorUnaryPrefix{
					parserNodeData: parserNodeData{
						source:   ctx.source,
						children: []ParserNode{expr},
						tokens:   ctx.fromMark(mark),
						errors:   make([]*compiler.Diagnostic, 0),
					},
				},
			}
		case lexer.TokenNot:
			return &expressionOperatorUnipreLogical{
				expressionOperatorUnaryPrefix: expressionOperatorUnaryPrefix{
					parserNodeData: parserNodeData{
						source:   ctx.source,
						children: []ParserNode{expr},
						tokens:   ctx.fromMark(mark),
						errors:   make([]*compiler.Diagnostic, 0),
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
				//ctx.error("expected identifier after '.'")
				ctx.gotoMark(mark)
				break
			}
			ctx.next(skipEOL) // consume identifier

			left = &expressionMemberAccess{
				parserNodeData: parserNodeData{
					source:   ctx.source,
					children: []ParserNode{left},
					tokens:   ctx.fromMark(mark),
					errors:   make([]*compiler.Diagnostic, 0),
				},
			}
			continue
		}

		// Subscript: expression '[' expression ']'
		if ctx.is(lexer.TokenBracketOpen) {
			ctx.next(skipEOL) // consume '['

			indexExpr := ctx.expression()
			if indexExpr == nil {
				//ctx.error("expected expression for array index")
				ctx.gotoMark(mark)
				break
			}

			if !ctx.is(lexer.TokenBracketClose) {
				//ctx.error("expected ']'")
				ctx.gotoMark(mark)
				break
			}
			ctx.next(skipEOL) // consume ']'

			left = &expressionSubscript{
				parserNodeData: parserNodeData{
					source:   ctx.source,
					children: []ParserNode{left, indexExpr},
					tokens:   ctx.fromMark(mark),
					errors:   make([]*compiler.Diagnostic, 0),
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
						source:   ctx.source,
						children: []ParserNode{left},
						tokens:   ctx.fromMark(mark),
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
						source:   ctx.source,
						children: []ParserNode{left},
						tokens:   ctx.fromMark(mark),
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
	// Array literals use [] and precedence uses (), so no ambiguity
	return ctx.parseOr([]func() ParserNode{
		ctx.expressionArrayInitializer,
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
		//ctx.error("expected expression")
		ctx.gotoMark(mark)
		return nil
	}

	if !ctx.is(lexer.TokenParenClose) {
		//ctx.error("expected ')'")
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume ')'

	return &expressionPrecedence{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: []ParserNode{expr},
			tokens:   ctx.fromMark(mark),
			errors:   make([]*compiler.Diagnostic, 0),
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
				source: ctx.source,
				tokens: ctx.fromMark(mark),
			},
		}
	}

	ctx.gotoMark(mark)
	return nil
}

// expressionIdentifier: simple identifier as expression
func (ctx *parserContext) expressionIdentifier() ParserNode {
	mark := ctx.mark()

	if !ctx.is(lexer.TokenIdentifier) {
		ctx.gotoMark(mark)
		return nil
	}
	ctx.next(skipEOL) // consume identifier

	return &expressionIdentifier{
		parserNodeData: parserNodeData{
			source: ctx.source,
			tokens: ctx.fromMark(mark),
		},
	}
}

// expression_function_invocation: identifier '(' function_argumentList? ')'
func (ctx *parserContext) expressionFunctionInvocation() ParserNode {
	node := ctx.functionInvocation()
	if node == nil || len(node.Errors()) > 0 {
		return nil
	}
	return node
}

// expression_array_initializer: array_initializer
func (ctx *parserContext) expressionArrayInitializer() ParserNode {
	mark := ctx.mark()

	arrayInit := ctx.arrayInitializer()
	if arrayInit == nil {
		return nil
	}

	return &expressionArrayInitializer{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: []ParserNode{arrayInit},
			tokens:   ctx.fromMark(mark),
		},
	}
}

// expression_type_initializer: type_ref type_initializer
func (ctx *parserContext) expressionTypeInitializer() ParserNode {
	mark := ctx.mark()

	typeRefNode := ctx.typeReference()
	if typeRefNode == nil {
		ctx.gotoMark(mark)
		return nil
	}

	initNode := ctx.typeInitializer()
	if initNode == nil || len(initNode.Errors()) > 0 {
		ctx.gotoMark(mark)
		return nil
	}

	return &expressionTypeInitializer{
		parserNodeData: parserNodeData{
			source:   ctx.source,
			children: []ParserNode{typeRefNode, initNode},
			tokens:   ctx.fromMark(mark),
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
			source: ctx.source,
			tokens: ctx.fromMark(mark),
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
				source: ctx.source,
				tokens: ctx.fromMark(mark),
			},
		}
	}

	ctx.gotoMark(mark)
	return nil
}

// ============================================================================
// end: eol | eof (removed - EOL now transparent)
// ============================================================================

func (ctx *parserContext) end() bool {
	// EOL handling removed - always return true
	return true
}
