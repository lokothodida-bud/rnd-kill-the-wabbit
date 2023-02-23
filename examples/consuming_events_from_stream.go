package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/lokothodida/rnd-kill-the-wabbit/internal/domain"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

// This example consumes events periodically from a particular stream
// and simply prints the output in order to the log
func main() {
	configFilename := flag.String("config-file", "./stream_config.dist.json", "JSON config file for consumers")

	flag.Parse()

	conf, err := loadConfig(*configFilename)

	if err != nil {
		log.Panic(err)
	}

	errs, _ := errgroup.WithContext(context.Background())

	for _, con := range conf {
		errs.Go(func(conf config) func() error {
			return func() error {
				return consumeEvents(conf)
			}
		}(con))
	}

	log.Panic(errs.Wait())
}

func consumeEvents(conf config) error {
	lastEventID := conf.LastEventID

	log.Printf("polling event server [%s]\n", conf.BaseURL)

	for range time.Tick(time.Duration(conf.Ticker)) {
		events, err := findLatestEvents(conf.BaseURL, conf.WellKnownPath, lastEventID)

		if err != nil {
			return err
		}

		for i, event := range events {
			printEvent(event)

			if i == len(events)-1 {
				lastEventID = event.EventID
			}
		}
	}

	return nil
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
	fmt.Printf("[%s] [%s] %s %s\n", event.OccurredAt.Format(time.RFC3339), event.EventID, event.EventName, string(event.Payload))
}

var ErrEventNotFound = errors.New("event not found")

type config struct {
	BaseURL       string   `json:"base_url"`
	WellKnownPath string   `json:"well_known_path"`
	Ticker        Duration `json:"ticker"`
	LastEventID   string   `json:"last_event_id"`
}

func loadConfig(filename string) ([]config, error) {
	blob, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	var results []config

	if err := json.Unmarshal(blob, &results); err != nil {
		return nil, err
	}

	return results, nil
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(bytes []byte) error {
	var str string

	if err := json.Unmarshal(bytes, &str); err != nil {
		return err
	}

	dur, err := time.ParseDuration(str)

	if err != nil {
		return err
	}

	*d = Duration(dur)
	return nil
}
