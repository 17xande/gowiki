package main

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte("scms-foh-the-win"))

// SessionHandler handles login sessions in the site
func SessionHandler(w http.ResponseWriter, r *http.Request, user User) {
	// Get a session - Get() always returns a session, even if empty
	session, err := store.Get(r, "login")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set some session values
	session.Values["user"] = user
	session.Save(r, w)
}

// LoginTest redirects to the login page if no session is found
func LoginTest(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "login")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if session.Values["user"] == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		fn(w, r)
	}
}
