package storage

import (
	"context"
	"errors"
	"github.com/thisisbud/backend-events-sidecar/internal/domain"
)

type PublishEvent func(ctx context.Context, event domain.Event) error

type GetEvent func(ctx context.Context, eventID string) (*domain.Event, map[string]domain.Reference, error)

type GetLatestEvent func(ctx context.Context) (*domain.Event, map[string]domain.Reference, error)

var ErrEventNotFound = errors.New("event not found")
