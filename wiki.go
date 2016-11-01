package main

import (
	"html/template"
	"net/http"
	"os"
	"path"
	"regexp"

	"github.com/gorilla/context"
)

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9])$")

// var templates = template.Must(template.ParseGlob("templates/*"))
var templates = make(map[string]*template.Template)

func initi() {
	temp := "templates/"
	templates["index.html"] = template.Must(template.ParseFiles(temp+"index.html", temp+"base.html"))
	templates["edit.html"] = template.Must(template.ParseFiles(temp+"edit.html", temp+"base.html"))
	templates["view.html"] = template.Must(template.ParseFiles(temp+"view.html", temp+"base.html"))
	templates["users.html"] = template.Must(template.ParseFiles(temp+"users.html", temp+"base.html"))
	templates["userEdit.html"] = template.Must(template.ParseFiles(temp+"userEdit.html", temp+"base.html"))
	templates["login.html"] = template.Must(template.ParseFiles(temp+"login.html", temp+"base.html"))
	templates["permissions.html"] = template.Must(template.ParseFiles(temp+"permissions.html", temp+"base.html"))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession()
	pages, err := findAllDocs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	data := map[string]interface{}{
		"pages": pages,
		"user":  user,
	}

	renderTemplate(w, r, "index", data)
}

func renderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, data map[string]interface{}) {
	data["flashSuccess"] = UserSession.Flashes("success")
	data["flashInfo"] = UserSession.Flashes("info")
	data["flashWarning"] = UserSession.Flashes("warning")
	data["flashDanger"] = UserSession.Flashes("danger")
	// Session must be saved to empty the flash messages
	UserSession.Save(r, w)

	err := templates[tmpl+".html"].ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
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

func main() {
	initi()
	SessionInit()

	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/", SessionHandler(indexHandler))
	http.HandleFunc("/login", userLoginHandler)
	http.HandleFunc("/logout", userLogoutHandler)
	http.HandleFunc("/view/", SessionHandler(makeHandler(viewHandler)))
	http.HandleFunc("/edit/", SessionHandler(makeHandler(editHandler)))
	http.HandleFunc("/save/", SessionHandler(makeHandler(saveHandler)))
	http.HandleFunc("/users/", SessionHandler(userHandler))
	http.HandleFunc("/users/edit/", SessionHandler(userEditHandler))
	http.HandleFunc("/users/save/", SessionHandler(userSaveHandler))

	p := os.Getenv("PORT")
	if p == "" {
		p = ":80"
	}

	http.ListenAndServe(p, context.ClearHandler(http.DefaultServeMux))
}
