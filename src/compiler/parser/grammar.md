# Zenith Parser

## Rules

- `()` group
- `?` optional
- `*` zero or more
- `+` one or more
- `|` or
- `'x'` literal (token)
- `#` comment

Parser Rules:

> All tokens (including whitespace, comments, EOL) are preserved in the parse tree for code regeneration.
> The parser skips whitespace/comments when matching grammar rules but stores them as trivia.

```txt
compilationUnit:
    (variable_declaration | function_declaration | type_declaration)*

code_block:
    (statement | expression_statement | function_invocation | variable_declaration | variable_assignment)*

variable_declaration:
    variable_declaration_type | variable_declaration_inferred
variable_declaration_inferred:
    label '=' expression
variable_declaration_type:
    label type_ref ('=' expression)?
variable_assignment:
    identifier (operator_arithmetic | operator_bitwise)? '=' expression

function_declaration:
    label '(' declaration_fieldlist? ')' type_ref? '{' code_block '}'
function_argumentList:
    (expression (',' expression)*)?

type_declaration:
    'struct' identifier type_declaration_fields
type_declaration_fields:
    '{' declaration_fieldlist '}'
type_ref:
    identifier ('[' number? ']')?
type_initializer:       # structs
    '{' type_initializer_fieldlist? '}'
type_initializer_fieldlist:
    type_initializer_field (',' type_initializer_field)*
type_initializer_field:
    identifier '=' expression
type_alias:
    'type' identifier '=' type_ref

declaration_fieldlist:
    declaration_field (',' declaration_field)*
declaration_field:
    label type_ref

statement:
    statement_if | statement_for | statement_select | statement_expression
statement_if:
    'if' expression '{' code_block '}'
        ('elsif' expression '{' code_block '}')*
        ('else' '{' code_block '}')?
statement_for:
    'for' (statement_for_init ';')? expression (';' expression)? '{' code_block '}'
statement_for_init:
    # requires extra validation for var-init
    variable_declaration | variable_assignment
statement_select:
    'select' expression '{' statement_select_cases statement_select_else? '}'
statement_select_cases:
    'case' expression '{' code_block '}'
statement_select_else:
    'else' '{' code_block '}'
statement_expression:
    expression_function_invocation

expression:
    expression_precedence |
    expression_operator_unaryprefix |
    expression_operator_unarypostfix |
    expression_operator_binary |
    expression_function_invocation |
    expression_type_initializer |
    expression_member_access |
    expression_literal |
    identifier

expression_precedence:
    '(' expression ')'
expression_member_access:
    expression '.' identifier
expression_operator_binary:
    expression_operator_bin_arithmetic | expression_operator_bin_bitwise | expression_operator_bin_comparison | expression_operator_bin_logical
expression_operator_unaryprefix:
    expression_operator_unipre_arithmetic | expression_operator_unipre_bitwise | expression_operator_unipre_logical
expression_operator_unarypostfix:
    expression_operator_unipost_arithmetic | expression_operator_unipost_logical
expression_function_invocation:
    identifier '(' function_argumentList? ')'
expression_type_initializer:
    type_ref type_initializer
expression_literal:
    string | number | bool_literal

expression_operator_bin_arithmetic:
    expression operator_arithmetic expression
expression_operator_bin_bitwise:
    expression operator_bitwise expression
expression_operator_bin_comparison:
    expression ('=' | '>' | '<' | '>=' | '<=' | '<>') expression
expression_operator_bin_logical:
    expression ('and' | 'or') expression
expression_operator_unipre_arithmetic:
    ('-' | '+') expression
expression_operator_unipre_bitwise:
    '~' expression
expression_operator_unipre_logical:
    'not' expression
expression_operator_unipost_arithmetic:
    expression ('++' | '--')
expression_operator_unipost_logical:
    expression '?'

operator_arithmetic:
    '+' | '-' | '*' | '/'
operator_bitwise:
    '&' | '|' | '^'

label:
    identifier ':'
bool_literal:
    'true' | 'false'
end:
    eol | eof

# tokens without a hard predefined value
identifier
string
number
line_comment    # includes eol|eof
# special tokens
whitespace      # spaces, tabs
eol             # end of line
eof             # end of file
```
