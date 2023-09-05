package quickgraph

import (
	"context"
	"testing"
)

type simpleCache struct {
	values map[string]*RequestStub
}

func (s simpleCache) GetRequestStub(request string) (*RequestStub, error) {
	rs, found := s.values[request]
	if !found {
		return nil, nil
	}
	return rs, nil
}

func (s simpleCache) SetRequestStub(request string, stub *RequestStub) {
	s.values[request] = stub
}

func BenchmarkMutatorWithVars_Cached(b *testing.B) {

	createReview := func(ctx context.Context, episode episode, review Review) Review {
		return review
	}

	ctx := context.Background()
	g := Graphy{
		RequestCache: simpleCache{
			values: map[string]*RequestStub{},
		},
	}

	g.RegisterProcessorWithParamNames(ctx, "createReview", createReview, "episode", "review")

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

	for i := 0; i < b.N; i++ {
		_, _ = g.ProcessRequest(ctx, input, vars)
	}
}

func BenchmarkMutatorWithVars(b *testing.B) {

	createReview := func(ctx context.Context, episode episode, review Review) Review {
		return review
	}

	ctx := context.Background()
	g := Graphy{}

	g.RegisterProcessorWithParamNames(ctx, "createReview", createReview, "episode", "review")

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

	for i := 0; i < b.N; i++ {
		_, _ = g.ProcessRequest(ctx, input, vars)
	}
}
