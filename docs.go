package main

import (
	"html/template"

	"gopkg.in/mgo.v2/bson"
)

// Page represents any webpage on the site
type Page struct {
	Title string
	Body  template.HTML
	URL   string
}

const pageCol = "pages"

// Save saves the page to the database
func (p *Page) Save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	_, err := collection.Upsert(bson.M{"url": p.URL}, &Page{p.Title, p.Body, p.URL})
	return err
}

// LoadPage loads retrieves the page data from the database
func LoadPage(url string) (*Page, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	page := Page{}
	err := collection.Find(bson.M{"url": url}).One(&page)

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
