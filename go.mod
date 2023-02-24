module github.com/lokothodida/rnd-kill-the-wabbit

go 1.19

require (
	github.com/go-chi/chi/v5 v5.0.8
	github.com/google/uuid v1.3.0
	github.com/thisisbud/backend-events-sidecar v0.0.0-00010101000000-000000000000
	golang.org/x/sync v0.1.0
)

require github.com/go-chi/cors v1.2.1 // indirect

replace github.com/thisisbud/backend-events-sidecar => ./sidecar
