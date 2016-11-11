package models

import (
	"html/template"
	"net/http"
	"path"
	"time"
)

// var templates = template.Must(template.ParseGlob("templates/*"))
var templates = make(map[string]*template.Template)

func init() {
	temp := "templates/"
	funcMap := template.FuncMap{
		"mod0": func(i int, mod int) bool {
			return i%mod == 0
		},
		"timeFormat": func(d time.Time) string {
			return d.Format(time.RFC822)
		},
	}

	templates["notFound.html"] = template.Must(template.ParseFiles(temp+"notFound.html", temp+"base.html"))
	templates["index.html"] = template.Must(template.New("index").Funcs(funcMap).ParseFiles(temp+"index.html", temp+"base.html"))
	templates["edit.html"] = template.Must(template.ParseFiles(temp+"edit.html", temp+"base.html"))
	templates["view.html"] = template.Must(template.New("view").Funcs(funcMap).ParseFiles(temp+"view.html", temp+"base.html"))
	templates["users.html"] = template.Must(template.ParseFiles(temp+"users.html", temp+"base.html"))
	templates["userEdit.html"] = template.Must(template.ParseFiles(temp+"userEdit.html", temp+"base.html"))
	templates["folder.html"] = template.Must(template.ParseFiles(temp+"folder.html", temp+"base.html"))
	templates["folders.html"] = template.Must(template.ParseFiles(temp+"folders.html", temp+"base.html"))
	templates["folderEdit.html"] = template.Must(template.ParseFiles(temp+"folderEdit.html", temp+"base.html"))
	templates["login.html"] = template.Must(template.ParseFiles(temp+"login.html", temp+"base.html"))
	templates["permissions.html"] = template.Must(template.ParseFiles(temp+"permissions.html", temp+"base.html"))
}

// RenderTemplate renders the given template and handles flash messages
func RenderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, data map[string]interface{}) {
	data["flashSuccess"] = UserSession.Flashes("success")
	data["flashInfo"] = UserSession.Flashes("info")
	data["flashWarning"] = UserSession.Flashes("warning")
	data["flashDanger"] = UserSession.Flashes("danger")
	// Session must be saved to empty the flash messages
	UserSession.Save(r, w)

	if data["page"] == nil {
		data["page"] = tmpl
	}

	err := templates[tmpl+".html"].ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// MakeHandler returns a handler function after it's checked that the URL is valid
func MakeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// m := validPath.FindStringSubmatch(r.URL.Path)
		// if m == nil {
		// 	http.NotFound(w, r)
		// 	return
		// }
		// fn(w, r, m[2])
		_, id := path.Split(r.URL.Path)

		fn(w, r, id)
	}
}

// NotFoundHandler handles requests for pages that don't exist
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	templates["notFound.html"].ExecuteTemplate(w, "base", nil)
}
