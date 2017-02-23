package models

import (
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/unrolled/render"
)

// RenderTemplate renders the given template and handles flash messages
func RenderTemplate(rend *render.Render, w http.ResponseWriter, r *http.Request, tmpl string, data map[string]interface{}) {
	// Get the user session from the context.
	ctx := r.Context()
	s, ok := ctx.Value(sessKey).(*sessions.Session)
	if !ok {
		err := errors.New("Error retrieving the session from context.\n")
		ErrorLogger.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if flashSuccess := s.Flashes("success"); len(flashSuccess) > 0 {
		data["flashSuccess"] = flashSuccess[0]
	}
	if flashInfo := s.Flashes("info"); len(flashInfo) > 0 {
		data["flashInfo"] = flashInfo[0]
	}
	if flashWarning := s.Flashes("warning"); len(flashWarning) > 0 {
		data["flashWarning"] = flashWarning[0]
	}
	if flashDanger := s.Flashes("danger"); len(flashDanger) > 0 {
		data["flashDanger"] = flashDanger[0]
	}
	// Session must be saved to empty the flash messages
	s.Save(r, w)

	if data["page"] == nil {
		data["page"] = tmpl
	}

	err := rend.HTML(w, http.StatusFound, tmpl, data)
	if err != nil {
		ErrorLogger.Print("Error trying to render page: "+tmpl, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// NotFoundHandler handles requests for pages that don't exist
func NotFoundHandler(rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := rend.HTML(w, http.StatusFound, "notFound", nil)
		if err != nil {
			ErrorLogger.Print("Error trying to render page: notFound", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
