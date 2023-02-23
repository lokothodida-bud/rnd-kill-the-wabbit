package handlers

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/thisisbud/backend-events-sidecar/internal/domain"
	"github.com/thisisbud/backend-events-sidecar/internal/storage"
	"net/http"
)

func Wellknown(w http.ResponseWriter, r *http.Request) {

}

func GetLatestEvent(getLatestEvent storage.GetLatestEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, refs, err := getLatestEvent(r.Context())

		if errors.Is(err, storage.ErrEventNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(domain.Response{
			Data:     *event,
			Metadata: refs,
		})
	}
}

func GetEvent(getEventByID storage.GetEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, refs, err := getEventByID(r.Context(), chi.URLParam(r, "event_id"))

		if errors.Is(err, storage.ErrEventNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(domain.Response{
			Data:     *event,
			Metadata: refs,
		})
	}
}

func PublishEvent(publish storage.PublishEvent, newID func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body domain.Event

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if body.EventID == "" {
			body.EventID = newID()
		}

		if err := publish(r.Context(), body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", "/v1/events/"+body.EventID)
		w.WriteHeader(http.StatusCreated)
	}
}
