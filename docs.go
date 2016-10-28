package main

import (
	"html/template"

	"gopkg.in/mgo.v2/bson"
)

// Page represents any webpage on the site
type Page struct {
	// ID    bson.ObjectId `json:"id" bson:"_id"`
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
	_, err := collection.Upsert(bson.M{"url": p.URL}, p)
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
