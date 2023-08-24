package quickgraph

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestGraphy_schemaForType(t *testing.T) {
	g := Graphy{}
	c := Character{}

	cl := g.typeLookup(reflect.TypeOf(c))
	schema, extraTypes, err := g.schemaForType(cl)
	assert.NoError(t, err)
	expected := `type Character {
	appearsIn: [String!]
	friends: [Character]
	id: String!
	name: String!
}
`
	assert.Equal(t, expected, schema)
	assert.Len(t, extraTypes, 1)
	charType := reflect.TypeOf(Character{})
	assert.Equal(t, charType, extraTypes[0])
}

func TestGraphy_schemataForTypes(t *testing.T) {
	g := Graphy{}
	c := Character{}

	cl := g.typeLookup(reflect.TypeOf(c))

	schema, err := g.schemataForTypes(cl)
	assert.NoError(t, err)
	expected := `type Character {
	appearsIn: [episode!]
	friends: [Character]
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
