package models

import (
	"net/http"
	"path"
	"strconv"

	"gopkg.in/mgo.v2/bson"
)

const col = "folders"

// Folder represents folders used to store documents
type Folder struct {
	ID          bson.ObjectId   `json:"id" bson:"_id"`
	Name        string          `json:"name"`
	Level       int             `json:"level"`
	UserIDs     []bson.ObjectId `json:"userIDs" bson:"userIDs"`
	Users       []User          `json:"-" bson:"-"` // doesn't get stored in the database
	Documents   []Document      `json:"-" bson:"documents,omitempty"`
	Permissions []Permission    `json:"-" bson:"permissions,omitempty"`
	// We might have folders within folders in the future
	// FolderIDs   []bson.ObjectId `json:"folderIDs" bson:"folderIDs"`
	// Folders     []Folder        `json:"-" bson:"-"` // doesn't get stored in the database
}

// FolderHandler handles the folder page, where all the documents in a folder are displayed
func FolderHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	f := &Folder{}
	user := getUserFromSession()

	_, id := path.Split(r.URL.Path)

	if len(id) == 0 {
		http.Redirect(w, r, "/folders/", http.StatusFound)
		return
	}

	f, err = findFolder(id)
	if err != nil {
		ErrorLogger.Print("Error trying to find folder: {id: "+id+"}\n", err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		UserSession.Save(r, w)
		http.Redirect(w, r, "/folders/", http.StatusFound)
		err = nil
		return
	}

	err = f.findDocsForFolder()
	if err != nil {
		ErrorLogger.Print("Error trying to find document for folder: {id: "+id+"}\n", err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		UserSession.Save(r, w)
		http.Redirect(w, r, "/folders/", http.StatusFound)
		err = nil
		return
	}

	data := map[string]interface{}{
		"user":   user,
		"folder": f,
	}

	RenderTemplate(w, r, "folder", data)
}

// FoldersHandler handles the indexing of folders
func FoldersHandler(w http.ResponseWriter, r *http.Request) {
	folders, err := findAllFolders()
	user := getUserFromSession()
	if err != nil {
		ErrorLogger.Print("Error trying to find all users: \n", err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		UserSession.Save(r, w)
		http.Redirect(w, r, "/", http.StatusFound)
		err = nil
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
		UserSession.Save(r, w)
		err = nil
	}

	users, err = findAllUsers()

	if err != nil {
		ErrorLogger.Print("Error trying to find folder {id: "+id+"}", err)
		UserSession.AddFlash("Error trying to find users for this folder {id: "+id+"}", "error")
		UserSession.Save(r, w)
		err = nil
	}

	tmpData := map[string]interface{}{
		"user":   user,
		"users":  users,
		"folder": f,
		"exists": exists,
	}

	RenderTemplate(w, r, "folderEdit", tmpData)
	if err != nil {
		ErrorLogger.Print("Error trying to display folder {id: "+id+"}", err)
		UserSession.AddFlash("Error. Folder could not be displayed.", "error")
		UserSession.Save(r, w)
		err = nil
	}
}

// FolderSaveHandler handles the saving of folders
func FolderSaveHandler(w http.ResponseWriter, r *http.Request) {
	var f *Folder
	_, id := path.Split(r.URL.Path)

	if r.Method == "POST" {
		var userIDs []bson.ObjectId
		r.ParseForm()
		name := r.Form["name"][0]
		strUserIDs := r.Form["users"]
		level, err := strconv.Atoi(r.Form["level"][0])

		if err != nil {
			ErrorLogger.Print("Error parsing folder level POST. {id: "+id+"} ", err.Error())
			UserSession.AddFlash("Error saving folder settings. If this error persists, please contact support.", "error")
			UserSession.Save(r, w)
			err = nil
		}

		for _, uID := range strUserIDs {
			userIDs = append(userIDs, bson.ObjectIdHex(uID))
		}

		f = &Folder{
			Name:    name,
			Level:   level,
			UserIDs: userIDs,
		}

		if id != "" {
			f.ID = bson.ObjectIdHex(id)
		} else {
			f.ID = bson.NewObjectId()
		}

		err = f.save()

		if err != nil {
			ErrorLogger.Print("Error saving folder to database. {id: "+id+"} ", err.Error())
			UserSession.AddFlash("Error saving folder settings. If this error persists, please contact support.", "error")
			UserSession.Save(r, w)
			err = nil
		}

		InfoLogger.Print("Folder saved {id: " + f.ID.Hex() + "}")
	}

	http.Redirect(w, r, "/folders/", http.StatusFound)
}

// FolderPermissionEditHandler handles permission editing for folders
func FolderPermissionEditHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	f := &Folder{}
	user := getUserFromSession()

	_, id := path.Split(r.URL.Path)

	if len(id) == 0 {
		UserSession.AddFlash("Couldn't edit permissions for that folder.", "warning")
		UserSession.Save(r, w)
		ErrorLogger.Print("Tried to load the Folder Permissions page without a FolderID")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	f, err = findFolder(id)
	if err != nil {
		ErrorLogger.Print("Error trying to find folder {id: "+id+"} ", err)
		UserSession.AddFlash(" Folder could not be retrieved.", "error")
		UserSession.Save(r, w)
		http.Redirect(w, r, "/folder/"+id, http.StatusFound)
		err = nil
	}

	err = f.getPermissions()
	if err != nil {
		ErrorLogger.Print("Error trying to get permissions for folder {id: "+id+"} ", err)
		UserSession.AddFlash(" Folder permissions could not be retrieved.", "error")
		UserSession.Save(r, w)
		http.Redirect(w, r, "/folder/"+id, http.StatusFound)
		err = nil
	}

	// Get all the userIDs out of any permissions for this folder.
	iPerm := len(f.Permissions)
	notUserIds := make([]bson.ObjectId, iPerm)

	for i, perm := range f.Permissions {
		notUserIds[i] = perm.UserID
	}

	users, err := findNotUsers(&notUserIds)
	if err != nil {
		ErrorLogger.Print("Error trying to find rest of users.", err)
		UserSession.AddFlash("Couldn't retrieve rest of users from database", "error")
		UserSession.Save(r, w)
		err = nil
	}

	// permittedUsers, err := findUsers()

	tmpData := map[string]interface{}{
		"user":   user,
		"users":  users,
		"folder": f,
	}

	RenderTemplate(w, r, "folderPermissions", tmpData)
	if err != nil {
		ErrorLogger.Print("Error trying to display folder {id: "+id+"}", err)
		UserSession.AddFlash("Error. Folder could not be displayed.", "error")
		UserSession.Save(r, w)
		err = nil
	}
}

func findAllFolders() (*[]Folder, error) {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(col)
	var folders []Folder

	err := collection.Find(nil).Sort("name").All(&folders)
	if err != nil {
		return nil, err
	}

	return &folders, nil
}

func findFolder(idHex string) (*Folder, error) {
	id := bson.ObjectIdHex(idHex)
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(col)
	f := &Folder{}

	err := collection.FindId(id).One(f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func findFolderWithPermissions(id bson.ObjectId) (*Folder, error) {
	var folder Folder
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(col)

	query := []bson.M{
		{
			"$unwind": "$permissions",
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "permissions.userId",
				"foreignField": "_id",
				"as":           "permissions.user",
			},
		},
		{
			"$match": bson.M{"name": "test"},
		},
	}

	err := collection.Pipe(query).One(folder)

	if err != nil {
		return nil, err
	}

	return &folder, err
}

func findFoldersAndDocuments() (*[]Folder, error) {
	session := dbConnect()
	defer session.Close()
	user := getUserFromSession()
	collection := session.DB(db).C(col)
	var folders []Folder

	query := []bson.M{{
		"$lookup": bson.M{ // lookup the documents table here
			"from":         "documents",
			"localField":   "_id",
			"foreignField": "folderID",
			"as":           "documents",
		}},
		{"$match": bson.M{
			"$or": []bson.M{
				bson.M{"documents.level": bson.M{"$lte": user.Level}},
				bson.M{"documents.userIDs": user.ID},
				bson.M{"level": bson.M{"$lte": user.Level}},
				bson.M{"userIDs": user.ID},
			}},
		}}

	pipe := collection.Pipe(query)
	err := pipe.All(&folders)

	if err != nil {
		return nil, err
	}

	return &folders, nil
}

func (f *Folder) getPermissions() error {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C("folderPermissions")

	query := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "userId",
				"foreignField": "_id",
				"as":           "user",
			},
		},
		{"$match": bson.M{"folderId": f.ID}},
	}

	pipe := collection.Pipe(query)
	err := pipe.All(&f.Permissions)

	return err
}

func (f *Folder) save() (err error) {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(col)

	_, err = collection.UpsertId(f.ID, f)

	return err
}

func (f *Folder) findDocsForFolder() error {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(documentCol)
	var docs []Document

	err := collection.Find(bson.M{"folderID": f.ID}).All(&docs)
	f.Documents = docs

	return err
}
