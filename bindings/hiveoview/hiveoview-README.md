# HiveOView

Hive of Things Viewer, written in golang, html/htmx, web components, and sse using chi router, sprinkled with a little javascript.

## Status

This viewer is in early development. The information below is subject to change.

### Phase 1: SSR infrastructure and session management [done]
### Phase 2: Directory view
11. Show progress of action and config changes
11. Re-usable DataSchema component/template with text,number,bool,enum,on-off [partial]
12. Show history of events
Todo at some point...
- only show edit button if the user has permissions to edit
- briefly fade in/out a highlight of a changed value
- color value based on age - red is older than 3 days
- sort on various columns
- remember open/closed sections on page details (session storage) [done]
- fix handling of server restart
   * force logout after runtime restart (the sse reconnect will fail as users need to reauth)
   * F5 has 2 notifications, success connected and unauthenticated, but doesn't return to login page.
   * fails receiving zwavejs delivery status updates until zwavejs restart ??? 
- disable actions and config for things that are not reachable because the agent is offline 
  * api to get agent/thing status and Listen for the agent connect/disconnect event, sent by the runtime (todo)

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
2. Migrate to digital twin model [done]
3. Distinguish between basic and advanced attr/config/events - todo; how to describe this in the TD?
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
