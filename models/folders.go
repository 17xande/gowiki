package models

import (
	"errors"
	"html/template"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/unrolled/render"

	"encoding/json"

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
func FolderHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		f := &Folder{}

		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) == 0 {
			http.Redirect(w, r, "/folders/", http.StatusFound)
			return
		}

		// Get the user session from the context.
		ctx := r.Context()
		s, ok := ctx.Value(sessKey).(*sessions.Session)
		if !ok {
			err := errors.New("Error retrieving the session from context.\n")
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user, ok := getUserFromSession(s)
		if !ok {
			return
		}

		f, err = findFolder(db, id)
		if err != nil {
			ErrorLogger.Print("Error trying to find folder: {id: "+id+"}\n", err)
			s.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			s.Save(r, w)
			http.Redirect(w, r, "/folders/", http.StatusFound)
			err = nil
			return
		}

		err = f.findDocsForFolder(db)
		if err != nil {
			ErrorLogger.Print("Error trying to find document for folder: {id: "+id+"}\n", err)
			s.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			s.Save(r, w)
			http.Redirect(w, r, "/folders/", http.StatusFound)
			err = nil
			return
		}

		data := map[string]interface{}{
			"user":   user,
			"folder": f,
		}

		RenderTemplate(rend, w, r, "folder/view", data)
	}
}

// FoldersHandler handles the indexing of folders
func FoldersHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		folders, err := findAllFolders(db)
		// Get the user session from the context.
		ctx := r.Context()
		s, ok := ctx.Value(sessKey).(*sessions.Session)
		if !ok {
			err := errors.New("Error retrieving the session from context.\n")
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user, ok := getUserFromSession(s)
		if !ok {
			return
		}

		if err != nil {
			ErrorLogger.Print("Error trying to find all users: \n", err)
			s.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			s.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
			err = nil
			return
		}

		data := map[string]interface{}{
			"folders": folders,
			"user":    user,
		}

		RenderTemplate(rend, w, r, "folder/index", data)
	}
}

// FolderEditHandler handles the editing of folders
func FolderEditHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var users *[]User
		f := &Folder{}
		exists := false

		// Get the user session from the context.
		ctx := r.Context()
		s, ok := ctx.Value(sessKey).(*sessions.Session)
		if !ok {
			err := errors.New("Error retrieving the session from context.\n")
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user, ok := getUserFromSession(s)
		if !ok {
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) > 0 {
			f, err = findFolder(db, id)
			exists = true
		}

		if err != nil {
			ErrorLogger.Print("Error trying to find folder {id: "+id+"}", err)
			s.AddFlash("Error. Folder could not be retrieved.", "error")
			s.Save(r, w)
			err = nil
		}

		users, err = findAllUsers(db)

		if err != nil {
			ErrorLogger.Print("Error trying to find folder {id: "+id+"}", err)
			s.AddFlash("Error trying to find users for this folder {id: "+id+"}", "error")
			s.Save(r, w)
			err = nil
		}

		data := map[string]interface{}{
			"user":   user,
			"users":  users,
			"folder": f,
			"exists": exists,
		}

		RenderTemplate(rend, w, r, "folder/edit", data)
	}
}

// FolderSaveHandler handles the saving of folders
func FolderSaveHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var f *Folder
		vars := mux.Vars(r)
		id := vars["id"]

		// Get the user session from the context.
		ctx := r.Context()
		s, ok := ctx.Value(sessKey).(*sessions.Session)
		if !ok {
			err := errors.New("Error retrieving the session from context.\n")
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.Method == "POST" {
			var userIDs []bson.ObjectId
			r.ParseForm()
			name := r.Form["name"][0]
			strUserIDs := r.Form["users"]
			level, err := strconv.Atoi(r.Form["level"][0])

			if err != nil {
				ErrorLogger.Print("Error parsing folder level POST. {id: "+id+"} ", err.Error())
				s.AddFlash("Error saving folder settings. If this error persists, please contact support.", "error")
				s.Save(r, w)
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

			err = f.save(db)

			if err != nil {
				ErrorLogger.Print("Error saving folder to database. {id: "+id+"} ", err.Error())
				s.AddFlash("Error saving folder settings. If this error persists, please contact support.", "error")
				s.Save(r, w)
				err = nil
			}

			InfoLogger.Print("Folder saved {id: " + f.ID.Hex() + "}")
		}

		http.Redirect(w, r, "/folders/", http.StatusFound)
	}
}

