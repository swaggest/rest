package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
)

func Test_queryObject(t *testing.T) {
	r := NewRouter()

	srv := httptest.NewServer(r)
	defer srv.Close()

	for _, tc := range []struct {
		name string
		url  string
		code int
		resp string
	}{
		{
			name: "validation_failed_deep_object",
			url:  `/query-object?in_query[1]=0&in_query[2]=0&in_query[3]=0&json_filter={"foo":"strin"}&deep_object_filter[bar]=sd`,
			code: http.StatusBadRequest,
			resp: `{
			  "msg":"invalid argument: validation failed",
			  "details":{"query:deep_object_filter":["#/bar: length must be \u003e= 3, but got 2"]}
			}`,
		},
		{
			name: "validation_failed_deep_object_2",
			url:  `/query-object?in_query[1]=0&in_query[2]=0&in_query[3]=0&json_filter={"foo":"strin"}&deep_object_filter[bar]=asd&deep_object_filter[baz]=sd`,
			code: http.StatusBadRequest,
			resp: `{
			  "msg":"invalid argument: validation failed",
			  "details":{"query:deep_object_filter":["#/baz: length must be \u003e= 3, but got 2"]}
			}`,
		},
		{
			name: "validation_failed_json",
			url:  `/query-object?in_query[1]=0&in_query[2]=0&in_query[3]=0&json_filter={"foo":"string"}&deep_object_filter[bar]=asd`,
			code: http.StatusBadRequest,
			resp: `{
			  "msg":"invalid argument: validation failed",
			  "details":{"query:json_filter":["#/foo: length must be \u003c= 5, but got 6"]}
			}`,
		},
		{
			name: "ok",
			url:  `/query-object?in_query[1]=0&in_query[2]=0&in_query[3]=0&json_map={"123":123.45}&json_filter={"foo":"strin"}&deep_object_filter[bar]=asd`,
			code: http.StatusOK,
			resp: `{
			  "inQuery":{"1":0,"2":0,"3":0},"jsonMap":{"123":123.45},"jsonFilter":{"foo":"strin"},
			  "deepObjectFilter":{"bar":"asd"}
			}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(
				http.MethodGet,
				srv.URL+tc.url,
				nil,
			)
			require.NoError(t, err)

			resp, err := http.DefaultTransport.RoundTrip(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.NoError(t, resp.Body.Close())
			assertjson.EqMarshal(t, tc.resp, json.RawMessage(body))
			assert.Equal(t, tc.code, resp.StatusCode)
		})
	}
}
