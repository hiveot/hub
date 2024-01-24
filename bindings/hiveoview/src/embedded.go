package src

import (
	"embed"
)

// EmbeddedViews contains all html templates
//
//go:embed views/*.html
var EmbeddedViews embed.FS

// EmbeddedStatic contains all static assets
//
//go:embed static/* webcomp/*
var EmbeddedStatic embed.FS
