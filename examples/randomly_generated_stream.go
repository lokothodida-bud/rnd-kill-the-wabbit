package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/thisisbud/backend-events-sidecar/pkg/budevents"
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
	r.Use(cors.AllowAll().Handler)
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
			w.Header().Set("Content-Type", budevents.ContentType)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(budevents.Response{
				Metadata: map[string]budevents.Reference{
					"latest": latestEventReference(),
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

func generateEvents(maxEvents int) []budevents.Event {
	var events []budevents.Event
	rand.Seed(time.Now().Unix())

	for i := 0; i < rand.Intn(maxEvents); i++ {
		events = append(events, budevents.Event{
			EventID:    uuid.NewString(),
			EventName:  "some_fake_event",
			OccurredAt: time.Now(),
		})
	}

	return events
}

func findEvent(eventID string, events []budevents.Event) ([]budevents.Event, error) {
	var ret []budevents.Event
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

func renderEvents(w http.ResponseWriter, events []budevents.Event, baseURL string) {
	w.Header().Set("Content-Type", budevents.ContentType)
	resp := budevents.Response{
		Metadata: map[string]budevents.Reference{
			"latest": latestEventReference(),
		},
	}

	if len(events) > 0 {
		resp.Data = events[0]
	}

	if len(events) > 1 {
		resp.Metadata["next"] = budevents.Reference{
			Href: fmt.Sprintf("/events/%s", events[1].EventID),
			Type: http.MethodGet,
		}
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func latestEventReference() budevents.Reference {
	return budevents.Reference{
		Href: "/events",
		Type: http.MethodGet,
	}
}
