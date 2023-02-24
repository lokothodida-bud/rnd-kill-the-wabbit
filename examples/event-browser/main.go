package main

import (
	"embed"
	"flag"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

// Provides a simple interface for browsing a particular event stream in
// the web browser (assuming the application/vnd.bud.events+json content type)
// static server code copied from https://github.com/go-chi/chi/issues/611#issuecomment-1251085383

// go:embed examples/event-browser/assets
var embeddedFS embed.FS

func main() {
	port := flag.String("port", "8080", "http port for server")
	directory := flag.String("directory", "./examples/event-browser/assets", "directory to load html assets from")
	flag.Parse()
	r := chi.NewRouter()
	r.Handle("/*", http.FileServer(http.Dir(*directory)))

	log.Printf("listening on port %s", *port)
	log.Panic(http.ListenAndServe(":"+*port, r))
}
