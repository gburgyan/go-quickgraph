package quickgraph

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Wrapper is the top-level GraphQL wrapper.
type Wrapper struct {
	Mode         string        `parser:"@Ident?"`
	OperationDef *OperationDef `parser:"@@?"`
	Commands     []Command     `parser:"( '{' @@+ '}' )+"`
	Fragments    []Fragment    `parser:"(FragmentToken @@)*"`
}

type OperationDef struct {
	Name      string        `parser:"@Ident"`
	Variables []VariableDef `parser:"( '(' @@ (',' @@)* ')' )?"`
}

type VariableDef struct {
	Name  string        `parser:"@Variable ':'"`
	Type  string        `parser:"'['? @Ident '!'? ']'? '!'?"`
	Value *GenericValue `parser:"('=' @@)?"`
}

// Command is a GraphQL command. This will be 'query' or 'mutation.'
type Command struct {
	Alias        *string        `parser:"(@Ident ':')?"`
	Name         string         `parser:"@Ident"`
	Parameters   *ParameterList `parser:"('(' @@ ')')?"`
	ResultFilter *ResultFilter  `parser:"('{' @@ '}')?"`
	Pos          lexer.Position
}

// ParameterList is a list of parameters for a call to a function.
type ParameterList struct {
	Values []NamedValue `parser:"(@@ (',' @@)*)?"`
}

// NamedValue is a named value. This is used for both parameters and object initialization.
type NamedValue struct {
	Name  string       `parser:"@Ident ':'"`
	Value GenericValue `parser:"@@"`
}

// GenericValue is a value of some type.
type GenericValue struct {
	Variable   *string        `parser:"@Variable"`
	Identifier *string        `parser:"| @Ident"`
	String     *string        `parser:"| @String"`
	Int        *int64         `parser:"| @Int"`
	Float      *float64       `parser:"| @Float"`
	Map        []NamedValue   `parser:"| '{' ( @@ (',' @@)*)? '}'"`
	List       []GenericValue `parser:"| '[' ( @@ (',' @@)*)? ']'"`
}

// ResultFilter is a filter for the result.
type ResultFilter struct {
	Fields    []ResultField  `parser:"@@*"`
	Fragments []FragmentCall `parser:"(FragmentStart @@)*"`
}

// ResultField is a field in the result to be returned.
type ResultField struct {
	Name       string         `parser:"@Ident"`
	Params     *ParameterList `parser:"('(' @@ ')')?"`
	Directives []Directive    `parser:"@@*"`
	SubParts   *ResultFilter  `parser:"('{' @@ '}')?"`
}

type FragmentCall struct {
	Inline      *FragmentDef `parser:"@@"`
	FragmentRef *string      `parser:"| @Ident "`
}

type Fragment struct {
	Name       string       `parser:"@Ident"`
	Definition *FragmentDef `parser:"@@"`
}

type FragmentDef struct {
	TypeName string        `parser:"'on' @Ident"`
	Filter   *ResultFilter `parser:"'{' @@ '}'"`
}

type Directive struct {
	Name       string         `parser:"@Directive"`
	Parameters *ParameterList `parser:"('(' @@ ')')?"`
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
	parser = participle.MustBuild[Wrapper](
		participle.Lexer(graphQLLexer),
		participle.Elide("Whitespace", "Comment"),
		participle.UseLookahead(2),
	)
)

func ParseRequest(input string) (Wrapper, error) {
	r, err := parser.ParseString("", input)
	if err != nil {
		return Wrapper{}, err
	}
	return *r, nil
}
