package memory

import (
	"context"
	"github.com/thisisbud/backend-events-sidecar/internal/storage"
	"github.com/thisisbud/backend-events-sidecar/pkg/budevents"
	"net/http"
	"sync"
)

type eventRepository struct {
	mu     *sync.Mutex
	events []budevents.Event
}

func NewEventRepository() *eventRepository {
	return &eventRepository{
		mu:     new(sync.Mutex),
		events: []budevents.Event{},
	}
}

func (repo *eventRepository) Publish(ctx context.Context, event budevents.Event) error {
	repo.mu.Lock()
	repo.events = append([]budevents.Event{event}, repo.events...)
	repo.mu.Unlock()

	return nil
}

func (repo *eventRepository) GetLatestEvent(
	ctx context.Context,
) (*budevents.Event, map[string]budevents.Reference, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.events) == 0 {
		return nil, nil, storage.ErrEventNotFound
	}

	if len(repo.events) == 1 {
		return &repo.events[0], nil, nil
	}

	return &repo.events[0], map[string]budevents.Reference{
		"next": {
			Href: "/v1/events/" + repo.events[1].EventID,
			Type: http.MethodGet,
		},
	}, nil
}

func (repo *eventRepository) GetEvent(
	ctx context.Context,
	eventID string,
) (*budevents.Event, map[string]budevents.Reference, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	refs := map[string]budevents.Reference{}

	for i, e := range repo.events {
		if e.EventID == eventID {
			if i < len(repo.events)-1 {
				refs["next"] = budevents.Reference{
					Href: "/v1/events/" + repo.events[i+1].EventID,
					Type: http.MethodGet,
				}
			}

			return &e, refs, nil
		}
	}

	return nil, nil, storage.ErrEventNotFound
}
