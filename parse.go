package quickgraph

import (
	"errors"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// wrapper is the top-level GraphQL wrapper.
type wrapper struct {
	Mode         string        `parser:"@Ident?"`
	OperationDef *operationDef `parser:"@@?"`
	Commands     []command     `parser:"( '{' @@+ '}' )+"`
	Fragments    []fragment    `parser:"(FragmentToken @@)*"`
	Pos          lexer.Position
}

type operationDef struct {
	Name      string        `parser:"@Ident"`
	Variables []variableDef `parser:"( '(' @@ (',' @@)* ')' )?"`
	Pos       lexer.Position
}

type variableDef struct {
	Name  string        `parser:"@Variable ':'"`
	Type  variableType  `parser:"@@"`
	Value *genericValue `parser:"('=' @@)?"`
	Pos   lexer.Position
}

type variableType struct {
	Array        *variableArrayType    `parser:"( '[' @@ ']'"`
	ConcreteType *variableConcreteType `parser:"| @@ )"`
	IsRequired   string                `parser:"'!'?"`
}

type variableArrayType struct {
	InnerType *variableType `parser:"@@"`
}

type variableConcreteType struct {
	Name string `parser:"@Ident"`
}

// command is a GraphQL command. This will be 'query' or 'mutation.'
type command struct {
	Alias        *string        `parser:"(@Ident ':')?"`
	Name         string         `parser:"@Ident"`
	Parameters   *parameterList `parser:"('(' @@ ')')?"`
	ResultFilter *resultFilter  `parser:"('{' @@ '}')?"`
	Pos          lexer.Position
}

// parameterList is a list of parameters for a call to a function.
type parameterList struct {
	Values []namedValue `parser:"(@@ (',' @@)*)?"`
	Pos    lexer.Position
}

// namedValue is a named value. This is used for both parameters and object initialization.
type namedValue struct {
	Name  string       `parser:"@Ident ':'"`
	Value genericValue `parser:"@@"`
	Pos   lexer.Position
}

// genericValue is a value of some type.
type genericValue struct {
	Variable   *string        `parser:"@Variable"`
	Identifier *string        `parser:"| @Ident"`
	String     *string        `parser:"| @String"`
	Int        *int64         `parser:"| @Int"`
	Float      *float64       `parser:"| @Float"`
	Map        []namedValue   `parser:"| '{' ( @@ (',' @@)* )? '}'"`
	List       []genericValue `parser:"| '[' ( @@ (',' @@)* )? ']'"`
	Pos        lexer.Position
}

// resultFilter is a filter for the result.
type resultFilter struct {
	Fields    []resultField  `parser:"@@*"`
	Fragments []fragmentCall `parser:"(FragmentStart @@)*"`
	Pos       lexer.Position
}

// resultField is a field in the result to be returned.
type resultField struct {
	Name       string         `parser:"@Ident"`
	Params     *parameterList `parser:"('(' @@ ')')?"`
	Directives []directive    `parser:"@@*"`
	SubParts   *resultFilter  `parser:"('{' @@ '}')?"`
	Pos        lexer.Position
}

type fragmentCall struct {
	Inline      *fragmentDef `parser:"@@"`
	FragmentRef *string      `parser:"| @Ident "`
}

type fragment struct {
	Name       string       `parser:"@Ident"`
	Definition *fragmentDef `parser:"@@"`
	Pos        lexer.Position
}

type fragmentDef struct {
	TypeName string        `parser:"'on' @Ident"`
	Filter   *resultFilter `parser:"'{' @@ '}'"`
}

type directive struct {
	Name       string         `parser:"@Directive"`
	Parameters *parameterList `parser:"('(' @@ ')')?"`
	Pos        lexer.Position
}

var (
	graphQLLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"FragmentStart", `\.\.\.`},
		{"FragmentToken", `fragment`},
		{"Ident", `[a-zA-Z_]\w*`},
		//{"TypeName", `[a-zA-Z]\w*`},
		{"Variable", `\$[a-zA-Z]\w*`},
		{"Directive", `@[a-zA-Z]\w*`},
		{"String", `"(([^"])|\\\")*"`},
		{"Float", `\d+\.\d*`},
		{"Int", `\d+`},
		{"Comment", `#[^\n]*`},
		{"Punct", `[-[!@#$%^&*()+_={}\|:;"'<,>.?/]|]`},
		{"Whitespace", `[ \t\n\r]+`},
	})
	parser = participle.MustBuild[wrapper](
		participle.Lexer(graphQLLexer),
		participle.Elide("Whitespace", "Comment"),
		participle.UseLookahead(2),
	)
)

func parseRequest(input string) (wrapper, error) {
	r, err := parser.ParseString("", input)
	if err != nil {
		var pErr participle.Error
		var position lexer.Position
		if errors.As(err, &pErr) {
			position = pErr.Position()
		}
		return wrapper{}, AugmentGraphError(err, "error parsing request", position)
	}
	return *r, nil
}
