package gorillamux_test

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/swaggest/assertjson"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/gorillamux"
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

func (s structuredHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}

func TestOpenAPICollector_Walker(t *testing.T) {
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
			h.Output = struct{}{}
		})).
		Methods(http.MethodGet).
		Queries("key", "value")
	r.Handle("/articles/{category}/",
		newStructuredHandler(func(h *structuredHandler) {
			h.Input = struct {
				Filter   string `query:"filter"`
				Category string `path:"category"`
			}{}
		})).
		Methods(http.MethodGet).
		Host("{subdomain:[a-z]+}.example.com")

	s := r.Host("www.example.com").Subrouter()

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

	rf := openapi3.NewReflector()
	rf.Spec.Info.
		WithTitle("Test Server").
		WithVersion("v1.2.3").
		WithDescription("Provides API over HTTP")

	c := gorillamux.NewOpenAPICollector(rf)

	assert.NoError(t, r.Walk(c.Walker))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3",
	  "info":{
		"title":"Test Server","description":"Provides API over HTTP",
		"version":"v1.2.3"
	  },
	  "paths":{
		"/articles":{
		  "get":{
			"tags":["Incomplete"],
			"description":"Information about this operation was obtained using only HTTP method and path pattern. It may be incomplete and/or inaccurate.",
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  }
		},
		"/products":{
		  "get":{
			"tags":["Incomplete"],
			"description":"Information about this operation was obtained using only HTTP method and path pattern. It may be incomplete and/or inaccurate.",
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
			"tags":["Incomplete"],
			"description":"Information about this operation was obtained using only HTTP method and path pattern. It may be incomplete and/or inaccurate.",
			"responses":{
			  "200":{
				"description":"OK",
				"content":{"text/html":{"schema":{"type":"string"}}}
			  }
			}
		  }
		}
	  }
	}`, rf.Spec)

	rf = openapi3.NewReflector()
	rf.Spec.Info.
		WithTitle("Test Server (www.example.com)").
		WithVersion("v1.2.3").
		WithDescription("Provides API over HTTP")
	rf.Spec.WithServers(openapi3.Server{
		URL: "www.example.com",
	})

	c = gorillamux.NewOpenAPICollector(rf)
	c.Host = "www.example.com"

	assert.NoError(t, r.Walk(c.Walker))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3",
	  "info":{
		"title":"Test Server (www.example.com)",
		"description":"Provides API over HTTP","version":"v1.2.3"
	  },
	  "servers":[{"url":"www.example.com"}],
	  "paths":{
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
		}
	  }
	}`, rf.Spec)

	rf = openapi3.NewReflector()
	rf.Spec.Info.
		WithTitle("Test Server ({subdomain}.example.com)").
		WithVersion("v1.2.3").
		WithDescription("Provides API over HTTP")
	rf.Spec.WithServers(openapi3.Server{
		URL: "{subdomain}.example.com",
		Variables: map[string]openapi3.ServerVariable{
			"subdomain": {
				Default: "foo",
			},
		},
	})

	c = gorillamux.NewOpenAPICollector(rf)
	c.Host = "{subdomain:[a-z]+}.example.com"

	assert.NoError(t, r.Walk(c.Walker))

	assertjson.EqMarshal(t, `{
	  "openapi":"3.0.3",
	  "info":{
		"title":"Test Server ({subdomain}.example.com)",
		"description":"Provides API over HTTP","version":"v1.2.3"
	  },
	  "servers":[
		{
		  "url":"{subdomain}.example.com",
		  "variables":{"subdomain":{"default":"foo"}}
		}
	  ],
	  "paths":{
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
		}
	  }
	}`, rf.Spec)
}
