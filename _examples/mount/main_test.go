package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
)

func Test_service(t *testing.T) {
	s := service()

	rw := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/api/docs/openapi.json", nil)
	require.NoError(t, err)

	s.ServeHTTP(rw, r)
	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3","info":{"title":"Security and Mount Example","version":""},
	  "paths":{
		"/api/v1/mul":{
		  "post":{
			"tags":["V1"],"summary":"Mul","operationId":"_examples/mount.mul",
			"requestBody":{
			  "content":{
				"application/json":{"schema":{"type":"array","items":{"type":"integer"}}}
			  }
			},
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"application/json":{"schema":{"type":"integer"}}}
			  },
			  "401":{
				"description":"Unauthorized",
				"content":{
				  "application/json":{"schema":{"$ref":"#/components/schemas/RestErrResponse"}}
				}
			  }
			},
			"security":[{"Admin":[]}]
		  }
		},
		"/api/v1/sum":{
		  "post":{
			"tags":["V1"],"summary":"Sum","operationId":"_examples/mount.sum",
			"requestBody":{
			  "content":{
				"application/json":{"schema":{"type":"array","items":{"type":"integer"}}}
			  }
			},
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"application/json":{"schema":{"type":"integer"}}}
			  },
			  "401":{
				"description":"Unauthorized",
				"content":{
				  "application/json":{"schema":{"$ref":"#/components/schemas/RestErrResponse"}}
				}
			  }
			},
			"security":[{"Admin":[]}]
		  }
		},
		"/api/v2/multiplication":{
		  "post":{
			"tags":["V2"],"summary":"Mul","operationId":"_examples/mount.mul2",
			"requestBody":{
			  "content":{
				"application/json":{"schema":{"type":"array","items":{"type":"integer"}}}
			  }
			},
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"application/json":{"schema":{"type":"integer"}}}
			  }
			}
		  }
		},
		"/api/v2/summarization":{
		  "post":{
			"tags":["V2"],"summary":"Sum","operationId":"_examples/mount.sum2",
			"requestBody":{
			  "content":{
				"application/json":{"schema":{"type":"array","items":{"type":"integer"}}}
			  }
			},
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"application/json":{"schema":{"type":"integer"}}}
			  }
			}
		  }
		}
	  },
	  "components":{
		"schemas":{
		  "RestErrResponse":{
			"type":"object",
			"properties":{
			  "code":{"type":"integer","description":"Application-specific error code."},
			  "context":{
				"type":"object","additionalProperties":{},
				"description":"Application context."
			  },
			  "error":{"type":"string","description":"Error message."},
			  "status":{"type":"string","description":"Status text."}
			}
		  }
		},
		"securitySchemes":{"Admin":{"type":"http","scheme":"basic","description":"Admin access"}}
	  }
	}`, json.RawMessage(rw.Body.Bytes()))

	rw = httptest.NewRecorder()
	r, err = http.NewRequest(http.MethodPost, "/api/v2/multiplication", bytes.NewReader([]byte(`[1,2,3]`)))

	s.ServeHTTP(rw, r)
	assert.Equal(t, "6\n", rw.Body.String())
}
