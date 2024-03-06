package request_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/request"
)

func TestDecoderFactory_SetDecoderFunc(t *testing.T) {
	df := request.NewDecoderFactory()
	df.SetDecoderFunc("jwt", func(r *http.Request) (url.Values, error) {
		ah := r.Header.Get("Authorization")
		if ah == "" || len(ah) < 8 || strings.ToLower(ah[0:7]) != "bearer " {
			return nil, nil
		}

		var m map[string]json.RawMessage

		err := json.Unmarshal([]byte(ah[7:]), &m)
		if err != nil {
			return nil, err
		}

		res := make(url.Values)

		for k, v := range m {
			if len(v) > 2 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}

			res[k] = []string{string(v)}
		}

		return res, err
	})

	type req struct {
		Q    string `query:"q"`
		Name string `jwt:"name"`
		Iat  int    `jwt:"iat"`
		Sub  string `jwt:"sub"`
	}

	r, err := http.NewRequest(http.MethodGet, "/?q=abc", nil)
	require.NoError(t, err)

	r.Header.Add("Authorization", `Bearer {"sub":"1234567890","name":"John Doe","iat": 1516239022}`)

	d := df.MakeDecoder(http.MethodGet, new(req), nil)

	rr := new(req)
	require.NoError(t, d.Decode(r, rr, nil))

	assert.Equal(t, "John Doe", rr.Name)
	assert.Equal(t, "1234567890", rr.Sub)
	assert.Equal(t, 1516239022, rr.Iat)
	assert.Equal(t, "abc", rr.Q)
}

// BenchmarkDecoderFactory_SetDecoderFunc-4   	  577378	      1994 ns/op	    1024 B/op	      16 allocs/op.
func BenchmarkDecoderFactory_SetDecoderFunc(b *testing.B) {
	df := request.NewDecoderFactory()
	df.SetDecoderFunc("jwt", func(r *http.Request) (url.Values, error) {
		ah := r.Header.Get("Authorization")
		if ah == "" || len(ah) < 8 || strings.ToLower(ah[0:7]) != "bearer " {
			return nil, nil
		}

		// Pretending json.Unmarshal has passed to improve benchmark relevancy.
		m := map[string]json.RawMessage{
			"sub":  []byte(`"1234567890"`),
			"name": []byte(`"John Doe"`),
			"iat":  []byte(`1516239022`),
		}

		res := make(url.Values)

		for k, v := range m {
			if len(v) > 2 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}

			res[k] = []string{string(v)}
		}

		return res, nil
	})

	type req struct {
		Q    string `query:"q"`
		Name string `jwt:"name"`
		Iat  int    `jwt:"iat"`
		Sub  string `jwt:"sub"`
	}

	r, err := http.NewRequest(http.MethodGet, "/?q=abc", nil)
	require.NoError(b, err)

	r.Header.Add("Authorization", `Bearer {"sub":"1234567890","name":"John Doe","iat": 1516239022}`)

	d := df.MakeDecoder(http.MethodGet, new(req), nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rr := new(req)

		err = d.Decode(r, rr, nil)
		if err != nil {
			b.Fail()
		}
	}
}

func TestDecoderFactory_MakeDecoder_default(t *testing.T) {
	type Embed struct {
		Baz bool `query:"baz" default:"true"`
	}

	type DeeplyEmbedded struct {
		Embed
	}

	type MyInput struct {
		ID     int    `query:"id" default:"123"`
		Name   string `header:"X-Name" default:"foo"`
		Deeper struct {
			Foo        string `query:"foo" default:"abc"`
			EvenDeeper struct {
				Bar float64 `query:"bar" default:"1.23"`
			} `query:"even_deeper"`
		} `query:"deeper"`
		*DeeplyEmbedded
		unexported bool `query:"unexported"` // This field is skipped because it is unexported.
	}

	df := request.NewDecoderFactory()
	df.ApplyDefaults = true

	dec := df.MakeDecoder(http.MethodPost, new(MyInput), nil)
	assert.NotNil(t, dec)

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	require.NoError(t, err)

	i := new(MyInput)

	err = dec.Decode(req, i, nil)
	assert.NoError(t, err)
	assert.Equal(t, "foo", i.Name)
	assert.Equal(t, 123, i.ID)
	assert.Equal(t, "abc", i.Deeper.Foo)
	assert.Equal(t, 1.23, i.Deeper.EvenDeeper.Bar)
	assert.Equal(t, true, i.Baz)

	req, err = http.NewRequest(
		http.MethodPost,
		"/?id=321&deeper[foo]=def&deeper[even_deeper][bar]=3.21&baz=false",
		nil,
	)
	require.NoError(t, err)

	req.Header.Set("X-Name", "bar")

	i = new(MyInput)

	err = dec.Decode(req, i, nil)
	assert.NoError(t, err)
	assert.Equal(t, "bar", i.Name)
	assert.Equal(t, 321, i.ID)
	assert.Equal(t, "def", i.Deeper.Foo)
	assert.Equal(t, 3.21, i.Deeper.EvenDeeper.Bar)
	assert.Equal(t, false, i.Baz)
}

