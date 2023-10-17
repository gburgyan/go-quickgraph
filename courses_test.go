package quickgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type Course struct {
	Title      string  `json:"title"`
	Instructor string  `json:"instructor"`
	Price      float64 `json:"price"`
}

type PriceConvertInput struct {
	Currency string `json:"currency"`
}

func (c *Course) PriceConvert(in PriceConvertInput) (string, error) {
	if in.Currency == "ERR" {
		return "", errors.New("forced error")
	}
	return fmt.Sprintf("%.2f %s", c.Price, in.Currency), nil
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
	g.RegisterQuery(ctx, "courses", GetCourses, "categories")

	resultAny, err := g.ProcessRequest(ctx, input, vars)
	assert.NoError(t, err)

	assert.Equal(t, `{"data":{"alias":[{"__typename":"Course","instructor":"John Doe","title":"Golang"},{"__typename":"Course","instructor":"Judy Doe","title":"C#"}]}}`, resultAny)
}

func TestCourses_Graph_Cache(t *testing.T) {
	input := `
{
  courses(categories: ["C#"]) {
    title
    instructor
  }
}`

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "courses", GetCourses)

	g.RequestCache = simpleCache{
		values: map[string]*simpleCacheEntry{},
	}

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)

	assert.Equal(t, `{"data":{"courses":[{"instructor":"Judy Doe","title":"C#"}]}}`, resultAny)

	cache := g.RequestCache.(simpleCache)
	assert.Len(t, cache.values, 1)

	resultAny, err = g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"courses":[{"instructor":"Judy Doe","title":"C#"}]}}`, resultAny)
}

func Test_Missing_Named_Param(t *testing.T) {
	input := `
{
  courses {
    title
    instructor
  }
}`

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "courses", GetCourses, "categories")

	_, err := g.ProcessRequest(ctx, input, "")
	assert.Error(t, err)

	// Get that as a GraphError
	var ge GraphError
	ok := errors.As(err, &ge)
	assert.True(t, ok)
	assert.Equal(t, "error getting call parameters for function courses (path: courses) [3:3]: missing required parameters: categories", ge.Error())
}

func Test_Missing_Struct_Param(t *testing.T) {
	input := `
{
  courses {
    title
    instructor
  }
}`

	type TestInput struct {
		Categories []*string `json:"categories"`
	}

	f := func(ctx context.Context, input TestInput) []*Course {
		return GetCourses(ctx, input.Categories)
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "courses", f)

	_, err := g.ProcessRequest(ctx, input, "")
	assert.Error(t, err)

	// Get that as a GraphError
	var ge GraphError
	ok := errors.As(err, &ge)
	assert.True(t, ok)
	assert.Equal(t, "error getting call parameters for function courses (path: courses) [3:3]: missing required parameters: categories", ge.Error())
}

func Test_Missing_OutputParam(t *testing.T) {
	input := `
{
  courses {
    title
    instructor
	priceconvert
  }
}`

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "courses", GetCourses, "categories")

	_, err := g.ProcessRequest(ctx, input, "")
	assert.Error(t, err)

	// Get that as a GraphError
	assert.Equal(t, "error validating parameters for PriceConvert (path: courses/PriceConvert) [6:2]: missing parameter currency", err.Error())
	jsonError, _ := json.Marshal(err)
	assert.Equal(t, `{"message":"error validating parameters for PriceConvert: missing parameter currency","locations":[{"line":6,"column":2}],"path":["courses","PriceConvert"]}`, string(jsonError))
}

func Test_MismatchedParams(t *testing.T) {
	input := `
query GetCourses($other: String!) {
  courses {
    title
    instructor
	priceconvert(other: $FOO)
  }
}`

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "courses", GetCourses, "categories")

	_, err := g.ProcessRequest(ctx, input, "")
	assert.Error(t, err)

	// Get that as a GraphError
	assert.Equal(t, "error validating parameters for PriceConvert (path: courses/PriceConvert) [6:2]: missing parameter currency", err.Error())
}

func Test_UnsupportedMode(t *testing.T) {
	input := `
BlahBlah GetCourses {
  courses {
    title
    instructor
	priceconvert(currency: "USD")
  }
}`

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "courses", GetCourses, "categories")

	_, err := g.ProcessRequest(ctx, input, "")
	assert.Error(t, err)

	// Get that as a GraphError
	assert.Equal(t, "unknown/unsupported call mode BlahBlah [2:1]", err.Error())
}

func Test_OutputError(t *testing.T) {
	input := `
query GetCourses {
  courses(categories: ["Golang"]) {
    title
    instructor
	priceconvert(currency: "ERR")
  }
}`

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "courses", GetCourses, "categories")

	_, err := g.ProcessRequest(ctx, input, "")
	assert.Error(t, err)

	// Get that as a GraphError
	assert.Equal(t, "function PriceConvert returned error (path: courses/0/priceconvert) [6:15]: forced error", err.Error())

	jsonError, _ := json.Marshal(err)
	assert.Equal(t, `{"message":"function PriceConvert returned error: forced error","locations":[{"line":6,"column":15}],"path":["courses","0","priceconvert"]}`, string(jsonError))
}
