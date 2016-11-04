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
	Name     string        `json:"name" bson:"name"`
	Email    string        `json:"email" bson:"email"`
	Password []byte        `json:"password" bson:"password"`
	Level    int           `json:"level" bson:"level"`
	Admin    bool          `json:"admin" bson:"admin"`
}

// Permission defines additional user permissions for document
type Permission struct {
	// ID     bson.ObjectId `json:"id" bson:"_id"`
	UserID bson.ObjectId `json:"userId" bson:"userId"`
	DocID  bson.ObjectId `json:"docId" bson:"docId"`
	List   bool          `json:"list" bson:"list"`
	Read   bool          `json:"read" bson:"read"`
	Write  bool          `json:"write" bson:"write"`
}

// Save the current permission
func (p *Permission) save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C("permissions")
	IDs := bson.M{"userId": p.UserID, "docId": p.DocID}
	_, err := collection.Upsert(IDs, p)

	return err
}

func permissionsSave(userIds []string, docID bson.ObjectId) error {
	var err error
	usersLen := len(userIds)
	for i := 0; i < usersLen; i++ {
		p := &Permission{
			UserID: bson.ObjectIdHex(userIds[i]),
			DocID:  docID,
			List:   true,
			Read:   true,
			Write:  true,
		}

		p.save()
	}

	return err
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
	var editUser *User
	user := getUserFromSession()
	var err error
	exists := false

	_, id := path.Split(r.URL.Path)

	if len(id) > 0 {
		editUser, err = findUser(id)
		exists = true
	} else {
		editUser = &User{}
	}

	tmpData := map[string]interface{}{
		"editUser":     editUser,
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

// UserSaveHandler handles the save user page
func UserSaveHandler(w http.ResponseWriter, r *http.Request) {
	admin := r.FormValue("admin") == "on"
	level, err := strconv.Atoi(r.FormValue("level"))

	if err != nil {
		ErrorLogger.Print("Could not convert 'level' to int. ", err)
	}

	user := &User{
		Name:  r.FormValue("name"),
		Email: r.FormValue("email"),
		Level: level,
		Admin: admin,
	}

	err = user.hashPassword(r.FormValue("password"))

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

	// handle login form submit
	if r.FormValue("email") != "" && r.FormValue("password") != "" {
		user = &User{
			Email: r.FormValue("email"),
		}
		user.hashPassword(r.FormValue("password"))
		found, err = user.authenticate()
		if err != nil {
			ErrorLogger.Print("Problem while looking for user in database. ", err)
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

func findAllUsers() (users *[]User, err error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(userCol)
	err = collection.Find(nil).All(&users)
	if err != nil {
		return nil, err
	}

	return users, nil
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

func (user *User) hashPassword(pass string) (err error) {
	var key []byte
	// TODO: Move the salt into a non-committed file so that it does not end up on github
	salt := []byte("You are the salt of the earth. But if the salt loses its saltiness, how can it be made salty again?" + user.Email)
	key, err = scrypt.Key([]byte(user.Password), salt, 16384, 8, 1, 32)
	user.Password = key
	return err
}
