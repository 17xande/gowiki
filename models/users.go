package models

import (
	"net/http"
	"path"

	"strconv"

	"golang.org/x/crypto/scrypt"
	"gopkg.in/mgo.v2/bson"
)

// User defines a user in the system
type User struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Name     string        `json:"name"`
	Email    string        `json:"email"`
	Password []byte        `json:"password"`
	Level    int           `json:"level"`
	Admin    bool          `json:"admin"`
	Tech     bool          `json:"tech"`
}

const userCol = "users"

// UserHandler handles any requests made to the user interface
func UserHandler(w http.ResponseWriter, r *http.Request) {
	users, err := findAllUsers()
	user := getUserFromSession()
	if err != nil {
		ErrorLogger.Print("Error trying to find all users: \n", err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"users": users,
		"user":  user,
	}

	RenderTemplate(w, r, "users", data)
}

// UserEditHandler handles the edit user page
func UserEditHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	editUser := &User{}
	exists := false
	user := getUserFromSession()

	_, id := path.Split(r.URL.Path)

	if len(id) > 0 {
		editUser, err = findUser(id)
		exists = true
	}

	if err != nil {
		ErrorLogger.Print("Error trying to find user {id: "+id+"}", err)
		UserSession.AddFlash("Error. User could not be retrieved.", "error")
	}

	tmpData := map[string]interface{}{
		"exists":       exists,
		"editUser":     editUser,
		"user":         user,
		"flashError":   UserSession.Flashes("error"),
		"flashWarning": UserSession.Flashes("warning"),
		"flashAlert":   UserSession.Flashes("alert"),
	}

	err = templates["userEdit.html"].ExecuteTemplate(w, "base", tmpData)
	if err != nil {
		ErrorLogger.Print("Error trying to display user {id: "+id+"}", err)
		UserSession.AddFlash("Error. User could not be displayed.", "error")
	}
}

// FirstUserHandler inserts a default user into the database
func FirstUserHandler(w http.ResponseWriter, r *http.Request) {
	// if Conf.Bools["setup"] {
	user := &User{
		ID:       bson.NewObjectId(),
		Name:     "Admin",
		Email:    "admin@email.com",
		Password: []byte("admin"),
		Level:    7,
		Admin:    true,
		Tech:     false,
	}

	user.hashPassword()
	user.saveUser()

	// Conf.Bools["setup"] = false
	// }
	http.Redirect(w, r, "/login", http.StatusFound)
}

// UserSaveHandler handles the save user page
func UserSaveHandler(w http.ResponseWriter, r *http.Request) {
	admin := r.FormValue("admin") == "on"
	tech := r.FormValue("admin") == "on"
	password := r.FormValue("password")
	level, err := strconv.Atoi(r.FormValue("level"))

	if err != nil {
		ErrorLogger.Print("Could not convert 'level' to int. ", err)
	}

	user := &User{
		Name:  r.FormValue("name"),
		Email: r.FormValue("email"),
		Level: level,
		Admin: admin,
		Tech:  tech,
	}

	// If no new password is supplied, don't change the old one.
	if password != "" {
		user.Password = []byte(password)
		err = user.hashPassword()
	}

	if err != nil {
		ErrorLogger.Print("Could not hash user's password. ", err)
	}

	// we can ignore the directory result of this function
	_, id := path.Split(r.URL.Path)

	if id != "" {
		user.ID = bson.ObjectIdHex(id)
	} else {
		user.ID = bson.NewObjectId()
	}

	err = user.saveUser()
	if err != nil {
		ErrorLogger.Print("Could not save user. ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	InfoLogger.Print("User saved: ", user.Name)
	http.Redirect(w, r, "/users/", http.StatusFound)
}

// UserLoginHandler handles login attempts
func UserLoginHandler(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	var err error
	var user *User
	var found bool

	if r.Method == "POST" {
		r.ParseForm()
		password := r.Form["password"][0]

		user = &User{
			Email:    r.Form["email"][0],
			Password: []byte(password),
		}

		err = user.hashPassword()
		if err != nil {
			ErrorLogger.Print("Could not hash password for user {email: "+user.Email+"} ", err)
			data["flashError"] = "There was a problem with your password. Please try again."
			_ = templates["login.html"].ExecuteTemplate(w, "base", data)
			return
		}

		found, err = user.authenticate()
		if err != nil {
			ErrorLogger.Print("Problem while looking for user in database. {email: "+user.Email+"} ", err)
			data["flashError"] = "Error trying to auntenticate your account. Please try again."
			_ = templates["login.html"].ExecuteTemplate(w, "base", data)
			return
		}

		if !found {
			data["flashWarning"] = "User not found"
			InfoLogger.Print("Failed user login attempt: {email: " + user.Email + "}")
		} else {
			InfoLogger.Print("Successful user login: {email: " + user.Email + "}")
			SessionCreate(w, r, user)
		}
	}

	// handle if a user is already logged in
	if UserSession != nil && UserSession.Values["id"] != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	err = templates["login.html"].ExecuteTemplate(w, "base", data)
	if err != nil {
		ErrorLogger.Print("Trouble handling login page render. ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// UserLogoutHandler handles logouts
func UserLogoutHandler(w http.ResponseWriter, r *http.Request) {
	SessionDelete(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func getUserFromSession() (user *User) {
	return &User{
		ID:    bson.ObjectIdHex(UserSession.Values["id"].(string)),
		Name:  UserSession.Values["name"].(string),
		Email: UserSession.Values["email"].(string),
		Level: UserSession.Values["level"].(int),
		Admin: UserSession.Values["admin"].(bool),
	}
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
	u := &User{}

	err := collection.FindId(id).One(u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func findUsers(ids *[]bson.ObjectId) (users *[]User, err error) {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(userCol)

	query := bson.M{"_id": bson.M{"$in": ids}}
	err = collection.Find(query).All(users)

	return nil, err
}

func (user *User) authenticate() (found bool, err error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(userCol)
	err = collection.Find(bson.M{
		"email":    user.Email,
		"password": user.Password,
	}).One(&user)

	found = user.ID.Hex() != ""
	return found, err
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

func (user *User) hashPassword() (err error) {
	var key []byte
	// TODO: Move the salt into a non-committed file so that it does not end up on github
	salt := []byte("You are the salt of the earth. But if the salt loses its saltiness, how can it be made salty again?" + user.Email)
	key, err = scrypt.Key(user.Password, salt, 16384, 8, 1, 32)
	user.Password = key
	return err
}

// func (user *User) in(userIDs []bson.ObjectId) bool {
// 	for _, id := range userIDs {
// 		if user.ID == id {
// 			return true
// 		}
// 	}
// 	return false
// }
