package main

import (
	"log"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

func main() {
	log.Println("http://localhost:8011/docs")
	if err := fasthttp.ListenAndServe(":8011", fchi.RequestHandler(NewRouter())); err != nil {
		log.Fatal(err)
	}
}
