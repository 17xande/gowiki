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
	Level    int           `json:"level"`
	Admin    bool          `json:"admin"`
	Tech     bool          `json:"tech"`
	Password []byte        `json:"password"`
}

const userCol = "users"

// UserHandler handles any requests made to the user interface
func UserHandler(w http.ResponseWriter, r *http.Request) {
	users, err := findAllUsers()
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
	page := "userEdit"

	_, id := path.Split(r.URL.Path)

	if len(id) > 0 {
		editUser, err = findUser(id)
		exists = true

		if err != nil {
			ErrorLogger.Print("Error trying to find user {id: "+id+"}", err)
			UserSession.AddFlash("Error. User could not be retrieved.", "error")
			UserSession.Save(r, w)
			err = nil
		} else if editUser.ID == user.ID { // check if the user is editing his own account
			page = "account"
		}
	}

	tmpData := map[string]interface{}{
		"exists":   exists,
		"editUser": editUser,
		"user":     user,
		"page":     page,
	}

	RenderTemplate(w, r, "userEdit", tmpData)
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
	// user.insert()

	// Conf.Bools["setup"] = false
	// }
	http.Redirect(w, r, "/login", http.StatusFound)
}

// UserSaveHandler handles the save user page
func UserSaveHandler(w http.ResponseWriter, r *http.Request) {
	var exists bool
	var u *User
	user := getUserFromSession()

	if r.Method == "POST" {
		r.ParseForm()
		name := r.Form["name"][0]
		email := r.Form["email"][0]
		password := r.Form["password"][0]
		admin := len(r.Form["admin"]) > 0 && r.Form["admin"][0] == "on"
		tech := len(r.Form["tech"]) > 0 && r.Form["tech"][0] == "on"
		level, err := strconv.Atoi(r.Form["level"][0])

		if err != nil {
			ErrorLogger.Print("Could not convert 'level' to int. ", err)
			level = 0
			err = nil
		}

		u = &User{
			Name:  name,
			Email: email,
			Level: level,
			Admin: admin,
			Tech:  tech,
		}

		_, id := path.Split(r.URL.Path)
		if id != "" { // existing user
			u.ID = bson.ObjectIdHex(id)
		} else { // new user
			// check if user already exists in the database
			exists, err = u.exists()
			if err != nil {
				ErrorLogger.Print("Error lookin for duplicate user: {name: "+u.Name+", email: "+u.Email+"}", err)
				err = nil
			}

			if exists {
				UserSession.AddFlash("Sorry, a user with this name or email already exists.", "warning")
				UserSession.Save(r, w)
				InfoLogger.Print("User tried to add a duplicate user: {name: " + u.Name + ", email: " + u.Email + "}")
				http.Redirect(w, r, "/users/edit/", http.StatusFound)
				return
			}

			u.ID = bson.NewObjectId()
		}

		// If no new password is supplied, don't change the old one.
		if password != "" {
			u.Password = []byte(password)
			err := u.hashPassword()

			if err != nil {
				ErrorLogger.Print("Could not hash user's password. ", err)
			}
		}

		err = u.save()
		if err != nil {
			ErrorLogger.Print("Could not save user. ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		InfoLogger.Print("User saved: {id: " + u.ID.Hex() + "}")
		UserSession.AddFlash("User saved successfully", "success")
		UserSession.Save(r, w)
	}

	if user.Admin {
		http.Redirect(w, r, "/users/edit/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
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
	err := collection.Find(nil).Sort("name").All(&users)
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

func (user *User) save() error {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(userCol)

	update := bson.M{
		"name":  user.Name,
		"email": user.Email,
		"level": user.Level,
		"admin": user.Admin,
		"tech":  user.Tech,
	}

	if len(user.Password) > 0 {
		update["password"] = user.Password
	}

	_, err := collection.UpsertId(user.ID, bson.M{"$set": update})

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

// Checks if a user with this name or email address already exists.
func (user *User) exists() (exists bool, err error) {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(userCol)
	count := 0

	query := bson.M{
		"$or": []bson.M{
			bson.M{"name": user.Name},
			bson.M{"email": user.Email},
		},
	}
	count, err = collection.Find(query).Count()
	exists = count > 0

	return exists, err
}
