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
	eventServerURL := flag.String("server-url", "http://localhost:8080/events", "origin server to poll for events")
	tickerTimeString := flag.String("ticker", "3s", "time.Duration string on time between ticks, e.g. 5s")

	flag.Parse()

	duration, err := time.ParseDuration(*tickerTimeString)

	if err != nil {
		log.Panic(err)
	}

	log.Printf("polling event server [%s]\n", *eventServerURL)

	var lastEventID string

	for range time.Tick(duration) {
		events, err := findLatestEvents(*eventServerURL, lastEventID)

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

func findLatestEvents(wellknownURL string, latestEventID string) ([]domain.Event, error) {
	resp, err := queryForEvent(wellknownURL)

	if err != nil {
		return nil, err
	}

	if resp.Data.EventID == latestEventID {
		return []domain.Event{}, nil
	}

	events := []domain.Event{resp.Data}

	currentEventID := resp.Data.EventID

	for currentEventID != latestEventID && resp.Metadata["next"].Href != "" {
		resp, err = queryForEvent(resp.Metadata["next"].Href)

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

	if resp.StatusCode == http.StatusFound {
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
	fmt.Printf("[%s] [%s] %s\n", event.OccurredAt, event.EventID, event.EventName)
}

var ErrEventNotFound = errors.New("event not found")
