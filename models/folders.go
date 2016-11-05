package models

import (
	"net/http"
	"path"

	"gopkg.in/mgo.v2/bson"
)

const col = "folders"

// Folder represents folders used to store documents
type Folder struct {
	ID              bson.ObjectId   `json:"id" bson:"_id"`
	Name            string          `json:"name" bson:"name"`
	ParentFolderIDs []bson.ObjectId `json:"parentFolderIDs" bson:"parentFolderIDs"`
	ParentFolders   []Folder
}

// FolderHandler handles the indexing of folders
func FolderHandler(w http.ResponseWriter, r *http.Request) {
	folders, err := findAllFolders()
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

	RenderTemplate(w, r, "folders", data)
}

// FolderEditHandler handles the editing of folders
func FolderEditHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var users *[]User
	f := &Folder{}
	exists := false
	user := getUserFromSession()

	_, id := path.Split(r.URL.Path)

	if len(id) > 0 {
		f, err = findFolder(id)
		exists = true
	}

	if err != nil {
		ErrorLogger.Print("Error trying to find folder {id: "+id+"}", err)
		UserSession.AddFlash("Error. Folder could not be retrieved.", "error")
	}

	users, err = findAllUsers()

	if err != nil {
		ErrorLogger.Print("Error trying to find folder {id: "+id+"}", err)
		UserSession.AddFlash("Error trying to find users for this folder {id: "+id+"}", "error")
	}

	tmpData := map[string]interface{}{
		"user":         user,
		"users":        users,
		"folder":       f,
		"exists":       exists,
		"flashError":   UserSession.Flashes("error"),
		"flashWarning": UserSession.Flashes("warning"),
		"flashAlert":   UserSession.Flashes("alert"),
	}

	err = templates["folderEdit.html"].ExecuteTemplate(w, "base", tmpData)
	if err != nil {
		ErrorLogger.Print("Error trying to display folder {id: "+id+"}", err)
		UserSession.AddFlash("Error. Folder could not be displayed.", "error")
	}
}

// FolderSaveHandler handles the saving of folders
func FolderSaveHandler(w http.ResponseWriter, r *http.Request) {
}

func findAllFolders() (*[]Folder, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(col)
	var folders []Folder
	err := collection.Find(nil).All(&folders)
	if err != nil {
		return nil, err
	}

	return &folders, nil
}

func findFolder(idHex string) (folder *Folder, err error) {
	id := bson.ObjectIdHex(idHex)
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(col)
	err = collection.FindId(id).One(folder)

	if err != nil {
		return nil, err
	}

	return folder, nil
}

func (f *Folder) save() (err error) {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(col)

	// Loop through the folders and store their IDs for the database
	for _, folder := range f.ParentFolders {
		f.ParentFolderIDs = append(f.ParentFolderIDs, folder.ID)
	}

	info, err := collection.UpsertId(f.ID, f)

	// If there was no user id, grab the DB generated id
	if len(f.ID.Hex()) == 0 {
		f.ID = info.UpsertedId.(bson.ObjectId)
	}

	return err
}
