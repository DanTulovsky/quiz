package handlers

import "embed"

// AssetsFS embeds the templates/assets directory for static serving
//
//go:embed templates/assets/*
var AssetsFS embed.FS
