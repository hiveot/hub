# HiveOView

Hive of Things Viewer, written in golang, html/htmx, web components, and sse using chi router, sprinkled with a little javascript.

## Status

This viewer is in early development. The information below is subject to change.

### Phase 1: SSR infrastructure and session management [done]

1. Setup a server with HTML template renderer [done]
2. Define Html templates for base layout, dashboard, about, and login pages [done]
3. Support SSE connections for dynamic updates [done]
4. Session management for hub connections support for push events [done]
7. Push connection status update. Present status view. [done]

### Phase 2: Directory view

1. Directory view using SSR rendering in golang [done]
2. Responsive layout [done]
3. Use Htmx to interactively browse the directory [done]
4. Thing details view [done]
5. Thing configuration edit [done]
6. Configuration of title (if supported by Thing) [partial]
7. Server push of property and event values [done]
8. View raw TD [done]
9. Delete TD w dialog [done]
10. Invoke action with dialog
11. DataSchema component with text,number,bool,enum,on-off 
todo at some point
- is text still serialized? event if data is a string?
  - what about stringified numbers. serialize again?
- only show edit button if the user has permissions to edit
- briefly fade in/out a highlight of a changed value
- color value based on age
- sort on various columns
- remember open/closed sections on page details (session storage) 

### Phase 3: Dashboard

1. Add/remove dashboard pages
2. Tiling engine in HTML/HTMX with persistent configuration
3. Value tile with a single Thing value with min/max
4. Card tile with values from Things
5. Line chart tile with value history
6. Subscribe to Thing values from tiles
7. Refresh tile

### Phase 4: iteration 2  (tbd, based on learnings)
1. Layout improvements for small screens
2. Subscribe to and organize Things by IoT Agents
2. Migrate to digital twin model [done]
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
