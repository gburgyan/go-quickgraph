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
	schema, extraTypes, err := g.schemaForOutputType(cl)
	assert.NoError(t, err)
	expected := `type Character {
	appearsIn: [episode!]
	friends: [Character]
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

func TestGraphy_schemataForTypes(t *testing.T) {
	g := Graphy{}
	c := Character{}

	cl := g.typeLookup(reflect.TypeOf(c))

	schema, enums, err := g.schemaForOutputTypes(cl)
	assert.NoError(t, err)

	types, err := g.schemaForEnumTypes(enums...)
	assert.NoError(t, err)

	schema += types

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
