package rest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest"
)

func TestRequestErrors_Error(t *testing.T) {
	err := rest.RequestErrors{
		"foo": []string{"bar"},
	}

	assert.EqualError(t, err, "bad request")
	assert.Equal(t, map[string]any{"foo": []string{"bar"}}, err.Fields())
}
