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
	_, outputMap := g.solveInputOutputNameMapping(nil, typeLookups)

	schema := g.schemaForType(TypeOutput, cl, outputMap, nil)
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

interface ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type Character implements ICharacter {
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

type Human implements ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	Height(arg1: String): Float!
	HeightMeters: Float!
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

	schema := g.SchemaDefinition(ctx)

	expected := `type Query {
	search(search: String!): [SearchResult!]!
}

interface ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type Character implements ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}

type ConnectionEdge {
	node: Character
}

type Droid implements ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
	primaryFunction: String!
}

type FriendsConnection {
	edges: [ConnectionEdge]!
	totalCount: Int!
}

type Human implements ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	Height(arg1: String): Float!
	HeightMeters: Float!
	id: String!
	name: String!
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

"""An extended object"""
type ExtendedObject {
	"""The character"""
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

// TestUnionWithInterface tests that unions containing interfaces are expanded to concrete types
func TestUnionWithInterface(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Define an interface (embedded type)
	type Employee struct {
		ID       int
		Name     string
		Email    string
		Salary   float64
		HireDate string
	}

	// Define concrete types that embed Employee
	type Manager struct {
		Employee
		Department string
		TeamSize   int
	}

	type Developer struct {
		Employee
		Language string
		Level    string
	}

	// Define other concrete types for the union
	type Product struct {
		ID    int
		Name  string
		Price float64
	}

	type Widget struct {
		ID   int
		Name string
	}

	// Register the search function that returns a union including the interface
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "search",
		Function: func(query string) any {
			return nil
		},
		ReturnAnyOverride: []any{Employee{}, Product{}, Widget{}},
		Mode:              ModeQuery,
	})

	// Register functions to ensure concrete types are in schema
	g.RegisterQuery(ctx, "getManager", func() Manager {
		return Manager{
			Employee: Employee{
				ID:       1,
				Name:     "John Doe",
				Email:    "john@example.com",
				Salary:   100000,
				HireDate: "2020-01-01",
			},
			Department: "Engineering",
			TeamSize:   5,
		}
	})

	g.RegisterQuery(ctx, "getDeveloper", func() Developer {
		return Developer{
			Employee: Employee{
				ID:       2,
				Name:     "Jane Smith",
				Email:    "jane@example.com",
				Salary:   90000,
				HireDate: "2021-01-01",
			},
			Language: "Go",
			Level:    "Senior",
		}
	})

	schema := g.SchemaDefinition(ctx)

	// The union should list concrete types including Employee
	assert.Contains(t, schema, "union searchResultUnion = Developer | Employee | Manager | Product | Widget")

	// Verify IEmployee is an interface and Employee is a concrete type
	assert.Contains(t, schema, "interface IEmployee {")
	assert.Contains(t, schema, "type Employee implements IEmployee {")

	// Verify the concrete types implement IEmployee
	assert.Contains(t, schema, "type Manager implements IEmployee {")
	assert.Contains(t, schema, "type Developer implements IEmployee {")
}

// TestUnionWithMultipleInterfaces tests unions with types implementing multiple interfaces
func TestUnionWithMultipleInterfaces(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Define two interfaces
	type Flyable struct {
		MaxAltitude int
		WingSpan    float64
	}

	type Swimmable struct {
		MaxDepth  int
		SwimSpeed float64
	}

	// Define types that implement one or both
	type Duck struct {
		Flyable
		Swimmable
		Name string
	}

	type Airplane struct {
		Flyable
		Model string
	}

	type Fish struct {
		Swimmable
		Species string
	}

	// Register a function that returns a union of interfaces
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "getVehicle",
		Function: func() any {
			return nil
		},
		ReturnAnyOverride: []any{Flyable{}, Swimmable{}},
		Mode:              ModeQuery,
	})

	// Ensure concrete types are in schema
	g.RegisterQuery(ctx, "getDuck", func() Duck {
		return Duck{
			Flyable:   Flyable{MaxAltitude: 1000, WingSpan: 1.5},
			Swimmable: Swimmable{MaxDepth: 10, SwimSpeed: 5},
			Name:      "Donald",
		}
	})
	g.RegisterQuery(ctx, "getAirplane", func() Airplane {
		return Airplane{
			Flyable: Flyable{MaxAltitude: 35000, WingSpan: 50},
			Model:   "Boeing 747",
		}
	})
	g.RegisterQuery(ctx, "getFish", func() Fish {
		return Fish{
			Swimmable: Swimmable{MaxDepth: 200, SwimSpeed: 20},
			Species:   "Salmon",
		}
	})

	schema := g.SchemaDefinition(ctx)

	// The union should contain all concrete types including the embedded types
	assert.Contains(t, schema, "union getVehicleResultUnion = Airplane | Duck | Fish | Flyable | Swimmable")

	// Verify the interfaces exist with I prefix
	assert.Contains(t, schema, "interface IFlyable {")
	assert.Contains(t, schema, "interface ISwimmable {")

	// Verify concrete types for embedded types
	assert.Contains(t, schema, "type Flyable implements IFlyable {")
	assert.Contains(t, schema, "type Swimmable implements ISwimmable {")

	// Verify Duck implements both interfaces
	assert.Contains(t, schema, "type Duck implements IFlyable & ISwimmable {")
}