func TestDecoderFactory_MakeDecoder_invalidMapping(t *testing.T) {
	assert.PanicsWithValue(t, "non existent fields in mapping: ID2, WrongName", func() {
		type MyInput struct {
			ID   int    `default:"123"`
			Name string `default:"foo"`
		}

		df := request.NewDecoderFactory()

		customMapping := rest.RequestMapping{
			rest.ParamInQuery:  map[string]string{"ID2": "id"},
			rest.ParamInHeader: map[string]string{"WrongName": "X-Name"},
		}

		_ = df.MakeDecoder(http.MethodPost, new(MyInput), customMapping)
	})
}

func TestDecoderFactory_MakeDecoder_customMapping(t *testing.T) {
	type MyInput struct {
		ID   int    `default:"123"`
		Name string `default:"foo"`
	}

	df := request.NewDecoderFactory()
	df.ApplyDefaults = true

	customMapping := rest.RequestMapping{
		rest.ParamInQuery:  map[string]string{"ID": "id"},
		rest.ParamInHeader: map[string]string{"Name": "X-Name"},
	}

	dec := df.MakeDecoder(http.MethodPost, new(MyInput), customMapping)
	assert.NotNil(t, dec)

	req, err := http.NewRequest(http.MethodPost, "/", nil)
	require.NoError(t, err)

	i := new(MyInput)

	err = dec.Decode(req, i, nil)
	assert.NoError(t, err)
	assert.Equal(t, "foo", i.Name)
	assert.Equal(t, 123, i.ID)

	req, err = http.NewRequest(http.MethodPost, "/?id=321", nil)
	require.NoError(t, err)

	req.Header.Set("X-Name", "bar")

	i = new(MyInput)

	err = dec.Decode(req, i, nil)
	assert.NoError(t, err)
	assert.Equal(t, "bar", i.Name)
	assert.Equal(t, 321, i.ID)
}

func TestDecoderFactory_MakeDecoder_header_case_sensitivity(t *testing.T) {
	df := request.NewDecoderFactory()

	type input struct {
		A string `header:"x-one-two-three" required:"true"`
		B string `header:"X-One-Two-Three"`
		C string `header:"X-One-two-three"`
		D string `header:"x-one-two-three"`
	}

	d := df.MakeDecoder(http.MethodGet, input{}, nil)

	var v input

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("x-One-Two-threE", "hello!")

	require.NoError(t, d.Decode(req, &v, rest.ValidatorFunc(func(_ rest.ParamIn, namedData map[string]interface{}) error {
		fmt.Printf("%+v", namedData)

		return nil
	})))
	assert.Equal(t, "hello!", v.A)
	assert.Equal(t, "hello!", v.B)
	assert.Equal(t, "hello!", v.C)
	assert.Equal(t, "hello!", v.D)
}

type defaultFromSchema string

func (d *defaultFromSchema) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithDefault(enum1)
	schema.WithTitle("Value with default from schema")

	return nil
}

type defaultFromSchemaVal string

func (d defaultFromSchemaVal) PrepareJSONSchema(schema *jsonschema.Schema) error {
	schema.WithDefault(enum1)
	schema.WithTitle("Value with default from schema")

	return nil
}

const (
	enum1 = "all"
	enum2 = "none"
)

func (d *defaultFromSchema) Enum() []interface{} {
	return []interface{}{enum1, enum2}
}

func (d defaultFromSchemaVal) Enum() []interface{} {
	return []interface{}{enum1, enum2}
}

func TestNewDecoderFactory_default(t *testing.T) {
	type NewThing struct {
		DefaultedQuery    *defaultFromSchema    `query:"dq"`
		DefaultedPtr      *defaultFromSchema    `json:"dp,omitempty"`
		Defaulted         defaultFromSchema     `json:"d"`
		DefaultedTag      defaultFromSchema     `query:"dt" default:"none"`
		DefaultedQueryVal *defaultFromSchemaVal `query:"dqv"`
		DefaultedPtrVal   *defaultFromSchemaVal `json:"dpv,omitempty"`
		DefaultedVal      defaultFromSchemaVal  `json:"dv"`
		DefaultedTagVal   defaultFromSchemaVal  `query:"dtv" default:"none"`
	}

	df := request.NewDecoderFactory()
	df.ApplyDefaults = true
	df.JSONSchemaReflector = &jsonschema.Reflector{}

	var input NewThing
	dec := df.MakeDecoder(http.MethodPost, input, nil)

	req, err := http.NewRequest(http.MethodPost, "/foo", bytes.NewReader([]byte(`{}`)))
	require.NoError(t, err)

	require.NoError(t, dec.Decode(req, &input, nil))
	assert.Equal(t, enum1, string(*input.DefaultedPtr))
	assert.Equal(t, enum1, string(input.Defaulted))
	assert.Equal(t, enum1, string(*input.DefaultedQuery))
	assert.Equal(t, enum2, string(input.DefaultedTag))

	assert.Equal(t, enum1, string(*input.DefaultedPtrVal))
	assert.Equal(t, enum1, string(input.DefaultedVal))
	assert.Equal(t, enum1, string(*input.DefaultedQueryVal))
	assert.Equal(t, enum2, string(input.DefaultedTagVal))
}
