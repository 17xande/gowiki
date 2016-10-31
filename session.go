package main

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte("scms-foh-the-win"))

// UserSession stores the current user session
var UserSession *sessions.Session

// SessionInit initialises the sessio options
func SessionInit() {
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400, // One day
		HttpOnly: true,
	}
}

// SessionCreate handles login sessions in the site
func SessionCreate(w http.ResponseWriter, r *http.Request, user *User) {
	getSession(w, r)

	// Set some session values
	UserSession.Values["id"] = user.ID.Hex()
	UserSession.Values["name"] = user.Name
	UserSession.Values["email"] = user.Email
	UserSession.Values["level"] = user.Level
	UserSession.Values["admin"] = user.Admin
	UserSession.Save(r, w)
}

// SessionDelete removes the current user session
func SessionDelete(w http.ResponseWriter, r *http.Request) {
	sess, _ := store.Get(r, "user")
	sess.Options.MaxAge = -1
	sess.Save(r, w)
}

func levelCheck(w http.ResponseWriter, r *http.Request, p *Page) bool {
	level := UserSession.Values["level"].(int)

	if level < p.Level {
		UserSession.AddFlash("Sorry. You're not allowed to view that page", "warning")
		UserSession.Save(r, w)
		return false
	}

	return true
}

// SessionHandler redirects to the login page if no session is found
func SessionHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getSession(w, r)

		if UserSession.Values["id"] == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		fn(w, r)
	}
}

// LoginTest is a non-handler verson of LoginTestHandler
// func LoginTest(w http.ResponseWriter, r *http.Request) {
// 	getSession(w, r)

// 	if UserSession.Values["id"] == nil {
// 		http.Redirect(w, r, "/login", http.StatusFound)
// 	}
// }

// getSession Retrieves session if there is one
func getSession(w http.ResponseWriter, r *http.Request) {
	// Get a session - Get() always returns a session, even if empty
	var err error
	UserSession, err = store.Get(r, "user")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
