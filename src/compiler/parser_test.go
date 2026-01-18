package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseLabel(t *testing.T) {
	code := "publicLabel:"
	tokens := RunTokenizer(code)
	node, err := Parse(tokens[:2])
	assert.Nil(t, err)

	label, ok := node.(Label)
	assert.True(t, ok)
	assert.Equal(t, "publicLabel", label.Label())
	assert.True(t, label.IsPublic())
}
