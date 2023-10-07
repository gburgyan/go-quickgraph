package quickgraph

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRequest_Error(t *testing.T) {
	_, err := ParseRequest("invalid")
	assert.Error(t, err)
	jsonError, _ := json.Marshal(err)
	assert.Equal(t, `{"message":"error parsing request: 1:8: sub-expression (\"{\" Command+ \"}\")+ must match at least once","locations":[{"line":1,"column":8}]}`, string(jsonError))
}
