package quickgraph

import (
	"context"
	"errors"
	"fmt"
	"github.com/gburgyan/go-timing"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

// These tests are all taken verbatim from the official GraphQL documentation
// retrieved from https://graphql.org/learn/queries/.

type Character struct {
	Id        string       `graphy:"id"`
	Name      string       `graphy:"name"`
	Friends   []*Character `graphy:"friends"`
	AppearsIn []episode    `graphy:"appearsIn"`
}

type ConnectionEdge struct {
	Node *Character `graphy:"node"`
}

type FriendsConnection struct {
	TotalCount int               `graphy:"totalCount"`
	Edges      []*ConnectionEdge `graphy:"edges"`
}

type Starship struct {
	Id   string `graphy:"id"`
	Name string `graphy:"name"`
	// TODO: Add support for length.
}

type SearchResultUnion struct {
	Human    *Human
	Droid    *Droid
	Starship *Starship
}

func (c *Character) FriendsConnection(first int) *FriendsConnection {
	result := &FriendsConnection{
		TotalCount: len(c.Friends),
		Edges:      make([]*ConnectionEdge, 0, len(c.Friends)),
	}
	for i, f := range c.Friends {
		if i >= first {
			break
		}
		result.Edges = append(result.Edges, &ConnectionEdge{
			Node: f,
		})
	}
	return result
}

type Review struct {
	Stars      int     `graphy:"stars"`
	Commentary *string `graphy:"commentary"`
	Ignore     *string `graphy:"-"`
}

type episode string

func (e episode) EnumValues() []EnumValue {
	return []EnumValue{
		{Name: "NEWHOPE"},
		{Name: "EMPIRE"},
		{Name: "JEDI"},
	}
}

const (
	NewHope episode = "NEWHOPE"
	Empire  episode = "EMPIRE"
	Jedi    episode = "JEDI"
)

type Human struct {
	Character
	HeightMeters float64 `graphy:"HeightMeters"`
}

type Droid struct {
	Character
	PrimaryFunction string `graphy:"primaryFunction"`
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
	g.RegisterQuery(ctx, "hero", heroProvider)

	input := `
{
  hero {
    name
  }
}`

	stub, err := g.getRequestStub(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, "hero", stub.Name())

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
	g.RegisterQuery(ctx, "hero", heroProvider)

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
	g.RegisterQuery(ctx, "Human", getHumanProvider, "id")

	input := `
{
  Human(id: "1000") {
    name
    height
  }
}`

	stub, err := g.getRequestStub(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, "Human", stub.Name())

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
	g.RegisterQuery(ctx, "hero", getHumanProvider, "episode")

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

	stub, err := g.getRequestStub(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, "leftComparison,rightComparison", stub.Name())

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"leftComparison":{"appearsIn":["NEWHOPE","EMPIRE","JEDI"],"friends":[{"name":"Han Solo"},{"name":"Leia Organa"},{"name":"C-3PO"},{"name":"R2-D2"}],"name":"Luke Skywalker"},"rightComparison":{"appearsIn":["NEWHOPE","EMPIRE","JEDI"],"friends":[{"name":"Luke Skywalker"},{"name":"Han Solo"},{"name":"Leia Organa"}],"name":"R2-D2"}}}`, resultAny)
}

func TestFragmentVariable(t *testing.T) {
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

	getHeroFunction := func(ctx context.Context, ep episode) any {
		if ep == Empire {
			return &luke
		} else if ep == Jedi {
			return &r2d2
		}
		return nil
	}

	ctx := context.Background()
	g := Graphy{}
	//g.RegisterProcessorWithParamNames(ctx, "hero", getHeroFunction, "episode")
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:              "hero",
		Function:          getHeroFunction,
		ReturnAnyOverride: []any{Human{}, Droid{}},
	})

	input := `
query HeroComparison($first: Int = 3) {
  leftComparison: hero(episode: EMPIRE) {
    ...comparisonFields
  }
  rightComparison: hero(episode: JEDI) {
    ...comparisonFields
  }
}

fragment comparisonFields on Character {
  name
  friendsConnection(first: $first) {
    totalCount
    edges {
      node {
        name
      }
    }
  }
}`

	stub, err := g.getRequestStub(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, "HeroComparison", stub.Name())

	g.EnableTiming = true

	tCtx, complete := timing.StartRoot(ctx, "GraphRequest")
	resultAny, err := g.ProcessRequest(tCtx, input, "")
	complete()
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"leftComparison":{"friendsConnection":{"edges":[{"node":{"name":"Han Solo"}},{"node":{"name":"Leia Organa"}},{"node":{"name":"C-3PO"}}],"totalCount":4},"name":"Luke Skywalker"},"rightComparison":{"friendsConnection":{"edges":[{"node":{"name":"Luke Skywalker"}},{"node":{"name":"Han Solo"}},{"node":{"name":"Leia Organa"}}],"totalCount":3},"name":"R2-D2"}}}`, resultAny)

	fmt.Printf("timing:\n%s\n", tCtx.String())
}

