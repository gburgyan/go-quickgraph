package quickgraph

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestGraphy_schemataForType(t *testing.T) {
	g := Graphy{}
	c := Character{}

	cl := g.typeLookup(reflect.TypeOf(c))
	schema, extraTypes, err := g.schemataForType(cl)
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

func TestGraphy_schemaForTypes(t *testing.T) {
	g := Graphy{}
	c := Character{}

	cl := g.typeLookup(reflect.TypeOf(c))

	schema, err := g.schemaForTypes(cl)
	assert.NoError(t, err)
	expected := `type Character {
	appearsIn: [String!]
	friends: [Character]
	id: String!
	name: String!
}

`
	assert.Equal(t, expected, schema)
}
