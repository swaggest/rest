// Originally from https://github.com/go-chi/jwtauth/blob/v5.1.1/_example/main.go.
//
// jwtauth example
//
// Sample output:
//
// [peter@pak ~]$ curl -v http://localhost:3333/
// *   Trying ::1...
// * Connected to localhost (::1) port 3333 (#0)
// > GET / HTTP/1.1
// > Host: localhost:3333
// > User-Agent: curl/7.49.1
// > Accept: */*
// >
// < HTTP/1.1 200 OK
// < Date: Tue, 13 Sep 2016 15:53:17 GMT
// < Content-Length: 17
// < Content-Type: text/plain; charset=utf-8
// <
// * Connection #0 to host localhost left intact
// welcome anonymous%
//
//
// [peter@pak ~]$ curl -v http://localhost:3333/admin
// *   Trying ::1...
// * Connected to localhost (::1) port 3333 (#0)
// > GET /admin HTTP/1.1
// > Host: localhost:3333
// > User-Agent: curl/7.49.1
// > Accept: */*
// >
// < HTTP/1.1 401 Unauthorized
// < Content-Type: text/plain; charset=utf-8
// < X-Content-Type-Options: nosniff
// < Date: Tue, 13 Sep 2016 15:53:19 GMT
// < Content-Length: 13
// <
// Unauthorized
// * Connection #0 to host localhost left intact
//
//
// [peter@pak ~]$ curl -H"Authorization: BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjN9.PZLMJBT9OIVG2qgp9hQr685oVYFgRgWpcSPmNcw6y7M" -v http://localhost:3333/admin
// *   Trying ::1...
// * Connected to localhost (::1) port 3333 (#0)
// > GET /admin HTTP/1.1
// > Host: localhost:3333
// > User-Agent: curl/7.49.1
// > Accept: */*
// > Authorization: BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjN9.PZLMJBT9OIVG2qgp9hQr685oVYFgRgWpcSPmNcw6y7M
// >
// < HTTP/1.1 200 OK
// < Date: Tue, 13 Sep 2016 15:54:26 GMT
// < Content-Length: 22
// < Content-Type: text/plain; charset=utf-8
// <
// * Connection #0 to host localhost left intact
// protected area. hi 123%
//

package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	jwtauth "github.com/go-chi/jwtauth/v5"
	"github.com/swaggest/openapi-go/openapi31"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
)

var tokenAuth *jwtauth.JWTAuth

func init() {
	tokenAuth = jwtauth.New("HS256", []byte("secret"), nil)

	// For debugging/example purposes, we generate and print
	// a sample jwt token with claims `user_id:123` here:
	_, tokenString, _ := tokenAuth.Encode(map[string]interface{}{"user_id": 123})
	fmt.Printf("DEBUG: a sample jwt is %s\n\n", tokenString)
}

func main() {
	addr := "localhost:3333"
	fmt.Printf("Starting server on http://%v\n", addr)
	http.ListenAndServe(addr, router())
}

func Get() usecase.Interactor {
	u := usecase.NewInteractor(
		func(ctx context.Context, _ struct{}, output *map[string]interface{}) error {
			jwt, claims, err := jwtauth.FromContext(ctx)
			fmt.Println(err)
			fmt.Println(jwt)
			fmt.Println(claims)

			*output = claims

			return nil
		})
	return u
}

func router() http.Handler {
	s := web.NewService(openapi31.NewReflector())

	s.Route("/admin", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(
				jwtauth.Verifier(tokenAuth),
				jwtauth.Authenticator,
			)

			r.Method(http.MethodGet, "/", nethttp.NewHandler(Get()))
		})
	})

	// Public routes
	s.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("welcome anonymous"))
		})
	})

	return s
}
