package quickgraph

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Wrapper is the top-level GraphQL wrapper.
type Wrapper struct {
	Mode      string         `@Ident?`
	Variables *OperationName `@@?`
	Commands  []Command      `"{" @@+ "}"`
}

type OperationName struct {
	Name      string        `@Ident`
	Variables []VariableDef `("(" @@ ("," @@)* ")")?`
}

type VariableDef struct {
	Name string `@Variable ":"`
	Type string `"["? @Ident "!"? "]"? "!"?`
}

// Command is a GraphQL command. This will be "query" or "mutation".
type Command struct {
	Alias        *string        `(@Ident ":")?`
	Name         string         `@Ident`
	Parameters   *ParameterList `"(" @@ ")"?`
	ResultFilter ResultFilter   `("{" @@ "}")?`
}

// ParameterList is a list of parameters for a call to a function.
type ParameterList struct {
	Values []NamedValue `(@@ ("," @@)*)?`
}

// NamedValue is a named value. This is used for both parameters and object initialization.
type NamedValue struct {
	Name  string       `@Ident ":"`
	Value GenericValue `@@`
}

// GenericValue is a value of some type.
type GenericValue struct {
	Variable *string        `@Variable`
	String   *string        `| @String`
	Int      *int           `| @Int`
	Float    *float64       `| @Float`
	Map      []NamedValue   `| "{" ( @@ ("," @@)*)? "}"`
	List     []GenericValue `| "[" ( @@ ("," @@)*)? "]"`
}

// ResultFilter is a filter for the result.
type ResultFilter struct {
	Fields      []ResultField `@@*`
	UnionLookup []UnionLookup `(TypeLookup @@)*`
}

// ResultField is a field in the result to be returned.
type ResultField struct {
	Name     string         `@Ident`
	Params   *ParameterList `("(" @@ ")")?`
	SubParts []ResultField  `("{" @@* "}")?`
}

type UnionLookup struct {
	TypeName string        `@Ident "{"`
	Fields   []ResultField `@@* "}"`
}

var (
	graphQLLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"TypeLookup", `\.\.\.\W*on`},
		{"Ident", `[a-zA-Z_]\w*`},
		//{"TypeName", `[a-zA-Z]\w*`},
		{"Variable", `\$[a-zA-Z]\w*`},
		{"String", `"(([^"])|\\\")*"`},
		{"Float", `(\d*\.)?\d+`},
		{"Int", `\d+`},
		{"Punct", `[-[!@#$%^&*()+_={}\|:;"'<,>.?/]|]`},
		{"Whitespace", `[ \t\n\r]+`},
	})
	parser = participle.MustBuild[Wrapper](
		participle.Lexer(graphQLLexer),
		participle.Elide("Whitespace"),
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
