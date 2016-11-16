package models

import (
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

// Permission defines permissions or folders and documents.
type Permission struct {
	UserID bson.ObjectId `json:"userId" bson:"userId"`
	List   bool          `json:"list"`
	Read   bool          `json:"read"`
	Update bool          `json:"update"`
	Create bool          `json:"create"`
}

// PermissionHandler handles GET requests to the Permissions page
func PermissionHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"hello": "there",
	}
	RenderTemplate(w, r, "permissions", data)
}
