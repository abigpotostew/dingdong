package handlers

import (
	"html/template"

	"github.com/pocketbase/pocketbase"
)

// Handlers contains all HTTP handlers for the application
type Handlers struct {
	app  *pocketbase.PocketBase
	tmpl *template.Template
}

// New creates a new Handlers instance
func New(app *pocketbase.PocketBase, tmpl *template.Template) *Handlers {
	return &Handlers{
		app:  app,
		tmpl: tmpl,
	}
}
