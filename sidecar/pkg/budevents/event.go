package budevents

import (
	"encoding/json"
	"time"
)

const ContentType = "application/vnd.bud.events+json"

type Event struct {
	EventID    string          `json:"event_id"`
	EventName  string          `json:"event_name"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

type Response struct {
	Data     Event                `json:"data"`
	Metadata map[string]Reference `json:"metadata"`
}

type Reference struct {
	Href string `json:"href"`
	Type string `json:"type"`
}
