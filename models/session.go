package models

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/urfave/negroni"
)

var store = sessions.NewCookieStore([]byte("scms-foh-the-win"))

type ctxKey string

const sessKey ctxKey = "session"

// UserSession stores the current user session
// var UserSession *sessions.Session

// SessionInit initialises the sessio options
func SessionInit() {
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400, // One day
		HttpOnly: true,
	}
}

// SessionMiddleware handles session checking in the Negroni flow.
func SessionMiddleware(db *DB) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		ctx := r.Context()
		// store.Get() creates a new entry if one doesn't already exists. So it will almost never throw and error.
		s, err := store.Get(r, "user")
		if err != nil {
			ErrorLogger.Print("Error trying to process session. ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			next(w, r)
			return
		}

		if s.Values["id"] == nil {
			// if we're already in the login page then don't redirect to the login page again.
			if r.URL.Path != "/login" {
				http.Redirect(w, r.WithContext(ctx), "/login", http.StatusFound)
			}
		}

		ctx = context.WithValue(ctx, sessKey, s)
		next(w, r.WithContext(ctx))
	})
}

// SessionDelete removes the current user session
func SessionDelete(w http.ResponseWriter, r *http.Request) {
	sess, _ := store.Get(r, "user")
	sess.Options.MaxAge = -1
	sess.Save(r, w)
}

func levelCheck(w http.ResponseWriter, r *http.Request, d *Document) bool {
	ctx := r.Context()
	s, ok := ctx.Value(sessKey).(*sessions.Session)
	if !ok {
		err := errors.New("Error retrieving the session from context.\n")
		ErrorLogger.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	level := s.Values["level"].(int)

	if level < d.Level {
		s.AddFlash("Sorry. You're not allowed to view that page", "warning")
		s.Save(r, w)
		return false
	}

	return true
}

// SessionHandler redirects to the login page if no session is found
// func SessionHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		getSession(w, r)

// 		if UserSession.Values["id"] == nil {
// 			http.Redirect(w, r, "/login", http.StatusFound)
// 			return
// 		}

// 		fn(w, r)
// 	}
// }

// // getSession Retrieves session if there is one
// func getSession(w http.ResponseWriter, r *http.Request) {
// 	// Get a session - Get() always returns a session, even if empty
// 	var err error
// 	UserSession, err = store.Get(r, "user")
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }
