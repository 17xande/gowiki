package models

import (
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/urfave/negroni"
	"gopkg.in/mgo.v2/bson"
)

const permissionCol = "folderPermissions"

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

// PermissionMiddleware handles permission checks in the Negroni flow.
func PermissionMiddleware(db *DB) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		ctx := r.Context()
		s, ok := ctx.Value(sessKey).(*sessions.Session)
		if !ok {
			err := errors.New("Error retrieving the session from context.\n")
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			next(w, r)
			return
		}

		user, ok := getUserFromSession(s)
		if !ok {
			next(w, r)
			return
		}

		session := db.sess.Clone()
		defer session.Close()
		collection := session.DB(db.name).C(permissionCol)

		p := &Permission{}
		q := bson.M{
			"folderId": 0,
			"userId":   user.ID,
		}

		err := collection.Find(q).One(p)
		if err != nil {
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			next(w, r)
			return
		}

		next(w, r.WithContext(ctx))
	})
}

func (f *Folder) getPermissions(db *DB) error {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(permissionCol)

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

func permissionSave(db *DB, ps []Permission) error {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(permissionCol)
	var update bson.M

	b := collection.Bulk()
	b.Unordered()
	for _, p := range ps {
		if p.ID.Hex() == "" {
			p.ID = bson.NewObjectId()
		}

		update = bson.M{
			"folderId": p.FolderID,
			"userId":   p.UserID,
		}
		b.Upsert(update, p)
	}

	_, err := b.Run()

	return err
}
