package gorillamux_test

import (
	"github.com/swaggest/rest/gorillamux"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/usecase"
)

type structuredHandler struct {
	usecase.Info
	usecase.WithInput
	usecase.WithOutput
}

func (s structuredHandler) SetupOpenAPIOperation(oc openapi.OperationContext) error {
	oc.AddReqStructure(s.Input)
	oc.AddRespStructure(s.Output)

	return nil
}

func newStructuredHandler(setup func(h *structuredHandler)) structuredHandler {
	h := structuredHandler{}
	setup(&h)

	return h
}

func (s structuredHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

func TestNewWrapper(t *testing.T) {
	r := mux.NewRouter()

	r.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/products", nil).Methods(http.MethodGet)
	r.HandleFunc("/articles", nil).Methods(http.MethodGet)
	r.Handle("/products/{key}",
		newStructuredHandler(func(h *structuredHandler) {
			h.Input = struct {
				Key string `path:"key"`
			}{}
			h.Output = struct {
			}{}
		})).
		Methods(http.MethodGet).
		Queries("key", "value")
	s := r.Host("www.example.com").Subrouter()
	r.Handle("/articles/{category}/",
		newStructuredHandler(func(h *structuredHandler) {
			h.Input = struct {
				Filter   string `query:"filter"`
				Category string `path:"category"`
			}{}
		})).
		Methods(http.MethodGet).
		Host("{subdomain:[a-z]+}.example.com")
	s.Handle("/articles/{category}/{id:[0-9]+}", newStructuredHandler(func(h *structuredHandler) {
		h.Input = struct {
			Filter   string `query:"filter"`
			Category string `path:"category"`
			ID       string `path:"id"`
		}{}
	})).
		Methods(http.MethodGet).
		Headers("X-Requested-With", "XMLHttpRequest")

	r.HandleFunc("/specific", nil).Methods(http.MethodPost)
	r.PathPrefix("/").Handler(nil)

	http.Handle("/", r)

	c := gorillamux.NewOpenAPICollector()

	assert.NoError(t, r.Walk(c.Walker))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3",
	  "info":{
		"title":"Test Server","description":"Provides API over HTTP",
		"version":"v1.2.3"
	  },
	  "paths":{
		"/":{
		  "delete":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  },
		  "get":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  },
		  "head":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  },
		  "patch":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  },
		  "post":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  },
		  "put":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  }
		},
		"/articles":{
		  "get":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  }
		},
		"/articles/{category}/":{
		  "get":{
			"parameters":[
			  {"name":"filter","in":"query","schema":{"type":"string"}},
			  {
				"name":"category","in":"path","required":true,
				"schema":{"type":"string"}
			  }
			],
			"responses":{"200":{"description":"OK"}}
		  }
		},
		"/articles/{category}/{id}":{
		  "get":{
			"parameters":[
			  {"name":"filter","in":"query","schema":{"type":"string"}},
			  {
				"name":"category","in":"path","required":true,
				"schema":{"type":"string"}
			  },
			  {"name":"id","in":"path","required":true,"schema":{"type":"string"}}
			],
			"responses":{"200":{"description":"OK"}}
		  }
		},
		"/products":{
		  "get":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  }
		},
		"/products/{key}":{
		  "get":{
			"parameters":[
			  {
				"name":"key","in":"path","required":true,"schema":{"type":"string"}
			  }
			],
			"responses":{"200":{"description":"OK"}}
		  }
		},
		"/specific":{
		  "post":{
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  }
		}
	  }
	}`, c.Collector.Reflector().Spec)
}
