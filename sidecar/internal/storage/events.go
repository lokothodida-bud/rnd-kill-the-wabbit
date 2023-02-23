package storage

import (
	"context"
	"errors"
	"github.com/thisisbud/backend-events-sidecar/pkg/budevents"
)

type PublishEvent func(ctx context.Context, event budevents.Event) error

type GetEvent func(ctx context.Context, eventID string) (*budevents.Event, map[string]budevents.Reference, error)

type GetLatestEvent func(ctx context.Context) (*budevents.Event, map[string]budevents.Reference, error)

var ErrEventNotFound = errors.New("event not found")
