# HiveOView

Hive of Things Viewer, written in golang, html/htmx, web components, and sse using chi router, sprinkled with a little javascript.

## Status

This viewer has reached alpha status with Phase 1-3 completed. The information below is subject to change.

### Phase 1: SSR infrastructure and session management [done]
### Phase 2: Directory view [done]
### Phase 3: Basic Dashboard (v1) [done]
### Phase 4: Request/Response Handling Improvements [in progress]
* Show action progress using async response messages (ActionStatus) [in progress]
* Indication of pending property writes (owserver updates can take 10 seconds)
* redirect to login when SSE receives unauthorized error (after server restart)
* Use cached values instead of reload when editing tiles
* Only view authorized sources

### Phase 5: Dashboard enhancements [in progress]
* multiple dashboards [done]
* Only show current dashboard tab on small screens
* Manual sort of tile sources in edit tile [done]
* dashboard configuration improvements [done]
  * dashboard static background image from file [done]
  * dashboard background image from URL [done]
  * configure tile transparency, background and color [done]
  * configure allowing tile placement anywhere  [done]
* only allow edit/move tiles if the user has permissions to edit, eg a manager+
  * depends on the authz change to define roles in the TD
* briefly fade in/out a highlight of a changed value (css transition?)
* option to color value based on age 
* show value error status in tiles
* disable actions and edits for things that are not reachable (agent offline, no auth) 
* Distinguish between basic and advanced attr/config/events using @type
   (TD does not support this facility, so use a config with @type values for basic values)
* Improve use of Thing vocabulary types in bindings

### Phase 6: Dashboard v2
* Dashboard groups
  * The dashboard consists of a grid containing tile groups
  * Each tile group consists a grid of tiles
  * Each tile contains a grid of widgets
  * 
* Customizable dashboard tiles
  * Tile template using grid layout
  * Select widget for one or more grid cell 
  * Configure tile with yaml
  * Widget placements in tile templates
  * Properties, events and action in/output as widget sources
* Built-in control widgets 
  * on/off toggle switch (lights, media control, valves, ...)
  * multi-value selector switch (for enums)
  * ranged control dial knob (dimmer, volume, ...)
  * ranged control slider (dimmer, volume, ...)
* Built-in indicators widgets
  * text gauge (values)
  * alarm indicator (motion)
  * dial gauge (ranged sensors such as humidity, temperature, )
    * min/max alarm threshold indicator - alarm color
  * horizontal/vertical slider gauge (ranged sensors such as humidity, temperature, )
    * min/max alarm threshold indicator - alarm color
  * history line/bar/scatter graph
    * 24 hours
    * current/min/max value
  * image widget (URL)
* Custom html widgets


## Objectives

Provide a reference implementation of a viewer on IoT data using the Hive Of Things backend. This includes a dashboard for presenting 'Thing' (device) outputs and controls for inputs.

Use HTMX with minimal of javascript. Just HTML, HTMX, CSS, Golang and web components. No frameworks, no nodejs, no JS/TS/CSS compile step. Single binary for easy deployment. 

## Summary

This service provides a web viewer for displaying IoT data.

It primary views are login, directory and one or more live dashboards that dynamically update as values change. The dashboard consists of tiles that present single or multiple values. Multiple dashboard pages can be added.

The frontend is generated from golang HTML/CSS templates and uses HTMX and JS for interactivity. It presents a directory and a dashboard with tiles that can be configured to display 'thing' values.

## Design

This viewer is implemented using golang, chi, go html templates, htmx, picocss and a sprinkle of javascript where needed. No UI framework such as react, vue, or svelte is used in the creation of this viewer. HTML is rendered server side using htmx and go templates, styled using picocss and served using chi router. Web components are used for providing client side behavior.

Dashboards:

The dashboard consists of a collection of tiles arranged in a grid. Tiles can be added, removed, moved and resized to occupy one or more dashboard grid cells.

Each tile consists of a configurable grid. Widgets are placed to occupy one or multiple tile grid cells. 

Widgets provides text, a graph, gauge, image or other graphic element.  A widget has zero or more data sources. A data source can be a Thing property, event, action output. A widget can have zero or more actuator outputs. An actuator output can be a Thing property (configuration) or action input.

has a presentation type that determines if it is text, a table, graph or gauge. Additional types can be added easily. Types are mapped to a html template that is inserted in the grid.

The tile editor constructs a html element from the configuration and stores this for rendering.

Data can be provided to tiles in multiple ways:
1. Statically during render. The tile contains all the data.
2. Dynamically after render. The tile uses htmx hx-get to load the data when mounted to the DOM.
3. Dynamically on sse events using htmx hx-trigger or sse-swap statements that are part of the tile.

Dashboards are stored with layout and tile configuration. The layout is 


## Build & Installation

Run 'make' to build the viewer as a standalone executable.

### Development Auto Reload

Tip: in the dev environment templates are reloaded on each render.

Tip for auto reload of the application during development, install 'air':
> go install github.com/cosmtrek/air

This installs 'air' into ~/go/bin. Add ~/go/bin to your path if it doesn't exist.

Run the application with 'air' from the development environment. This does not support breakpoints and does not run from
the IDE.
