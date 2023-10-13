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
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}
`
	assert.Equal(t, expected, schema)
	assert.Len(t, extraTypes, 3)

	assert.Equal(t, "episode", extraTypes[0].name)
	assert.Equal(t, "Character", extraTypes[1].name)
	assert.Equal(t, "FriendsConnection", extraTypes[2].name)
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
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

type ConnectionEdge {
	node: Character
}

enum episode {
	NEWHOPE
	EMPIRE
	JEDI
}

`
	assert.Equal(t, expected, schema)
}

func TestGraphy_MultiParamFunction(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name:           "Update",
		Function:       func(ep episode, count int) []Character { return nil },
		Mode:           ModeMutation,
		ParameterNames: []string{"Episode", "Count"},
	})

	schema, err := g.SchemaDefinition(ctx)
	assert.NoError(t, err)

	expected := `type Mutation {
	Update(Episode: episode!, Count: Int!): [Character!]!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

type ConnectionEdge {
	node: Character
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
	FriendsConnection(arg1: Int!): FriendsConnection
	Height(arg1: String): Float!
	HeightMeters: Float!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

type ConnectionEdge {
	node: Character
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
	FriendsConnection(arg1: Int!): FriendsConnection
	primaryFunction: String!
}

type Human implements Character {
	FriendsConnection(arg1: Int!): FriendsConnection
	Height(arg1: String): Float!
	HeightMeters: Float!
}

type Starship {
	id: String!
	name: String!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

type ConnectionEdge {
	node: Character
}

enum episode {
	NEWHOPE
	EMPIRE
	JEDI
}

`
	assert.Equal(t, expected, schema)
}
