package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/thisisbud/backend-events-sidecar/pkg/budevents"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Illustrates how to build a view-layer service purely out of listening to the
// event stream that is offered by the relevant service(s)
func main() {
	configFilename := flag.String("config-file", "./stream_config.dist.json", "JSON config file for consumers")
	port := flag.String("port", "8081", "HTTP port for server")

	flag.Parse()

	conf, err := loadConfig(*configFilename)

	if err != nil {
		log.Panic(err)
	}

	resp := getApplicationsResponder{
		mu:           new(sync.Mutex),
		applications: map[string]ApplicationView{},
	}

	consumer := budevents.NewConsumer(func(ctx context.Context, events ...budevents.Event) error {
		for _, event := range events {
			if err := resp.on(event); err != nil {
				return err
			}
		}
		return nil
	}, conf...)

	r := chi.NewRouter()
	r.Get("/loan-applications", func(w http.ResponseWriter, r *http.Request) {
		resp.render(w)
	})

	errs, ctx := errgroup.WithContext(context.Background())
	errs.Go(func() error {
		return consumer.Consume(ctx)
	})
	errs.Go(func() error {
		return http.ListenAndServe(":"+*port, r)
	})

	log.Panic(errs.Wait())
}

type getApplicationsResponder struct {
	mu           *sync.Mutex
	applications map[string]ApplicationView
}

func (apps *getApplicationsResponder) render(w http.ResponseWriter) {
	apps.mu.Lock()
	allApps := []ApplicationView{}
	for _, a := range apps.applications {
		allApps = append(allApps, a)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": allApps,
	})
	apps.mu.Unlock()
}

func (apps *getApplicationsResponder) on(event budevents.Event) error {
	apps.mu.Lock()
	defer apps.mu.Unlock()
	switch event.EventName {
	case "loan_application_submitted":
		return apps.loanApplicationSubmitted(event)
	case "loan_application_cancelled":
		return apps.loanApplicationCancelled(event)
	case "customer_requested_data_removal":
		return apps.customerRequestedDataRemoval(event)
	}

	return nil
}

func (apps *getApplicationsResponder) loanApplicationSubmitted(event budevents.Event) error {
	var payload struct {
		LoanApplicationID string `json:"loan_application_id"`
		CustomerID        string `json:"customer_id"`
		LoanType          string `json:"loan_type"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	apps.applications[payload.LoanApplicationID] = ApplicationView{
		ApplicationID: payload.LoanApplicationID,
		CustomerID:    payload.CustomerID,
		LoanType:      payload.LoanType,
		SubmittedAt:   event.OccurredAt.Format(time.RFC3339),
		Status:        "submitted",
	}

	return nil
}

func (apps *getApplicationsResponder) loanApplicationCancelled(event budevents.Event) error {
	var payload struct {
		LoanApplicationID string `json:"loan_application_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	app, ok := apps.applications[payload.LoanApplicationID]

	if !ok {
		return nil
	}

	app.Status = "cancelled"
	apps.applications[payload.LoanApplicationID] = app
	return nil
}

func (apps *getApplicationsResponder) customerRequestedDataRemoval(event budevents.Event) error {
	var payload struct {
		CustomerID string `json:"customer_id"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	for appID, app := range apps.applications {
		if app.CustomerID == payload.CustomerID {
			app.CustomerID = "-"
			app.Status = "data_removed"
			apps.applications[appID] = app
			return nil
		}
	}

	return nil
}

type ApplicationView struct {
	ApplicationID string `json:"application_id"`
	SubmittedAt   string `json:"submitted_at"`
	CustomerID    string `json:"customer_id"`
	LoanType      string `json:"loan_type"`
	Status        string `json:"status"`
}

func loadConfig(filename string) ([]budevents.Listener, error) {
	blob, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	var results []budevents.Listener

	if err := json.Unmarshal(blob, &results); err != nil {
		return nil, err
	}

	return results, nil
}
