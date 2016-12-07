package models

import "gopkg.in/mgo.v2/bson"

// Permission defines permissions or folders and documents.
type Permission struct {
	ID       bson.ObjectId `json:"_id" bson:"_id"`
	FolderID bson.ObjectId `json:"folderId" bson:"folderId"`
	UserID   bson.ObjectId `json:"userId" bson:"userId"`
	List     bool          `json:"list"`
	Read     bool          `json:"read"`
	Update   bool          `json:"update"`
	Create   bool          `json:"create"`
	Delete   bool          `json:"delete"`
	User     []User        `json:"-" bson:"user,omitempty"` // doesn't get stored in the database
}
