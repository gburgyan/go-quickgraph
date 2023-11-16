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
	assert.Equal(t, "typeLookup: quickgraph.Character", cl.String())

	typeLookups := g.expandTypeLookups([]*typeLookup{cl})
	_, outputMap := solveInputOutputNameMapping(nil, typeLookups)

	schema := g.schemaForType(TypeOutput, cl, outputMap)
	expected := `type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}
`
	assert.Equal(t, expected, schema)
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

	schema := g.SchemaDefinition(ctx)

	expected := `type Query {
	sample: Character
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type ConnectionEdge {
	node: Character
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
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

	g.RegisterMutation(ctx, "Update", func(ep episode, count int) []Character { return nil }, "Episode", "Count")

	schema := g.SchemaDefinition(ctx)

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

type ConnectionEdge {
	node: Character
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
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

	schema := g.SchemaDefinition(ctx)

	expected := `type Query {
	humans: [Human!]!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type ConnectionEdge {
	node: Character
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

type Human implements Character {
	FriendsConnection(arg1: Int!): FriendsConnection
	Height(arg1: String): Float!
	HeightMeters: Float!
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

	schema := g.SchemaDefinition(ctx)

	expected := `type Query {
	search(search: String!): [SearchResult!]!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type ConnectionEdge {
	node: Character
}

type Droid implements Character {
	FriendsConnection(arg1: Int!): FriendsConnection
	primaryFunction: String!
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

type Human implements Character {
	FriendsConnection(arg1: Int!): FriendsConnection
	Height(arg1: String): Float!
	HeightMeters: Float!
}

union SearchResult = Droid | Human | Starship

type Starship {
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

func TestGraphy_MutationWithObject(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	f := func(code int, ship Starship) bool {
		return true
	}

	g.RegisterFunction(ctx, FunctionDefinition{
		Name:           "AddShip",
		Function:       f,
		Mode:           ModeMutation,
		ParameterNames: []string{"code", "ship"},
	})

	schema := g.SchemaDefinition(ctx)

	expected := `type Mutation {
	AddShip(code: Int!, ship: Starship!): Boolean!
}

input Starship {
	id: String!
	name: String!
}

`
	assert.Equal(t, expected, schema)
}

func TestGraphy_MutationObjectFunction(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	f := func(characterId int, input FriendsConnection) FriendsConnection {
		return input
	}

	g.RegisterFunction(ctx, FunctionDefinition{
		Name:           "AddCharacterConnection",
		Function:       f,
		Mode:           ModeMutation,
		ParameterNames: []string{"code", "friends"},
	})

	schema := g.SchemaDefinition(ctx)

	expected := `type Mutation {
	AddCharacterConnection(code: Int!, friends: FriendsConnectionInput!): FriendsConnection!
}

input CharacterInput {
	appearsIn: [episode!]!
	friends: [CharacterInput]!
	id: String!
	name: String!
}

input ConnectionEdgeInput {
	node: CharacterInput
}

input FriendsConnectionInput {
	edges: [ConnectionEdgeInput]!
	totalCount: Int!
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type ConnectionEdge {
	node: Character
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

enum episode {
	NEWHOPE
	EMPIRE
	JEDI
}

`
	assert.Equal(t, expected, schema)
}

type extendedObject struct {
	OldCharacter Character `graphy:"char1,description=The character,deprecated=No longer used"`
}

func newCharacter(e *extendedObject, name string) Character {
	return Character{
		Name: name,
	}
}

func (e *extendedObject) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Name:        "ExtendedObject",
		Description: "An extended object",
		Deprecated:  "shouldn't use this",
		FunctionDefinitions: []FunctionDefinition{
			{
				Name:           "newCharacter",
				Function:       newCharacter,
				ParameterNames: []string{"name"},
			},
		},
	}
}

func TestGraphy_ExtendedObject(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterFunction(ctx, FunctionDefinition{
		Name:     "extended",
		Function: func() *extendedObject { return &extendedObject{} },
		Mode:     ModeQuery,
	})

	schema := g.SchemaDefinition(ctx)

	expected := `type Query {
	extended: ExtendedObject
}

type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type ConnectionEdge {
	node: Character
}

type ExtendedObject {
	char1: Character! @deprecated(reason: "No longer used")
	newCharacter(name: String!): Character!
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

enum episode {
	NEWHOPE
	EMPIRE
	JEDI
}

`
	assert.Equal(t, expected, schema)

	query := `{
  extended {
    newCharacter(name: "test") {
      name
    }
  }
}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"extended":{"newCharacter":{"name":"test"}}}}`, result)
}
