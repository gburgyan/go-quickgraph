package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestGraphy_schemaForType(t *testing.T) {
	g := Graphy{}
	c := Character{}

	cl := g.typeLookup(reflect.TypeOf(c))
	schema, extraTypes, err := g.schemaForOutputType(cl)
	assert.NoError(t, err)
	expected := `type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}
`
	assert.Equal(t, expected, schema)
	assert.Len(t, extraTypes, 2)

	episodeType := reflect.TypeOf(episode(""))
	charType := reflect.TypeOf(Character{})

	assert.Equal(t, episodeType, extraTypes[0])
	assert.Equal(t, charType, extraTypes[1])
}

func TestGraphy_simpleSchema(t *testing.T) {
	g := Graphy{}
	c := Character{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name:     "sample",
		Function: func() *Character { return &c },
		Mode:     ModeQuery,
	})

	schema, err := g.SchemaDefinition(ctx)
	assert.NoError(t, err)

	expected := `type Query {
	sample(): Character
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}

enum episode {
	NEWHOPE
	EMPIRE
	JEDI
}

`
	assert.Equal(t, expected, schema)
}

func TestGraphy_implementsSchema(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name:     "humans",
		Function: func() []Human { return []Human{} },
		Mode:     ModeQuery,
	})

	schema, err := g.SchemaDefinition(ctx)
	assert.NoError(t, err)

	expected := `type Query {
	humans(): [Human!]!
}

type Human implements Character {
	HeightMeters: Float!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}

enum episode {
	NEWHOPE
	EMPIRE
	JEDI
}

`
	assert.Equal(t, expected, schema)
}

func TestGraphy_enumSchema(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "search",
		Function: func(search string) []SearchResultUnion {
			return []SearchResultUnion{
				{
					Human: &Human{},
				},
			}
		},
		Mode:           ModeQuery,
		ParameterNames: []string{"search"},
	})

	schema, err := g.SchemaDefinition(ctx)
	assert.NoError(t, err)

	expected := `type Query {
	search(search: String!): [SearchResult!]!
}

union SearchResult = Droid | Human | Starship

type Droid implements Character {
	primaryFunction: String!
}

type Human implements Character {
	HeightMeters: Float!
}

type Starship {
	id: String!
	name: String!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}

enum episode {
	NEWHOPE
	EMPIRE
	JEDI
}

`
	assert.Equal(t, expected, schema)
}
