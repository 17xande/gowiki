package main

import (
	"html/template"
	"net/http"
	"strconv"

	"gopkg.in/mgo.v2/bson"
)

// Page represents any webpage on the site
type Page struct {
	ID    bson.ObjectId `json:"id" bson:"_id"`
	Title string        `json:"title" bson:"title"`
	Body  template.HTML `json:"body" bson:"body"`
	URL   string        `json:"url" bson:"url"`
	Level int           `json:"level" bson:"level"`
}

const pageCol = "pages"

// Save saves the page to the database
func (p *Page) Save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	_, err := collection.UpsertId(p.ID, p)
	return err
}

// LoadPage loads retrieves the page data from the database
func LoadPage(idHex string) (*Page, error) {
	id := bson.ObjectIdHex(idHex)
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	page := Page{}
	err := collection.FindId(id).One(&page)

	if err != nil {
		return nil, err
	}
	return &page, nil
}

func findAllDocs() (*[]Page, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	var pages []Page
	err := collection.Find(nil).All(&pages)
	if err != nil {
		return nil, err
	}

	return &pages, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, id string) {
	p, err := LoadPage(id)
	user := getUserFromSession()

	if err != nil {
		http.Redirect(w, r, "/edit/", http.StatusFound)
		return
	}

	if !levelCheck(w, r, p) {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	data := map[string]interface{}{
		"page": p,
		"user": user,
	}

	renderTemplate(w, r, "view", data)
}

func editHandler(w http.ResponseWriter, r *http.Request, id string) {
	p := &Page{}
	var err error
	user := getUserFromSession()

	if id != "" {
		p, err = LoadPage(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		if !levelCheck(w, r, p) {
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}

	users, err := findAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	data := map[string]interface{}{
		"page":  p,
		"users": users,
		"user":  user,
	}

	renderTemplate(w, r, "edit", data)
}

func saveHandler(w http.ResponseWriter, r *http.Request, idHex string) {
	title := r.FormValue("title")
	body := r.FormValue("body")
	// userIds := strings.Split(r.FormValue("permissions"), ",")
	level, err := strconv.Atoi(r.FormValue("level"))

	p := &Page{
		Title: title,
		Body:  template.HTML(body),
		Level: level,
	}

	if idHex != "" {
		p.ID = bson.ObjectIdHex(idHex)
	} else {
		p.ID = bson.NewObjectId()
	}

	err = p.Save()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// err = permissionsSave(userIds, id)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }

	http.Redirect(w, r, "/view/"+p.ID.Hex(), http.StatusFound)
}
