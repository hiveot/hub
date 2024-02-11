package src

import (
	"embed"
)

// EmbeddedViews contains all html templates
// FIXME: for some reason "views/*.gohtml" does not work
//
//go:embed views
var EmbeddedViews embed.FS

// EmbeddedStatic contains all static assets
//
//go:embed static/* webcomp/*
var EmbeddedStatic embed.FS
