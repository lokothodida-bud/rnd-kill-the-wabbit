# R&D Q1 2023: "Kill the Wabbit"
R&amp;D Q1 2023 Project: provide an API spec (and possible integration sidecar and example) for a polling-based event
store, as an alternative to using RabbitMQ for consuming events between services

# Investigation links
- YOW! 2011 Jim Webber - Domain-Driven Design for RESTful Systems: https://www.youtube.com/watch?v=aQVSzMV8DWc
- https://sookocheff.com/post/api/on-choosing-a-hypermedia-format/
- https://devblast.com/r/master-ruby-web-apis/exploring-more-hypermedia-formats

# RabbitMQ
## Benefits

## Problems
- Infrastructural issues with maintaining the stateful server
- Handling failures to consume events
  - E.g. read an event, but responding to it results in an error. Retry policies
    and deadletter queues become needed to manage it

# Polling
1. Record the latest event ID you saw
2. Hit the endpoint that tells you the latest event, and continue to follow
   the `next` link in the linked-list stream until you hit the latest event ID
3. Process the events you've seen in the list

# Intended design
- Event stream should have a linked-list structure that is easy to navigate, but
  also benefit from cacheability (i.e. shouldn't be changing too frequently)
- 

# Specification
- Schemas
  - `application/vnd.bud.events+json`
- Hypermedia controls
- Link Relations
  - `latest`
  - `next`
- Processing model
  - Atom-like

# Example
```json5
// Content-Type: application/vnd.bud.events+json
{
  "schema": "https://my-service-schema.json", // explains the payloads expected under data.payload
  "data": {
    "event_id": "9d536fb6-638e-4dcb-a2b7-ccef37949765",
    "event_name": "loan_application_originated",
    "occurred_at": "2023-02-22T00:00:00Z",
    "payload": {
      "organisation_id": "eaa78ed4-2aa3-426a-b6f9-9b857c99bc0a",
      "loan_application_id": "356d0346-4d7d-4595-aa42-123b784bb9da"
    },
  },
  "metadata": {
    "latest": {
      "href": "/events",
      type: "GET"
    },
    "next": {
      "href": "/events/ebb2ae28-10be-4f7a-bac4-796e28e25d85",
      "type": "GET"
    }
  }
}
```

# How do I consume events from the stream?
- Code an example of doing so
