package examples

import (
	"GoGraphy"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCourses_Graph(t *testing.T) {
	input := `
{
  alias: courses(categories: ["Golang", "C#"]) {
    title
    instructor
  }
}`

	ctx := context.Background()
	g := quickgraph.Graphy{}
	g.RegisterMutatorWithParamNames(ctx, "courses", GetCourses, "categories")

	resultAny, err := g.ProcessRequest(ctx, input)
	assert.NoError(t, err)

	assert.Equal(t, `{"data":{"alias":[{"instructor":"John Doe","title":"Golang"},{"instructor":"Judy Doe","title":"C#"}]}}`, resultAny)
}
