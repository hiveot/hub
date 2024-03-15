# HiveOView

Hive of Things Viewer, written in golang, html/htmx, web components, and sse using chi router, sprinkled with a little javascript.

## Status

This viewer is in early development. The information below is subject to change.

### Phase 1: SSR infrastructure and session management

1. Setup a server with HTML template renderer [done]
2. Define Html templates for base layout, dashboard, about, and login pages [done]
3. Support SSE connections for dynamic updates [done]
4. Session management for MQTT hub connections and sse support to push events [done]
5. MQTT Authentication in client session [done]
6. NATS Authentication in client session (todo: session auth without keys)
7. Push connection status update. Present status view. [done]

### Phase 2: Directory view

1. Directory view using SSR rendering in golang [done]
2. Responsive layout [done]
3. Use Htmx to interactively browse the directory [done]
4. Thing details view [done]
5. Thing configuration edit [done]
6. Configuration of title (if supported by Thing) [partial]
7. Server push of property and event values

### Phase 3: Dashboard

1. Add/remove dashboard pages
2. Tiling engine in HTML/HTMX with persistent configuration
3. Value tile with a single Thing value with min/max
4. Card tile with values from Things
5. Line chart tile with value history
    Phase 4: Live updates
6. Subscribe to Thing values from tiles
7. Refresh tile

### Phase 4: iteration 2  (tbd, based on learnings)
1. Subscribe to and organize Things by IoT Agents
2. Migrate to digital twin model 
3. Distinguish between basic and advanced attr/config/events
4. Indication of pending configuration update (owserver updates can take 10 seconds)


## Objective

Provide a reference implementation of a viewer on IoT data using the Hive Of Things backend. This includes a dashboard for presenting 'Thing' (device) outputs and controls for inputs.

Use HTMX with bare minimum of javascript. Just HTML, HTMX, CSS, Golang and maybe a sprinkle of plain javascript as needed. No frameworks, no nodejs, no JS/TS/CSS compile step. Single binary for easy deployment.

## Summary

This service provides a web viewer for displaying IoT data.

It primary views are login, directory and a live dashboard that updates as values change. The dashboard consists of
tiles that present single or multiple values. Multiple presentations are available.

The frontend is generated from HTML/CSS templates and uses HTMx for interactivity
It presents a dashboard with tiles that can be configured to display 'thing' values.
Each tile can present itself as a card or history graph using information from the directory and history service.

## Implementation

This viewer is implemented using golang, echo, go html templates, htmx, beercss and a sprinkle of javascript where needed. No UI framework such as react, vue, or svelte is used in the creation of this viewer. Instead HTML is rendered server side using htmx and go templates, styled using beercss and served using echo. For each client session a mqtt/nats client is created to receive thing updates.

## Build & Installation

### Development Auto Reload

For auto reload of the application during development, install 'air':
> go install github.com/cosmtrek/air

This installs 'air' into ~/go/bin. Add ~/go/bin to your path if it doesn't exist.

Run the application with 'air' from the development environment. This does not support breakpoints and does not run from
the IDE.

... todo ... lets get something working first
