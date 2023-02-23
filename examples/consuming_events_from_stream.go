package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/thisisbud/backend-events-sidecar/pkg/budevents"
	"log"
	"os"
	"time"
)

func main() {
	configFilename := flag.String("config-file", "./stream_config.dist.json", "JSON config file for consumers")

	flag.Parse()

	conf, err := loadConfig(*configFilename)

	if err != nil {
		log.Panic(err)
	}

	consumer := budevents.NewConsumer(printEvents, conf...)

	log.Panic(consumer.Consume(context.Background()))
}

func printEvents(ctx context.Context, events ...budevents.Event) error {
	for _, event := range events {
		fmt.Printf("[%s] [%s] %s %s\n", event.OccurredAt.Format(time.RFC3339), event.EventID, event.EventName, string(event.Payload))
	}
	return ctx.Err()
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
