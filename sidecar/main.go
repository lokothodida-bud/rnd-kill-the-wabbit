package main

import (
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"github.com/thisisbud/backend-events-sidecar/internal/handlers"
	"github.com/thisisbud/backend-events-sidecar/internal/storage/memory"
	"log"
	"net/http"
	"os"
)

func main() {
	repo := memory.NewEventRepository()
	port := flag.String("port", os.Getenv("HTTP_PORT"), "HTTP port to run on")

	flag.Parse()

	r := chi.NewRouter()
	r.Get("/", handlers.Wellknown)
	r.Get("/v1/events", handlers.GetLatestEvent(repo.GetLatestEvent))
	r.Get("/v1/events/{event_id}", handlers.GetEvent(repo.GetEvent))
	r.Post("/v1/events", handlers.PublishEvent(repo.Publish, uuid.NewString))

	log.Printf("running on port %s\n", *port)
	log.Panic(http.ListenAndServe(":"+*port, r))
}
