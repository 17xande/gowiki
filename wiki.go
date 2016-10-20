package main

import (
	"html/template"
	"net/http"
	"os"
	"regexp"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Page represents any webpage on the site
type Page struct {
	Title string
	Body  template.HTML
	URL   string
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

// var templates = template.Must(template.ParseGlob("templates/*"))
var templates = make(map[string]*template.Template)

func initi() {
	temp := "templates/"
	templates["index.html"] = template.Must(template.ParseFiles(temp+"index.html", temp+"base.html"))
	templates["edit.html"] = template.Must(template.ParseFiles(temp+"edit.html", temp+"base.html"))
	templates["view.html"] = template.Must(template.ParseFiles(temp+"view.html", temp+"base.html"))
}

func (p *Page) save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB("test").C("pages")
	_, err := collection.Upsert(bson.M{"url": p.URL}, &Page{p.Title, p.Body, p.URL})
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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	page := Page{}
	// err := templates.ExecuteTemplate(w, "index.html", page)
	err := templates["index.html"].ExecuteTemplate(w, "base", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
		p = &Page{URL: url}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, url string) {
	body := r.FormValue("body")
	p := &Page{Title: url, Body: template.HTML(body), URL: url}
	err := p.save()
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

func dbConnect() *mgo.Session {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	return session
}

func main() {
	initi()

	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	p := os.Getenv("PORT")
	if p == "" {
		p = ":8080"
	}

	http.ListenAndServe(p, nil)
}
