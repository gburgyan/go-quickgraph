package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type Course struct {
	Title      string  `json:"title"`
	Instructor string  `json:"instructor"`
	Price      float64 `json:"price"`
}

var courses = []*Course{
	{
		Title:      "Golang",
		Instructor: "John Doe",
		Price:      10.99,
	},
	{
		Title:      "Python",
		Instructor: "Jane Doe",
		Price:      12.99,
	},
	{
		Title:      "Java",
		Instructor: "Jack Doe",
		Price:      11.99,
	},
	{
		Title:      "Ruby",
		Instructor: "Jill Doe",
		Price:      9.99,
	},
	{
		Title:      "C++",
		Instructor: "James Doe",
		Price:      8.99,
	},
	{
		Title:      "C#",
		Instructor: "Judy Doe",
		Price:      7.99,
	},
}

func GetCourses(ctx context.Context, categories []*string) []*Course {
	if len(categories) == 0 {
		return courses
	}

	filteredCourses := []*Course{}
	for _, course := range courses {
		for _, category := range categories {
			if strings.Contains(course.Title, *category) {
				filteredCourses = append(filteredCourses, course)
			}
		}
	}
	return filteredCourses
}

func TestCourses_Graph(t *testing.T) {
	input := `
query GetCourses($categories: [String!]) {
  alias: courses(categories: $categories) {
    title
    instructor
    __typename
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
	g := Graphy{}
	g.RegisterProcessorWithParamNames(ctx, "courses", GetCourses, "categories")

	resultAny, err := g.ProcessRequest(ctx, input, vars)
	assert.NoError(t, err)

	assert.Equal(t, `{"data":{"alias":[{"__typename":"Course","instructor":"John Doe","title":"Golang"},{"__typename":"Course","instructor":"Judy Doe","title":"C#"}]}}`, resultAny)
}
