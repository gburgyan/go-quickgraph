package quickgraph

import (
	"context"
	"testing"
)

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