func TestVariableDefaultValue(t *testing.T) {
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

	heroProvider := func(ctx context.Context, ep string) *Character {
		return &h
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "hero", heroProvider, "episode")

	input := `
query HeroNameAndFriends($episode: Episode = JEDI) {
  hero(episode: $episode) {
    name
    friends {
      name
    }
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"hero":{"friends":[{"name":"Luke Skywalker"},{"name":"Han Solo"},{"name":"Leia Organa"}],"name":"R2-D2"}}}`, resultAny)

	definition := g.SchemaDefinition(ctx)
	fmt.Println(definition)
}

func TestMutatorWithComplexInput(t *testing.T) {

	createReview := func(ctx context.Context, episode episode, review Review) Review {
		return review
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "createReview", createReview, "episode", "review")

	input := `
mutation {
  createReview(episode: "JEDI", review: {stars: 5, commentary: "This is a great movie!"}) {
    stars
    commentary
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"createReview":{"commentary":"This is a great movie!","stars":5}}}`, resultAny)
}

func TestMutatorWithComplexInputVars(t *testing.T) {

	createReview := func(ctx context.Context, episode episode, review Review) Review {
		return review
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "createReview", createReview, "episode", "review")

	input := `
mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
  createReview(episode: $ep, review: $review) {
    stars
    commentary
  }
}`

	vars := `
{
  "ep": "JEDI",
  "review": {
    "stars": 5,
    "commentary": "This is a great movie!"
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, vars)
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"createReview":{"commentary":"This is a great movie!","stars":5}}}`, resultAny)
}

func TestMutatorWithComplexInputVarsWithError(t *testing.T) {

	createReview := func(ctx context.Context, episode episode, review Review) (Review, error) {
		return review, nil
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "createReview", createReview, "episode", "review")

	input := `
mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
  createReview(episode: $ep, review: $review) {
    stars
    commentary
  }
}`

	vars := `
{
  "ep": "JEDI",
  "review": {
    "stars": 5,
    "commentary": "This is a great movie!"
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, vars)
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"createReview":{"commentary":"This is a great movie!","stars":5}}}`, resultAny)
}

func TestMutatorWithComplexInputVarsWithErrorReturned(t *testing.T) {

	createReview := func(ctx context.Context, episode episode, review Review) (Review, error) {
		return review, fmt.Errorf("fixed error return")
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "createReview", createReview, "episode", "review")

	input := `
mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
  createReview(episode: $ep, review: $review) {
    stars
    commentary
  }
}`

	vars := `
{
  "ep": "JEDI",
  "review": {
    "stars": 5,
    "commentary": "This is a great movie!"
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, vars)
	assert.EqualError(t, err, "function createReview returned error (path: createReview) [3:16]: fixed error return")
	assert.Equal(t, `{"data":{},"errors":[{"message":"function createReview returned error: fixed error return","locations":[{"line":3,"column":16}],"path":["createReview"]}]}`, resultAny)
}

func TestMutatorWithComplexInputVarsPanic(t *testing.T) {

	createReview := func(ctx context.Context, episode episode, review Review) (Review, error) {
		panic("fixed error message")
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "createReview", createReview, "episode", "review")

	input := `
mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
  createReview(episode: $ep, review: $review) {
    stars
    commentary
  }
}`

	vars := `
{
  "ep": "JEDI",
  "review": {
    "stars": 5,
    "commentary": "This is a great movie!"
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, vars)
	assert.EqualError(t, err, "function createReview panicked: fixed error message (path: createReview) [3:16]: panic: fixed error message")
	assert.Contains(t, resultAny, `"message":"function createReview panicked: fixed error message: panic: fixed error message"`, resultAny)
	assert.Contains(t, resultAny, `"path":["createReview"]`, resultAny)
	assert.Contains(t, resultAny, `"extensions"`, resultAny)
	assert.Contains(t, resultAny, `"stack"`, resultAny)
	var gErr GraphError
	errors.As(err, &gErr)
	// The stack trace isn't stable so we can't compare it. Just verify we have it.
	// In the new system, stack trace is in SensitiveExtensions in dev mode but shown in the response
	assert.Contains(t, gErr.SensitiveExtensions, "stack")
}

func TestEnumInvalid(t *testing.T) {
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

	heroProvider := func(ctx context.Context, ep episode) *Character {
		return &h
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "hero", heroProvider, "episode")

	input := `
query  {
  hero(episode: INVALID) {
    name
    friends {
      name
    }
  }
}`

	resultAny, err := g.ProcessRequest(ctx, input, "")
	assert.Error(t, err)
	assert.Equal(t, `{"data":{},"errors":[{"message":"error getting call parameters for function hero: invalid enum value INVALID","locations":[{"line":3,"column":8}],"path":["hero"]}]}`, resultAny)
}
