# R&D Q1 2023: "Kill the Wabbit"
> **R&D Q1 2023 Project**: provide an API spec (and possible integration sidecar and example) for a polling-based event
store, as an alternative to using RabbitMQ for consuming events between services

![Shhh...be vewwy vewwy quiet...](/images/elmer-fudd.jpg)

# RabbitMQ
![](/images/services-with-rabbit.png)
## Benefits

## Problems
- Infrastructural issues with maintaining the stateful server
- Handling failures to consume events
  - E.g. read an event, but responding to it results in an error. Retry policies
    and deadletter queues become needed to manage it

# Polling
![](/images/services-without-rabbit.png)
1. Record the latest event ID you saw
2. Hit the endpoint that tells you the latest event, and continue to follow
   the `next` link in the linked-list stream until you hit the latest event ID
3. Process the events you've seen in the list

# Intended design
- Event stream should have a linked-list structure that is easy to navigate, but
  also benefit from cacheability (i.e. shouldn't be changing too frequently)

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
      "type": "GET"
    },
    "self": {
      "href": "/events/9d536fb6-638e-4dcb-a2b7-ccef37949765",
      "type": "GET"
    },
    "next": {
      "href": "/events/ebb2ae28-10be-4f7a-bac4-796e28e25d85",
      "type": "GET"
    }
  }
}
```

# Questions raised
1. How do I consume events from the stream?
   - [x] [Code an example of doing so](/examples/consuming_events_from_stream.go)
2. How do I consume from multiple event streams?
   - Example coded above
3. How do I implement the spec with a server?
   - Example server implemented in sidecar and can be used for publishing
4. Should this form of polling be exposed to clients instead of webhooks?
   - Different use-cases potentially - webhooks would be more useful for especially timely info
   - Potential downside on webhooks: what happens if the responding server fails?
     - A deadletter queue for a specific client's messages that have failed?
5. In practice, how would this look?

# Benefits discovered
- It is easy to build up "view model" joiner services to cache information that joins
  across service boundaries - makes for microservices that have very little infrastructure
  to care about, and which store a small subset of information
- Replaying events can be done for free; as long as you know where you are in the stream, you can follow it
- Refactoring steps are straightforward:
  - Instead of publishing to RabbitMQ, persist your events in whichever server you'd like
    - Sidecar provides an example one that can be used out-of-the-box with HTTP
  - Services that consume the events poll your well-known/latest event endpoint and follow its
    links until they reach the last event they previously saw, and then process the payloads seen
- What about private event streams? (e.g. Rhino <-> Ryan)
  - Simply have another wellknown endpoint corresponding to that particular stream and encrypt the payloads

# Investigation links
- YOW! 2011 Jim Webber - Domain-Driven Design for RESTful Systems: https://www.youtube.com/watch?v=aQVSzMV8DWc
- https://sookocheff.com/post/api/on-choosing-a-hypermedia-format/
- https://devblast.com/r/master-ruby-web-apis/exploring-more-hypermedia-formats
- Oktane17: Designing Beautiful REST + JSON APIs - https://www.youtube.com/watch?v=MiOSzpfP1Ww

# Presentation Ideas
1. What problem are you solving? Who for?
    - Infrastructural issues with maintaining RabbitMQ - cause of many incidents and platform instability
    - Complexities with integration create hidden bugs (e.g. how to process failed messages)
2. What are the benefits for consumers, Bud employees/clients/partners
    - Consumers/clients/partners: more stable platform (reduced likelihood of us failing to send webhooks for example)
    - Bud employees: easier integration, using tech we already understand (HTTP and whatever DB technology we choose)
      - No more issues with queues filling up w/ messages when there are no consumers
      - Unified schema also makes it possible to consume from multiple event streams easily
      - Can replay events from existing streams in new services
      - Easier to build caches/viewmodels using persistent events
3. Technical and/or commercial viability
    - Relatively straightforward to build
    - Polling is cheap - we can also cache the results because the event linked list is immutable
    - For situations where external clients still require webhooks, we can still implement webhooks externally
      (so it is a backwards-compatible feature to implement)
4. Demo and/or findings
    - Show schema + example applications
    - Clients may not want to implement against the event stream, but given that we can still turn a stream into webhooks, we can
      still use this internally without compatibility issues
5. Next steps
    - Implement SQL storage driver for sidecar
    - Pick a service to help decouple from RabbitMQ and test that this integration works
