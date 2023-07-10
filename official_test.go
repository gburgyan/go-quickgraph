package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type Character struct {
	Id        string       `json:"id"`
	Name      string       `json:"name"`
	Friends   []*Character `json:"friends"`
	AppearsIn []episode    `json:"appearsIn"`
}

type episode string

const (
	NewHope episode = "NEWHOPE"
	Empire  episode = "EMPIRE"
	Jedi    episode = "JEDI"
)

type Human struct {
	Character
	HeightMeters float64 `json:"HeightMeters"`
}

type Droid struct {
	Character
	PrimaryFunction string `json:"primaryFunction"`
}

func roundToPrecision(number float64, precision int) float64 {
	scale := math.Pow10(precision)
	return math.Round(number*scale) / scale
}

func (h *Human) Height(units *string) float64 {
	if units == nil {
		return h.HeightMeters
	}
	if *units == "FOOT" {
		return roundToPrecision(h.HeightMeters*3.28084, 7)
	}
	return h.HeightMeters
}

func TestSimpleFields1(t *testing.T) {
	var h = Character{
		Name: "R2-D2",
	}

	heroProvider := func(ctx context.Context) *Character {
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
	var h = Character{
		Name: "R2-D2",
		Friends: []*Character{
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

	heroProvider := func(ctx context.Context) *Character {
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
	var h = Human{
		Character: Character{
			Name: "Luke Skywalker",
		},
		HeightMeters: 1.72,
	}

	getHumanProvider := func(ctx context.Context, id string) *Human {
		return &h
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessorWithParamNames(ctx, "Human", getHumanProvider, "id")

	input := `
{
  Human(id: "1000") {
    name
    height
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"Human":{"height":1.72,"name":"Luke Skywalker"}}}`, resultAny)

	input = `
{
  Human(id: "1000") {
    name
    height(unit: FOOT)
  }
}`

	resultAny, err = g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"Human":{"height":5.6430448,"name":"Luke Skywalker"}}}`, resultAny)
}

func TestFragments1(t *testing.T) {
	var luke = Human{
		Character: Character{
			Name: "Luke Skywalker",
			AppearsIn: []episode{
				NewHope,
				Empire,
				Jedi,
			},
			Friends: []*Character{
				{
					Name: "Han Solo",
				},
				{
					Name: "Leia Organa",
				},
				{
					Name: "C-3PO",
				},
				{
					Name: "R2-D2",
				},
			},
		},
	}

	var r2d2 = Droid{
		Character: Character{
			Name: "R2-D2",
			AppearsIn: []episode{
				NewHope,
				Empire,
				Jedi,
			},
			Friends: []*Character{
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
		},
	}

	getHumanProvider := func(ctx context.Context, ep episode) any {
		if ep == Empire {
			return &luke
		} else if ep == Jedi {
			return &r2d2
		}
		return nil
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessorWithParamNames(ctx, "hero", getHumanProvider, "episode")

	//	input := `
	//{
	//  hero() {
	//    field
	//  }
	//}
	//
	//`

	input := `
{
  leftComparison: hero(episode: EMPIRE) {
    ...comparisonFields
  }
  rightComparison: hero(episode: JEDI) {
    ...comparisonFields
  }
}

fragment comparisonFields on Character {
  name
  appearsIn
  friends {
    name
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"leftComparison":{"appearsIn":["NEWHOPE","EMPIRE","JEDI"],"friends":[{"name":"Han Solo"},{"name":"Leia Organa"},{"name":"C-3PO"},{"name":"R2-D2"}],"name":"Luke Skywalker"},"rightComparison":{"appearsIn":["NEWHOPE","EMPIRE","JEDI"],"friends":[{"name":"Luke Skywalker"},{"name":"Han Solo"},{"name":"Leia Organa"}],"name":"R2-D2"}}}`, resultAny)
}
