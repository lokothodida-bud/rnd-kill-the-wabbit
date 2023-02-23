package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/lokothodida/rnd-kill-the-wabbit/internal/domain"
	"log"
	"net/http"
	"sort"
	"time"
)

// This example consumes events periodically from a particular stream
// and simply prints the output in order to the log
func main() {
	eventServerURL := flag.String("server-url", "http://localhost:8080", "origin server to poll for events")
	eventServerWellKnownPath := flag.String("wellknown-path", "/events", "well known path for latest event, e.g. /events")
	tickerTimeString := flag.String("ticker", "3s", "time.Duration string on time between ticks, e.g. 5s")
	lastEventIDFlag := flag.String("last-event-id", "", "last ID of the event stream to start from")

	flag.Parse()

	duration, err := time.ParseDuration(*tickerTimeString)

	if err != nil {
		log.Panic(err)
	}

	log.Printf("polling event server [%s]\n", *eventServerURL)

	lastEventID := *lastEventIDFlag

	for range time.Tick(duration) {
		events, err := findLatestEvents(*eventServerURL, *eventServerWellKnownPath, lastEventID)

		if err != nil {
			log.Panic(err)
		}

		for i, event := range events {
			printEvent(event)

			if i == len(events)-1 {
				lastEventID = event.EventID
			}
		}
	}
}

func findLatestEvents(baseURL string, wellknownURL string, latestEventID string) ([]domain.Event, error) {
	resp, err := queryForEvent(baseURL + wellknownURL)

	if errors.Is(err, ErrEventNotFound) {
		return []domain.Event{}, nil
	}

	if err != nil {
		return nil, err
	}

	if resp.Data.EventID == latestEventID {
		return []domain.Event{}, nil
	}

	events := []domain.Event{resp.Data}

	currentEventID := resp.Data.EventID

	for currentEventID != latestEventID && resp.Metadata["next"].Href != "" {
		resp, err = queryForEvent(baseURL + resp.Metadata["next"].Href)

		if err != nil {
			return nil, err
		}
		currentEventID = resp.Data.EventID

		if currentEventID != latestEventID {
			events = append(events, resp.Data)
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return j < i
	})

	return events, nil
}

func queryForEvent(eventURL string) (*domain.Response, error) {
	resp, err := http.Get(eventURL)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrEventNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code [%d]", resp.StatusCode)
	}

	var body domain.Response

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	return &body, nil
}

func printEvent(event domain.Event) {
	fmt.Printf("[%s] [%s] %s %s\n", event.OccurredAt, event.EventID, event.EventName, string(event.Payload))
}

var ErrEventNotFound = errors.New("event not found")
