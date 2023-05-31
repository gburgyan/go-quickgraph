package examples

import (
	"GoGraphy"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCourses_Graph(t *testing.T) {
	input := `
query GetCourses($categories: [String!]) {
  alias: courses(categories: $categories) {
    title
    instructor
    ... on CourseA {
      price
	}
    ... on CourseB {
      price
	}
  }
}`
	vars := `
{
	"categories": ["Golang", "C#"]
}`

	ctx := context.Background()
	g := quickgraph.Graphy{}
	g.RegisterProcessorWithParamNames(ctx, "courses", GetCourses, "categories")

	resultAny, err := g.ProcessRequest(ctx, input, vars)
	assert.NoError(t, err)

	assert.Equal(t, `{"data":{"alias":[{"instructor":"John Doe","title":"Golang"},{"instructor":"Judy Doe","title":"C#"}]}}`, resultAny)
}
