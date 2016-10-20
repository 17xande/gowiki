package main

import (
	"html/template"
	"net/http"
	"os"
	"regexp"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Page
type Page struct {
	Title string
	Body  []byte
	Url   string
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var templates = template.Must(template.ParseFiles("templates/edit.html", "templates/view.html"))

func (p *Page) save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB("test").C("pages")
	_, err := collection.Upsert(bson.M{"url": p.Url}, &Page{p.Title, p.Body, p.Url})
	return err
}

func loadPage(url string) (*Page, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB("test").C("pages")
	page := Page{}
	err := collection.Find(bson.M{"url": url}).One(&page)

	if err != nil {
		return nil, err
	}
	return &page, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, url string) {
	p, err := loadPage(url)
	if err != nil {
		http.Redirect(w, r, "/edit/"+url, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, url string) {
	p, err := loadPage(url)
	if err != nil {
		p = &Page{Url: url}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, url string) {
	body := r.FormValue("body")
	p := &Page{Title: url, Body: []byte(body), Url: url}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+url, http.StatusFound)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
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

func dbConnect() *mgo.Session {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	return session
}

func main() {
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	p := os.Getenv("PORT")
	if p == "" {
		p = ":8080"
	}

	http.ListenAndServe(p, nil)
}
