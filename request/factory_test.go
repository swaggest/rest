package request_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	type MyInput struct {
		ID   int    `query:"id" default:"123"`
		Name string `header:"X-Name" default:"foo"`
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

	req, err = http.NewRequest(http.MethodPost, "/?id=321", nil)
	require.NoError(t, err)

	req.Header.Set("X-Name", "bar")

	i = new(MyInput)

	err = dec.Decode(req, i, nil)
	assert.NoError(t, err)
	assert.Equal(t, "bar", i.Name)
	assert.Equal(t, 321, i.ID)
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