// FolderPermissionsEditHandler handles permission editing for folders
func FolderPermissionsEditHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		f := &Folder{}
		// Get the user session from the context.
		ctx := r.Context()
		s, ok := ctx.Value(sessKey).(*sessions.Session)
		if !ok {
			err := errors.New("Error retrieving the session from context.\n")
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user, ok := getUserFromSession(s)
		if !ok {
			return
		}

		_, id := path.Split(r.URL.Path)

		if len(id) == 0 {
			s.AddFlash("Couldn't edit permissions for that folder.", "warning")
			s.Save(r, w)
			ErrorLogger.Print("Tried to load the Folder Permissions page without a FolderID")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		f, err = findFolder(db, id)
		if err != nil {
			ErrorLogger.Print("Error trying to find folder {id: "+id+"} ", err)
			s.AddFlash(" Folder could not be retrieved.", "error")
			s.Save(r, w)
			http.Redirect(w, r, "/folder/"+id, http.StatusFound)
			err = nil
		}

		err = f.getPermissions(db)
		if err != nil {
			ErrorLogger.Print("Error trying to get permissions for folder {id: "+id+"} ", err)
			s.AddFlash(" Folder permissions could not be retrieved.", "error")
			s.Save(r, w)
			http.Redirect(w, r, "/folder/"+id, http.StatusFound)
			err = nil
		}

		users, err := findAllUsers(db)
		if err != nil {
			ErrorLogger.Print("Error trying to find all users.", err)
			s.AddFlash("Couldn't retrieve all users from database", "error")
			s.Save(r, w)
			err = nil
		}

		// permittedUsers, err := findUsers()
		jsPermissions, err := json.Marshal(f.Permissions)
		if err != nil {
			ErrorLogger.Print("Error trying to marshal permissions", err)
			s.AddFlash("We're experiencing some technical difficulties on that page.", "error")
			s.Save(r, w)
			err = nil
		}

		jsUsers, err := json.Marshal(users)
		if err != nil {
			ErrorLogger.Print("Error trying to marshal users.", err)
			s.AddFlash("We're experiencing some technical difficulties on that page.", "error")
			s.Save(r, w)
			err = nil
		}

		data := map[string]interface{}{
			"user":          user,
			"users":         users,
			"folder":        f,
			"jsPermissions": template.JS(jsPermissions),
			"jsUsers":       template.JS(jsUsers),
		}

		RenderTemplate(rend, w, r, "folder/permissions", data)
	}
}

// FolderPermissionsSaveHandler handles save POST requests with folder permission data.
func FolderPermissionsSaveHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p []Permission
		var err error
		_, id := path.Split(r.URL.Path)

		if r.Method == "POST" {
			// Get the user session from the context.
			ctx := r.Context()
			s, ok := ctx.Value(sessKey).(*sessions.Session)
			if !ok {
				err := errors.New("Error retrieving the session from context.\n")
				ErrorLogger.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			r.ParseForm()
			strPerms := r.Form["folderPermissions"][0]
			err = json.Unmarshal([]byte(strPerms), &p)
			if err != nil {
				ErrorLogger.Print("Error unmarshalling folder permissions. {id: "+id+"}\n", err.Error())
				s.AddFlash("Error loading folder permissions.", "error")
				s.Save(r, w)
				http.Redirect(w, r, "/folder/edit/"+id, http.StatusFound)
				return
			}

			err = permissionSave(db, p)
			if err != nil {
				ErrorLogger.Print("Error saving folder permissions. {id: "+id+"}\n", err.Error())
				s.AddFlash("Error saving folder permission.", "error")
				http.Redirect(w, r, "/folder/edit/"+id, http.StatusFound)
				return
			}
			InfoLogger.Print("Folder permissions saved {id: " + id + "}")
		}

		http.Redirect(w, r, "/folder/permissions/"+id, http.StatusFound)
	}
}

func findAllFolders(db *DB) (*[]Folder, error) {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(col)
	var folders []Folder

	err := collection.Find(nil).Sort("name").All(&folders)
	if err != nil {
		return nil, err
	}

	return &folders, nil
}

func findAllPermittedFolders(db *DB) (*[]Folder, error) {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(col)
	var folders []Folder
	q := bson.M{}

	err := collection.Find(q).Sort("name").All(&folders)
	if err != nil {
		return nil, err
	}

	return &folders, nil
}

func findFolder(db *DB, idHex string) (*Folder, error) {
	id := bson.ObjectIdHex(idHex)
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(col)
	f := &Folder{}

	err := collection.FindId(id).One(f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func findFolderWithPermissions(db *DB, id bson.ObjectId) (*Folder, error) {
	var folder Folder
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(col)

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

func findFoldersAndDocuments(db *DB, user *User) (*[]Folder, error) {
	s := db.sess.Clone()
	defer s.Close()
	collection := s.DB(db.name).C(col)
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

func (f *Folder) save(db *DB) (err error) {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(col)

	_, err = collection.UpsertId(f.ID, f)

	return err
}

func (f *Folder) findDocsForFolder(db *DB) error {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(documentCol)
	var docs []Document

	err := collection.Find(bson.M{"folderID": f.ID}).All(&docs)
	f.Documents = docs

	return err
}
