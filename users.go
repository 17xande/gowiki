package main

import (
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

// User defines a user in the system
type User struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Name     string        `json:"name" bson:"name"`
	Email    string        `json:"email" bson:"email"`
	Password string        `json:"password" bson:"password"`
}

const col = "users"

// UserHandler handles any requests made to the user interface
func userHandler(w http.ResponseWriter, r *http.Request) {
	users, err := findAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = templates["users.html"].ExecuteTemplate(w, "base", users)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func userEditHandler(w http.ResponseWriter, r *http.Request) {
	err := templates["userNew.html"].ExecuteTemplate(w, "base", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func findAllUsers() (*[]User, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(col)
	var users []User
	err := collection.Find(nil).All(&users)
	if err != nil {
		return nil, err
	}

	return &users, nil
}

func findUser(idHex string) (*User, error) {
	id := bson.ObjectIdHex(idHex)
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(col)
	var user User
	err := collection.Find(bson.M{"_id": id}).One(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (user *User) saveUser() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(col)

	info, err := collection.UpsertId(user.ID, &user)

	// If there was no user id, grab the DB generated id
	if len(user.ID.Hex()) == 0 {
		user.ID = info.UpsertedId.(bson.ObjectId)
	}

	return err
}
