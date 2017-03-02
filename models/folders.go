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

		f, err = findFolder(id)
		if err != nil {
			ErrorLogger.Print("Error trying to find folder: {id: "+id+"}\n", err)
			s.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			s.Save(r, w)
			http.Redirect(w, r, "/folders/", http.StatusFound)
			err = nil
			return
		}

		err = f.findDocsForFolder()
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
		folders, err := findAllFolders()
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
			f, err = findFolder(id)
			exists = true
		}

		if err != nil {
			ErrorLogger.Print("Error trying to find folder {id: "+id+"}", err)
			s.AddFlash("Error. Folder could not be retrieved.", "error")
			s.Save(r, w)
			err = nil
		}

		users, err = findAllUsers()

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
		_, id := path.Split(r.URL.Path)

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

			err = f.save()

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

		f, err = findFolder(id)
		if err != nil {
			ErrorLogger.Print("Error trying to find folder {id: "+id+"} ", err)
			s.AddFlash(" Folder could not be retrieved.", "error")
			s.Save(r, w)
			http.Redirect(w, r, "/folder/"+id, http.StatusFound)
			err = nil
		}

		err = f.getPermissions()
		if err != nil {
			ErrorLogger.Print("Error trying to get permissions for folder {id: "+id+"} ", err)
			s.AddFlash(" Folder permissions could not be retrieved.", "error")
			s.Save(r, w)
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
			s.AddFlash("Couldn't retrieve rest of users from database", "error")
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

			err = permissionSave(p)
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

func findFoldersAndDocuments(user *User) (*[]Folder, error) {
	session := dbConnect()
	defer session.Close()
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

func permissionSave(ps []Permission) error {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C("folderPermissions")
	var update bson.M

	b := collection.Bulk()
	b.Unordered()
	for _, p := range ps {
		update = bson.M{"folderId": p.ID, "userId": p.UserID}
		b.Upsert(update, p)
	}

	_, err := b.Run()

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
