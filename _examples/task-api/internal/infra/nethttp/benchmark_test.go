package nethttp_test

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bool64/httptestbench"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/rest/_examples/task-api/internal/domain/task"
	"github.com/swaggest/rest/_examples/task-api/internal/infra"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/nethttp"
	"github.com/swaggest/rest/_examples/task-api/internal/infra/service"
	"github.com/valyala/fasthttp"
)

// Benchmark_notFoundSrv-4   	   31236	     37106 ns/op	     26927 RPS	    8408 B/op	      76 allocs/op.
// Benchmark_notFoundSrv-4   	   33241	     33620 ns/op	     29745 RPS	    5796 B/op	      65 allocs/op.
// Benchmark_notFoundSrv-4   	   33656	     35653 ns/op	       336 B:rcvd/op	        74.0 B:sent/op	     28048 rps	    5813 B/op	      65 allocs/op.
// Benchmark_notFoundSrv-4   	   32262	     36431 ns/op	       337 B:rcvd/op	        74.0 B:sent/op	     27449 rps	    5769 B/op	      64 allocs/op.
func Benchmark_notFoundSrv(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	l := infra.NewServiceLocator(service.Config{})
	defer l.Close()

	srv := httptest.NewServer(nethttp.NewRouter(l))
	defer srv.Close()

	httptestbench.RoundTrip(b, 50,
		func(i int, req *fasthttp.Request) {
			req.SetRequestURI(srv.URL + "/dev/tasks/1")
		},
		func(i int, resp *fasthttp.Response) bool {
			return resp.StatusCode() == http.StatusNotFound
		},
	)
}

// Benchmark_ok-4            	   28002	     36993 ns/op	     27027 RPS	    8539 B/op	      75 allocs/op.
// Benchmark_ok-4   	   35078	     34293 ns/op	     29156 RPS	    5729 B/op	      61 allocs/op.
// Benchmark_ok-4   	   33270	     36366 ns/op	       360 B:rcvd/op	        74.0 B:sent/op	     27498 rps	    5730 B/op	      61 allocs/op.
// Benchmark_ok-4   	   32761	     37317 ns/op	       362 B:rcvd/op	        74.0 B:sent/op	     26797 rps	    5673 B/op	      60 allocs/op.
func Benchmark_ok(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	l := infra.NewServiceLocator(service.Config{})
	defer l.Close()

	srv := httptest.NewServer(nethttp.NewRouter(l))
	defer srv.Close()

	_, err := l.TaskCreator().Create(context.Background(), task.Value{Goal: "victory!"})
	require.NoError(b, err)

	httptestbench.RoundTrip(b, 50,
		func(i int, req *fasthttp.Request) {
			req.SetRequestURI(srv.URL + "/dev/tasks/1")
		},
		func(i int, resp *fasthttp.Response) bool {
			return resp.StatusCode() == http.StatusOK
		},
	)
}

// Benchmark_invalidBody-4   	   23670	     46677 ns/op	     21424 RPS	   13111 B/op	     132 allocs/op.
// Benchmark_invalidBody-4   	   23838	     46156 ns/op	     21666 RPS	    9724 B/op	     111 allocs/op.
// Benchmark_invalidBody-4   	   23589	     60475 ns/op	       439 B:rcvd/op	       137 B:sent/op	     16531 rps	    9781 B/op	     111 allocs/op.
// Benchmark_invalidBody-4   	   18458	     54945 ns/op	       435 B:rcvd/op	       137 B:sent/op	     18200 rps	    9634 B/op	     110 allocs/op
func Benchmark_invalidBody(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	l := infra.NewServiceLocator(service.Config{})
	defer l.Close()

	r := nethttp.NewRouter(l)
	srv := httptest.NewServer(r)

	tt, err := l.TaskCreator().Create(context.Background(), task.Value{Goal: "win"})
	require.NoError(b, err)
	assert.Equal(b, 1, tt.ID)

	body := []byte(`{"goal":""}`)

	httptestbench.RoundTrip(b, 50,
		func(i int, req *fasthttp.Request) {
			req.Header.SetMethod(http.MethodPut)
			req.Header.SetContentType("application/json")
			req.SetRequestURI(srv.URL + "/dev/tasks/1")
			req.SetBody(body)
		},
		func(i int, resp *fasthttp.Response) bool {
			return resp.StatusCode() == http.StatusBadRequest
		},
	)
}
