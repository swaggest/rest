package rest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swaggest/rest"
)

func TestHandlerTrait_RestHandler(t *testing.T) {
	h := &rest.HandlerTrait{
		ReqMapping: map[rest.ParamIn]map[string]string{},
	}
	assert.Equal(t, h, h.RestHandler())
	assert.Equal(t, h.ReqMapping, h.RequestMapping())
}

func TestOutputHasNoContent(t *testing.T) {
	assert.True(t, rest.OutputHasNoContent(nil))
	assert.True(t, rest.OutputHasNoContent(struct{}{}))
}
