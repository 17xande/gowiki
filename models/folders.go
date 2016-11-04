package models

import (
	"gopkg.in/mgo.v2/bson"
)

const col = "folders"

// Folder represents folders used to store documents
type Folder struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	ParentID bson.ObjectId `json:"parentId" bson:"parentId"`
	Name     string        `json:"name" bson:"name"`
}

// func init() {

// }

func (f *Folder) save() (err error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(col)

	info, err := collection.UpsertId(f.ID, &f)

	// If there was no user id, grab the DB generated id
	if len(f.ID.Hex()) == 0 {
		f.ID = info.UpsertedId.(bson.ObjectId)
	}

	return err
}
