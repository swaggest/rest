//go:build go1.18

package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("http://localhost:8012/docs")
	if err := http.ListenAndServe("localhost:8012", NewRouter()); err != nil {
		log.Fatal(err)
	}
}
