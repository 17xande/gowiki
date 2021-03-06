package models

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/unrolled/render"

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
	Password []byte        `json:"-"`
}

const userCol = "users"

// UserHandler handles any requests made to the user interface
func UserHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := findAllUsers(db)
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
			"users": users,
			"user":  user,
		}

		RenderTemplate(rend, w, r, "user/index", data)
	}
}

// UserEditHandler handles the edit user page
func UserEditHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		editUser := &User{}
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

		page := "user/edit"

		vars := mux.Vars(r)
		id := vars["id"]

		if len(id) > 0 {
			editUser, err = findUser(db, id)
			exists = true

			if err != nil {
				ErrorLogger.Print("Error trying to find user {id: "+id+"}", err)
				s.AddFlash("Error. User could not be retrieved.", "error")
				s.Save(r, w)
				err = nil
			} else if editUser.ID == user.ID { // check if the user is editing his own account
				page = "account"
			}
		}

		data := map[string]interface{}{
			"exists":   exists,
			"editUser": editUser,
			"user":     user,
			"page":     page,
		}

		RenderTemplate(rend, w, r, "user/edit", data)
	}
}

// UserSaveHandler handles the save user page
func UserSaveHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var exists bool
		var u *User

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

			vars := mux.Vars(r)
			id := vars["id"]

			if id != "" { // existing user
				u.ID = bson.ObjectIdHex(id)
			} else { // new user
				// check if user already exists in the database
				exists, err = u.exists(db)
				if err != nil {
					ErrorLogger.Print("Error lookin for duplicate user: {name: "+u.Name+", email: "+u.Email+"}", err)
					err = nil
				}

				if exists {
					s.AddFlash("Sorry, a user with this name or email already exists.", "warning")
					s.Save(r, w)
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

			err = u.Save(db)
			if err != nil {
				ErrorLogger.Print("Could not save user. ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			InfoLogger.Print("User saved: {id: " + u.ID.Hex() + "}")
			s.AddFlash("User saved successfully", "success")
			s.Save(r, w)
		}

		if user.Admin {
			http.Redirect(w, r, "/users/edit/", http.StatusFound)
		} else {
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}
}

// UserLoginHandler handles login attempts
func UserLoginHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]interface{})
		var err error
		var user *User
		var found bool

		// Get the user session from the context.
		ctx := r.Context()
		t := ctx.Value(sessKey)
		s, ok := t.(*sessions.Session)
		if !ok {
			err := errors.New("Error retrieving the session from context.\n")
			ErrorLogger.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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
				RenderTemplate(rend, w, r, "login", data)
				return
			}

			found, err = user.Authenticate(db)
			if err != nil {
				ErrorLogger.Print("Problem while looking for user in database. {email: "+user.Email+"} ", err)
				data["flashError"] = "Error trying to auntenticate your account. Please try again."
				RenderTemplate(rend, w, r, "login", data)
				return
			}

			if !found {
				data["flashWarning"] = "User not found"
				InfoLogger.Print("Failed user login attempt: {email: " + user.Email + "}")
			} else {
				InfoLogger.Print("Successful user login: {email: " + user.Email + "}")

				s.Values["id"] = user.ID.Hex()
				s.Values["name"] = user.Name
				s.Values["email"] = user.Email
				s.Values["level"] = user.Level
				s.Values["admin"] = user.Admin
				s.Save(r, w)
			}
		}

		// handle if a user is already logged in
		if s != nil && s.Values["id"] != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		RenderTemplate(rend, w, r, "login", data)
	}
}

// UserLogoutHandler handles logouts
func UserLogoutHandler(w http.ResponseWriter, r *http.Request) {
	SessionDelete(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func getUserFromSession(s *sessions.Session) (user *User, ok bool) {
	if s.Values["id"] == nil {
		ok = false
		err := errors.New("Error retrieving user from session.\n")
		ErrorLogger.Print(err)
		return nil, ok
	}

	ok = true
	return &User{
		ID:    bson.ObjectIdHex(s.Values["id"].(string)),
		Name:  s.Values["name"].(string),
		Email: s.Values["email"].(string),
		Level: s.Values["level"].(int),
		Admin: s.Values["admin"].(bool),
	}, ok
}

func findAllUsers(db *DB) (*[]User, error) {
	session := db.sess.Clone()
	defer session.Close()

	collection := session.DB(db.name).C(userCol)
	var users []User
	err := collection.Find(nil).Sort("name").All(&users)
	if err != nil {
		return nil, err
	}

	return &users, nil
}

func findUser(db *DB, idHex string) (*User, error) {
	id := bson.ObjectIdHex(idHex)
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(userCol)
	u := &User{}

	err := collection.FindId(id).One(u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func findUsers(db *DB, ids *[]bson.ObjectId) (users *[]User, err error) {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(userCol)

	query := bson.M{"_id": bson.M{"$in": ids}}
	err = collection.Find(query).Sort("name").All(users)

	return nil, err
}

func findNotUsers(db *DB, ids *[]bson.ObjectId) (*[]User, error) {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(userCol)
	users := &[]User{}

	query := bson.M{"_id": bson.M{"$nin": ids}}
	err := collection.Find(query).Sort("name").All(users)

	if err != nil {
		return nil, err
	}

	return users, nil
}

// Authenticate user based on email and password
func (user *User) Authenticate(db *DB) (found bool, err error) {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(userCol)

	err = collection.Find(bson.M{
		"email":    user.Email,
		"password": user.Password,
	}).One(&user)

	found = user.ID.Hex() != ""
	if err != nil && err.Error() == "not found" {
		err = nil
		found = false
	}
	return found, err
}

// Save or update the database record of the User
func (user *User) Save(db *DB) error {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(userCol)

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
func (user *User) exists(db *DB) (exists bool, err error) {
	session := db.sess.Clone()
	defer session.Close()
	collection := session.DB(db.name).C(userCol)
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

// InitAdmin inserts or updates the default Admin user.
func InitAdmin() (*User, error) {
	u := User{
		ID:       bson.NewObjectId(),
		Name:     "Admin",
		Email:    "admin@email.com",
		Password: []byte("admin"),
		Level:    7,
		Admin:    true,
		Tech:     false,
	}

	err := u.hashPassword()
	if err != nil {
		return nil, err
	}

	return &u, err
}
