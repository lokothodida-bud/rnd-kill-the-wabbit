package memory

import (
	"context"
	"github.com/thisisbud/backend-events-sidecar/internal/domain"
	"github.com/thisisbud/backend-events-sidecar/internal/storage"
	"net/http"
	"sync"
)

type eventRepository struct {
	mu     *sync.Mutex
	events []domain.Event
}

func NewEventRepository() *eventRepository {
	return &eventRepository{
		mu:     new(sync.Mutex),
		events: []domain.Event{},
	}
}

func (repo *eventRepository) Publish(ctx context.Context, event domain.Event) error {
	repo.mu.Lock()
	repo.events = append([]domain.Event{event}, repo.events...)
	repo.mu.Unlock()

	return nil
}

func (repo *eventRepository) GetLatestEvent(
	ctx context.Context,
) (*domain.Event, map[string]domain.Reference, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.events) == 0 {
		return nil, nil, storage.ErrEventNotFound
	}

	if len(repo.events) == 1 {
		return &repo.events[0], nil, nil
	}

	return &repo.events[0], map[string]domain.Reference{
		"next": {
			Href: "/v1/events/" + repo.events[1].EventID,
			Type: http.MethodGet,
		},
	}, nil
}

func (repo *eventRepository) GetEvent(
	ctx context.Context,
	eventID string,
) (*domain.Event, map[string]domain.Reference, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	refs := map[string]domain.Reference{}

	for i, e := range repo.events {
		if e.EventID == eventID {
			if i < len(repo.events)-1 {
				refs["next"] = domain.Reference{
					Href: "/v1/events/" + repo.events[i+1].EventID,
					Type: http.MethodGet,
				}
			}

			return &e, refs, nil
		}
	}

	return nil, nil, storage.ErrEventNotFound
}
