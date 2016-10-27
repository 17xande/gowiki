package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const host = "localhost"
const db = "test"

func dbConnect() *mgo.Session {
	session, err := mgo.Dial(host)
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	return session
}

// Save saves the page to the database
func (p *Page) Save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C("pages")
	_, err := collection.Upsert(bson.M{"url": p.URL}, &Page{p.Title, p.Body, p.URL})
	return err
}

// LoadPage loads retrieves the page data from the database
func LoadPage(url string) (*Page, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C("pages")
	page := Page{}
	err := collection.Find(bson.M{"url": url}).One(&page)

	if err != nil {
		return nil, err
	}
	return &page, nil
}
