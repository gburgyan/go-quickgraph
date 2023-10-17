package quickgraph

import (
	"context"
	"testing"
)

type simpleCacheEntry struct {
	request string
	stub    *RequestStub
	err     error
}

type simpleCache struct {
	values map[string]*simpleCacheEntry
}

func (s simpleCache) GetRequestStub(ctx context.Context, request string) (*RequestStub, error) {
	rs, found := s.values[request]
	if !found {
		return nil, nil
	}
	return rs.stub, rs.err
}

func (s simpleCache) SetRequestStub(ctx context.Context, request string, stub *RequestStub, err error) {
	cacheEntry := &simpleCacheEntry{
		request: request,
		stub:    stub,
		err:     err,
	}
	s.values[request] = cacheEntry
}

func BenchmarkMutatorWithVars_Cached(b *testing.B) {

	createReview := func(ctx context.Context, episode episode, review Review) Review {
		return review
	}

	ctx := context.Background()
	g := Graphy{
		RequestCache: simpleCache{
			values: map[string]*simpleCacheEntry{},
		},
	}

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

	for i := 0; i < b.N; i++ {
		_, _ = g.ProcessRequest(ctx, input, vars)
	}
}

func BenchmarkMissingProcessor(b *testing.B) {
	ctx := context.Background()
	g := Graphy{
		RequestCache: simpleCache{
			values: map[string]*simpleCacheEntry{},
		},
	}

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
