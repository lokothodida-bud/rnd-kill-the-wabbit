package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/lokothodida/rnd-kill-the-wabbit/internal/domain"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

func main() {
	port := flag.String("port", "8080", "http port for service")
	maxEventsFlag := flag.Uint("max-events", 10, "maximum number of events to generate")
	newEventsIntervalStr := flag.String("new-events-interval", "0s", "interval to randomly generate new events at")

	flag.Parse()

	newEventsInterval, err := time.ParseDuration(*newEventsIntervalStr)

	if err != nil {
		log.Panic(err)
	}

	maxEvents := int(*maxEventsFlag)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	events := generateEvents(maxEvents)
	baseURL := "http://localhost:" + *port
	var mu sync.Mutex

	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		renderEvents(w, events, baseURL)
		mu.Unlock()
	})

	r.Get("/events/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		eventID := chi.URLParam(r, "event_id")
		mu.Lock()
		eventAndNext, err := findEvent(eventID, events)
		mu.Unlock()

		if err != nil {
			w.Header().Set("Content-Type", domain.ContentType)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(domain.Response{
				Metadata: map[string]domain.Reference{
					"latest": latestEventReference(baseURL),
				},
			})

			return
		}

		renderEvents(w, eventAndNext, baseURL)
	})

	go func() {
		if newEventsInterval == 0 {
			return
		}

		for range time.Tick(newEventsInterval) {
			mu.Lock()
			newEvents := generateEvents(maxEvents)
			log.Printf("generated %d new events", len(newEvents))
			events = append(newEvents, events...)
			mu.Unlock()
		}
	}()

	log.Printf("%d events generated", len(events))
	log.Printf("running server on port %s\n", *port)
	log.Panic(http.ListenAndServe(fmt.Sprintf(":%s", *port), r))
}

func generateEvents(maxEvents int) []domain.Event {
	var events []domain.Event
	rand.Seed(time.Now().Unix())

	for i := 0; i < rand.Intn(maxEvents); i++ {
		events = append(events, domain.Event{
			EventID:    uuid.NewString(),
			EventName:  "some_fake_event",
			OccurredAt: time.Now().Add(time.Minute * time.Duration(-i)),
		})
	}

	return events
}

func findEvent(eventID string, events []domain.Event) ([]domain.Event, error) {
	var ret []domain.Event
	for i, e := range events {
		if e.EventID == eventID {
			ret = append(ret, e)

			if i < len(events)-1 {
				ret = append(ret, events[i+1])
			}

			return ret, nil
		}
	}

	return nil, errors.New("event not found")
}

func renderEvents(w http.ResponseWriter, events []domain.Event, baseURL string) {
	w.Header().Set("Content-Type", domain.ContentType)
	resp := domain.Response{
		Metadata: map[string]domain.Reference{
			"latest": latestEventReference(baseURL),
		},
	}

	if len(events) > 0 {
		resp.Data = events[0]
	}

	if len(events) > 1 {
		resp.Metadata["next"] = domain.Reference{
			Href: fmt.Sprintf("%s/events/%s", baseURL, events[1].EventID),
			Type: http.MethodGet,
		}
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func latestEventReference(baseURL string) domain.Reference {
	return domain.Reference{
		Href: fmt.Sprintf("%s/events", baseURL),
		Type: http.MethodGet,
	}
}
