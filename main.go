package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/pedro-git-projects/gymtracker/store"
	"github.com/pedro-git-projects/gymtracker/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	dataPath := flag.String("data", "gymtracker.json", "path to the JSON data file")
	flag.Parse()

	st, err := store.Open(*dataPath)
	if err != nil {
		log.Fatal(err)
	}
	srv, err := web.New(st)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("gymtracker listening on http://%s (data: %s)", *addr, *dataPath)
	log.Fatal(http.ListenAndServe(*addr, srv))
}
