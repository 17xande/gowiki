package main

import (
	"net/http"
	"path"

	"strconv"

	"gopkg.in/mgo.v2/bson"
)

// User defines a user in the system
type User struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Name     string        `json:"name" bson:"name"`
	Email    string        `json:"email" bson:"email"`
	Password string        `json:"password" bson:"password"`
	Level    int           `json:"level" bson:"level"`
	Admin    bool          `json:"admin" bson:"admin"`
}

const userCol = "users"

// UserHandler handles any requests made to the user interface
func userHandler(w http.ResponseWriter, r *http.Request) {
	users, err := findAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	data := map[string]interface{}{
		"users": users,
	}

	renderTemplate(w, r, "users", data)
}

func userEditHandler(w http.ResponseWriter, r *http.Request) {
	var user *User
	var err error
	exists := false

	_, id := path.Split(r.URL.Path)

	if len(id) > 0 {
		user, err = findUser(id)
		exists = true
	} else {
		user = &User{}
	}

	tmpData := map[string]interface{}{
		"user":         user,
		"exists":       exists,
		"flashError":   UserSession.Flashes("error"),
		"flashWarning": UserSession.Flashes("warning"),
		"flashAlert":   UserSession.Flashes("alert"),
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		panic(err)
	}

	err = templates["userEdit.html"].ExecuteTemplate(w, "base", tmpData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func userSaveHandler(w http.ResponseWriter, r *http.Request) {
	admin := r.FormValue("admin") == "on"
	level, err := strconv.Atoi(r.FormValue("level"))

	user := &User{
		Name:     r.FormValue("name"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
		Level:    level,
		Admin:    admin,
	}

	_, id := path.Split(r.URL.Path)
	// id := r.FormValue("userId")

	if id != "" {
		user.ID = bson.ObjectIdHex(id)
	} else {
		user.ID = bson.NewObjectId()
	}

	err = user.saveUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/users/", http.StatusFound)
}

func userLoginHandler(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})

	if r.FormValue("email") != "" {
		user, err := authenticateUser(r.FormValue("email"), r.FormValue("password"))
		if err != nil {
			data["flashWarning"] = "User not found"
		} else {
			SessionCreate(w, r, user)
		}
	}

	if UserSession != nil && UserSession.Values["id"] != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	err := templates["login.html"].ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func userLogoutHandler(w http.ResponseWriter, r *http.Request) {
	SessionDelete(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func findAllUsers() (*[]User, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(userCol)
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

	collection := session.DB(db).C(userCol)
	var user User
	err := collection.FindId(id).One(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func authenticateUser(email string, password string) (*User, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(userCol)
	var user User
	err := collection.Find(bson.M{"email": email, "password": password}).One(&user)

	// if there's an error or no user was found
	if err != nil || user.ID.Hex() == "" {
		return nil, err
	}

	return &user, nil
}

func (user *User) saveUser() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(userCol)

	info, err := collection.UpsertId(user.ID, &user)

	// If there was no user id, grab the DB generated id
	if len(user.ID.Hex()) == 0 {
		user.ID = info.UpsertedId.(bson.ObjectId)
	}

	return err
}
