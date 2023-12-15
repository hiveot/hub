package views

import (
	"embed"
)

// index.html app/*.html static/*
//
//go:embed *
var EmbeddedTemplates embed.FS

//go:embed static/*
var EmbeddedStatic embed.FS
