package models

import (
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

const col = "folders"

// Folder represents folders used to store documents
type Folder struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	ParentID bson.ObjectId `json:"parentId" bson:"parentId"`
	Name     string        `json:"name" bson:"name"`
}

// FolderHandler handles the indexing of folders
func FolderHandler(w http.ResponseWriter, r *http.Request) {
	folders, err := findAllUsers()
	user := getUserFromSession()
	if err != nil {
		ErrorLogger.Print("Error trying to find all users: \n", err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"folders": folders,
		"user":    user,
	}

	RenderTemplate(w, r, "users", data)
}

func findAllFolders() (folders *[]Folder, err error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(col)
	err = collection.Find(nil).All(&folders)
	if err != nil {
		return nil, err
	}

	return folders, nil
}

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
