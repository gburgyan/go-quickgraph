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
	schema, err := g.schemaForType(cl)
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
