package models

import "gopkg.in/mgo.v2/bson"

// Permission defines permissions or folders and documents.
type Permission struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	FolderID bson.ObjectId `json:"folderId" bson:"folderId"`
	UserID   bson.ObjectId `json:"userId" bson:"userId"`
	List     bool          `json:"list"`
	Read     bool          `json:"read"`
	Write    bool          `json:"write"`
	Create   bool          `json:"create"`
	Delete   bool          `json:"delete"`
	User     []User        `json:"-" bson:"user,omitempty"` // doesn't get stored in the database
}
