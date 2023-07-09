package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type character struct {
	Id        string      `json:"id"`
	Name      string      `json:"name"`
	Friends   []character `json:"friends"`
	AppearsIn []episode   `json:"appearsIn"`
}

type episode string

const (
	NewHope episode = "NEWHOPE"
	Empire  episode = "EMPIRE"
	Jedi    episode = "JEDI"
)

type human struct {
	character
	HeightMeters float64 `json:"HeightMeters"`
}

func roundToPrecision(number float64, precision int) float64 {
	scale := math.Pow10(precision)
	return math.Round(number*scale) / scale
}

func (h *human) Height(units *string) float64 {
	if units == nil {
		return h.HeightMeters
	}
	if *units == "FOOT" {
		return roundToPrecision(h.HeightMeters*3.28084, 7)
	}
	return h.HeightMeters
}

func TestSimpleFields1(t *testing.T) {
	var h = character{
		Name: "R2-D2",
	}

	heroProvider := func(ctx context.Context) *character {
		return &h
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessorWithParamNames(ctx, "hero", heroProvider)

	input := `
{
  hero {
    name
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)

	assert.Equal(t, `{"data":{"hero":{"name":"R2-D2"}}}`, resultAny)
}

func TestSimpleFields2(t *testing.T) {
	var h = character{
		Name: "R2-D2",
		Friends: []character{
			{
				Name: "Luke Skywalker",
			},
			{
				Name: "Han Solo",
			},
			{
				Name: "Leia Organa",
			},
		},
	}

	heroProvider := func(ctx context.Context) *character {
		return &h
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessorWithParamNames(ctx, "hero", heroProvider)

	input := `
{
  hero {
    name
    # Queries can have comments!
    friends {
      name
    }
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"hero":{"friends":[{"name":"Luke Skywalker"},{"name":"Han Solo"},{"name":"Leia Organa"}],"name":"R2-D2"}}}`, resultAny)

	// Verify that the result, even if we have more potential fields, is what we expect.
	input = `
{
  hero {
    name
    # Queries can have comments!
  }
}`

	resultAny, err = g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"hero":{"name":"R2-D2"}}}`, resultAny)
}

func TestArguments(t *testing.T) {
	var h = human{
		character: character{
			Name: "Luke Skywalker",
		},
		HeightMeters: 1.72,
	}

	getHumanProvider := func(ctx context.Context, id string) *human {
		return &h
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessorWithParamNames(ctx, "human", getHumanProvider, "id")

	input := `
{
  human(id: "1000") {
    name
    height
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"human":{"height":1.72,"name":"Luke Skywalker"}}}`, resultAny)

	input = `
{
  human(id: "1000") {
    name
    height(unit: FOOT)
  }
}`

	resultAny, err = g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"human":{"height":5.6430448,"name":"Luke Skywalker"}}}`, resultAny)
}
