package main

import (
	"github.com/go-chi/chi/v5"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"net/http"
	"os"
)

func main() {
	r := chi.NewRouter()
	log.Panic(http.ListenAndServe(":"+os.Getenv("HTTP_PORT"), r))
}
