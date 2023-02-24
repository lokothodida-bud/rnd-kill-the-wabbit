package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/thisisbud/backend-events-sidecar/pkg/budevents"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

func main() {
	port := flag.String("port", "8081", "HTTP port to run server")
	confFilename := flag.String("config-file", "config.json", "filename of config file to log from")
	flag.Parse()

	streams, err := loadStreamConfig(*confFilename)

	if err != nil {
		log.Panic(err)
	}

	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)
	r.Get("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		resp, err := generateResponse(streams)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", budevents.ContentType)
		_ = json.NewEncoder(w).Encode(resp)
	})
	r.Get("/v1/events/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		hrefs, err := parseIDToHrefs(chi.URLParam(r, "event_id"))

		if err != nil {
			log.Fatal(err)
		}

		resp, err := generateResponse(parseURLsToStreams(streams, hrefs))

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", budevents.ContentType)
		_ = json.NewEncoder(w).Encode(resp)
	})

	log.Printf("listening on port %s", *port)
	log.Panic(http.ListenAndServe(":"+*port, r))
}

type Stream struct {
	BaseURL       string `json:"base_url"`
	WellKnownPath string `json:"well_known_path"`
}

func loadStreamConfig(filename string) ([]Stream, error) {
	blob, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	var streams []Stream

	if err := json.Unmarshal(blob, &streams); err != nil {
		return nil, err
	}

	return streams, nil
}

func getEvent(fullURL string) (*budevents.Response, error) {
	resp, err := http.Get(fullURL)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unknown status code [%d]", resp.StatusCode)
	}

	var body budevents.Response

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	return &body, nil
}

func generateID(hrefs ...string) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(hrefs, ",")))
}

func parseIDToHrefs(id string) ([]string, error) {
	items, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(items), ","), nil
}

func parseURLsToStreams(originals []Stream, urls []string) []Stream {
	urlMap := map[string]bool{}

	for _, u := range urls {
		urlMap[u] = true
	}

	streams := []Stream{}

	for _, original := range originals {
		for _, url := range urls {
			if strings.HasPrefix(url, original.BaseURL) {
				streams = append(streams, Stream{
					BaseURL:       original.BaseURL,
					WellKnownPath: strings.ReplaceAll(url, original.BaseURL, ""),
				})
			}
		}
	}

	return streams
}

func generateResponse(streams []Stream) (*budevents.Response, error) {
	type wrappedResponse struct {
		budevents.Response
		StreamID string
		SelfRef  string
		NextRef  string
	}
	heads := []wrappedResponse{}

	for _, stream := range streams {
		head, err := getEvent(stream.BaseURL + stream.WellKnownPath)
		if err != nil {
			return nil, err
		}

		heads = append(heads, wrappedResponse{
			Response: *head,
			StreamID: stream.BaseURL,
			SelfRef:  stream.BaseURL + head.Metadata["self"].Href,
			NextRef:  stream.BaseURL + head.Metadata["next"].Href,
		})
	}

	sort.Slice(heads, func(i, j int) bool {
		return heads[i].Data.OccurredAt.After(heads[j].Data.OccurredAt)
	})

	// peek ahead to see what should be next
	selfIDs := []string{}
	nextIDs := []string{}

	if len(heads) > 0 && heads[0].Metadata["next"].Href != "" {
		selfIDs = append(nextIDs, heads[0].StreamID+heads[0].Metadata["self"].Href)
		nextIDs = append(nextIDs, heads[0].StreamID+heads[0].Metadata["next"].Href)
	}

	if len(heads) > 0 && heads[0].Metadata["self"].Href != "" {
		selfIDs = append(nextIDs, heads[0].StreamID+heads[0].Metadata["self"].Href)
	}

	if len(heads) > 1 {
		for _, head := range heads[1:] {
			nextIDs = append(nextIDs, head.SelfRef)
		}
	}

	resp := &budevents.Response{
		Data: heads[0].Data,
		Metadata: map[string]budevents.Reference{
			"self": {
				Href: "/v1/events/" + generateID(selfIDs...),
				Type: http.MethodGet,
			},
		},
	}

	if len(nextIDs) > 0 {
		resp.Metadata["next"] = budevents.Reference{
			Href: "/v1/events/" + generateID(nextIDs...),
			Type: http.MethodGet,
		}
	}

	return resp, nil
}
