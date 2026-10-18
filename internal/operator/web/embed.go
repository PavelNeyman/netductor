package web

import "embed"

//go:embed index.html app.css app.js mockups.html
var FS embed.FS
