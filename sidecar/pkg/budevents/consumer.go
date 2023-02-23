package budevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"sort"
	"time"
)

type Consumer struct {
	listeners []Listener
	callback  func(ctx context.Context, event Event) error
}

func NewConsumer(
	callback func(ctx context.Context, event Event) error,
	listeners ...Listener,
) Consumer {
	return Consumer{
		listeners: listeners,
		callback:  callback,
	}
}

type Listener struct {
	BaseURL       string   `json:"base_url"`
	WellKnownPath string   `json:"well_known_path"`
	Ticker        Duration `json:"ticker"`
	LastEventID   string   `json:"last_event_id"`
}

func (consumer Consumer) Consume(ctx context.Context) error {
	errs, ctx := errgroup.WithContext(ctx)

	for _, listener := range consumer.listeners {
		errs.Go(func(l Listener) func() error {
			return func() error {
				return consumer.consumeEvents(ctx, l)
			}
		}(listener))
	}

	return errs.Wait()
}

func (consumer Consumer) consumeEvents(ctx context.Context, conf Listener) error {
	lastEventID := conf.LastEventID

	for range time.Tick(time.Duration(conf.Ticker)) {
		events, err := findLatestEvents(conf.BaseURL, conf.WellKnownPath, lastEventID)

		if err != nil {
			return err
		}

		for i, event := range events {
			if err := consumer.callback(ctx, event); err != nil {
				return err
			}

			if i == len(events)-1 {
				lastEventID = event.EventID
			}
		}
	}

	return nil
}

func findLatestEvents(baseURL string, wellknownURL string, latestEventID string) ([]Event, error) {
	resp, err := queryForEvent(baseURL + wellknownURL)

	if errors.Is(err, ErrEventNotFound) {
		return []Event{}, nil
	}

	if err != nil {
		return nil, err
	}

	if resp.Data.EventID == latestEventID {
		return []Event{}, nil
	}

	events := []Event{resp.Data}

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

func queryForEvent(eventURL string) (*Response, error) {
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

	var body Response

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	return &body, nil
}

var ErrEventNotFound = errors.New("event not found")

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
