package quickgraph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphyTagPrioritySimple(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Test basic tag priority
	type SimpleTest struct {
		GraphyOnly  string `graphy:"g1"`
		JsonOnly    string `json:"j1"`
		BothTags    string `graphy:"g2" json:"j2"`
		NoTags      string
		GraphyEmpty string `graphy:"" json:"j3"`
	}

	getSimple := func(ctx context.Context) *SimpleTest {
		return &SimpleTest{
			GraphyOnly:  "graphy",
			JsonOnly:    "json",
			BothTags:    "both",
			NoTags:      "none",
			GraphyEmpty: "empty",
		}
	}
	g.RegisterQuery(ctx, "getSimple", getSimple)

	// Test query
	query := `query {
		getSimple {
			g1
			j1
			g2
			NoTags
			j3
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	require.NoError(t, err)

	assert.Contains(t, result, `"g1":"graphy"`)
	assert.Contains(t, result, `"j1":"json"`)
	assert.Contains(t, result, `"g2":"both"`) // graphy wins
	assert.Contains(t, result, `"NoTags":"none"`)
	assert.Contains(t, result, `"j3":"empty"`) // empty graphy falls back to json
}

func TestGraphyTagDescriptionSimple(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	type WithDescription struct {
		Field1 string `graphy:"f1,description=Test description"`
		Field2 string `json:"f2" graphy:"description=Another description"`
	}

	getDesc := func(ctx context.Context) *WithDescription {
		return &WithDescription{Field1: "value1", Field2: "value2"}
	}
	g.RegisterQuery(ctx, "getDesc", getDesc)

	// Just test that the query works with the correct field names
	query := `query {
		getDesc {
			f1
			f2
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	require.NoError(t, err)

	assert.Contains(t, result, `"f1":"value1"`)
	assert.Contains(t, result, `"f2":"value2"`)
}

func TestGraphyTagWithFunctionParams(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	type SearchInput struct {
		Query  string `graphy:"q"`
		Limit  int    `graphy:"max" json:"limit"`
		Offset int    `json:"skip"`
	}

	search := func(ctx context.Context, input SearchInput) string {
		return input.Query + "-" + string(rune(input.Limit)) + "-" + string(rune(input.Offset))
	}
	g.RegisterQuery(ctx, "search", search)

	// Test with graphy field names
	query := `query {
		search(q: "test", max: 10, skip: 5)
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	require.NoError(t, err)

	// The function should receive the correct values
	assert.Contains(t, result, "test")
}

func TestGraphyTagExclusion(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	type ExclusionTest struct {
		Included      string `graphy:"inc"`
		GraphyExclude string `graphy:"-"`
		JsonExclude   string `json:"-"`
		BothExclude   string `graphy:"-" json:"shouldNotAppear"`
	}

	getExcl := func(ctx context.Context) *ExclusionTest {
		return &ExclusionTest{
			Included:      "yes",
			GraphyExclude: "no1",
			JsonExclude:   "no2",
			BothExclude:   "no3",
		}
	}
	g.RegisterQuery(ctx, "getExcl", getExcl)

	// Query for all possible fields
	query := `query {
		getExcl {
			inc
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	require.NoError(t, err)

	assert.Contains(t, result, `"inc":"yes"`)

	// Try to query excluded fields - should fail
	badQueries := []string{
		`query { getExcl { GraphyExclude } }`,
		`query { getExcl { JsonExclude } }`,
		`query { getExcl { BothExclude } }`,
		`query { getExcl { shouldNotAppear } }`,
	}

	for _, q := range badQueries {
		_, err := g.ProcessRequest(ctx, q, "")
		assert.Error(t, err, "Query should fail: %s", q)
	}
}

func TestGraphyTagNameFormat(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	type NameFormatTest struct {
		Simple    string `graphy:"simple"`
		Named     string `graphy:"name=custom"`
		Complex   string `graphy:"name=comp,description=Test"`
		Quoted    string `graphy:"name=quoted,description=\"With spaces\""`
		Spaces    string `graphy:"spacey"`
		EmptyName string `graphy:"name="`
	}

	getNF := func(ctx context.Context) *NameFormatTest {
		return &NameFormatTest{
			Simple:    "s",
			Named:     "n",
			Complex:   "c",
			Quoted:    "q",
			Spaces:    "sp",
			EmptyName: "e",
		}
	}
	g.RegisterQuery(ctx, "getNF", getNF)

	query := `query {
		getNF {
			simple
			custom
			comp
			quoted
			spacey
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	require.NoError(t, err)

	assert.Contains(t, result, `"simple":"s"`)
	assert.Contains(t, result, `"custom":"n"`)
	assert.Contains(t, result, `"comp":"c"`)
	assert.Contains(t, result, `"quoted":"q"`)
	assert.Contains(t, result, `"spacey":"sp"`)
}
