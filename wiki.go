package main

import (
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/gorilla/context"
)

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

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
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	pages, err := findAllDocs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = templates["index.html"].ExecuteTemplate(w, "base", pages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, url string) {
	p, err := LoadPage(url)
	if err != nil {
		http.Redirect(w, r, "/edit/"+url, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, url string) {
	p, err := LoadPage(url)
	if err != nil {
		p = &Page{URL: url}
	}
	users, err := findAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	result := map[string]interface{}{
		"page":  p,
		"users": users,
	}

	err = templates["edit.html"].ExecuteTemplate(w, "base", result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request, url string) {
	body := r.FormValue("body")
	// permissions := r.FormValue("permissions")
	level, err := strconv.Atoi(r.FormValue("level"))

	p := &Page{Title: url, Body: template.HTML(body), URL: url, Level: level}
	err = p.Save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+url, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	// err := templates.ExecuteTemplate(w, tmpl+".html", p)
	err := templates[tmpl+".html"].ExecuteTemplate(w, "base", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	initi()
	SessionInit()

	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/", SessionHandler(indexHandler))
	http.HandleFunc("/login", userLoginHandler)
	http.HandleFunc("/logout", userLogoutHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/users/", userHandler)
	http.HandleFunc("/users/edit/", userEditHandler)
	http.HandleFunc("/users/save/", userSaveHandler)

	p := os.Getenv("PORT")
	if p == "" {
		p = ":8080"
	}

	http.ListenAndServe(p, context.ClearHandler(http.DefaultServeMux))
}
