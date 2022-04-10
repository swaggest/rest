//go:build go1.18

package main

import (
	"log"
	"net/http"
)

func main() {
	log.Println("http://localhost:8011/docs")
	if err := http.ListenAndServe(":8011", NewRouter()); err != nil {
		log.Fatal(err)
	}
}
