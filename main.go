package main

import (
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/17xande/gowiki/models"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
)

func init() {
	models.SessionInit()
}

func main() {
	funcMap := []template.FuncMap{{
		"mod0": func(i int, mod int) bool {
			return i%mod == 0
		},
		"timeFormat": func(d time.Time) string {
			return d.Format(time.RFC822)
		},
	}}
	rend := render.New(render.Options{
		Directory:     "templates",
		Layout:        "layout",
		Extensions:    []string{".html"},
		Funcs:         funcMap,
		IsDevelopment: true,
	})

	cfg := &models.Config{}
	cfg.Load()
	db, err := models.NewDB(cfg.Databases["app"])
	if err != nil {
		models.ErrorLogger.Print("Error creating DB instance.\n", err)
		err = nil
	}

	mux := mux.NewRouter()
	mux.HandleFunc("/", models.IndexHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/notfound", models.NotFoundHandler).Methods("GET")
	mux.HandleFunc("/login", models.UserLoginHandler(rend))
	mux.HandleFunc("/logout", models.UserLogoutHandler).Methods("GET")
	mux.HandleFunc("/view", models.ViewHandler).Methods("GET")
	mux.HandleFunc("/edit", models.EditHandler).Methods("GET")
	mux.HandleFunc("/save", models.SaveHandler).Methods("POST")
	mux.HandleFunc("/users", models.UserHandler).Methods("GET")
	mux.HandleFunc("/users/edit", models.UserEditHandler).Methods("GET")
	mux.HandleFunc("/users/save", models.UserSaveHandler).Methods("POST")
	mux.HandleFunc("/folders", models.FoldersHandler).Methods("GET")
	mux.HandleFunc("/folder/view", models.FolderHandler).Methods("GET")
	mux.HandleFunc("/folder/edit", models.FolderEditHandler).Methods("GET")
	mux.HandleFunc("/folder/save", models.FolderSaveHandler).Methods("POST")
	mux.HandleFunc("/folder/permissions", models.FolderPermissionsEditHandler).Methods("POST")
	mux.HandleFunc("/folder/permissions/save", models.FolderPermissionsSaveHandler).Methods("POST")

	n := negroni.New()
	recovery := negroni.NewRecovery()
	recovery.ErrorHandlerFunc = handleRecovery

	n.Use(recovery)
	n.Use(negroni.NewLogger())
	n.Use(negroni.NewStatic(http.Dir("./public")))
	n.Use(models.SessionMiddleware(db))
	n.UseHandler(mux)

	p := os.Getenv("PORT")
	if p == "" {
		p = ":8080"
	}

	s := &http.Server{
		Addr:           p,
		Handler:        n,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	println("Sever listening on port " + p)
	err = s.ListenAndServe()
	if err != nil {
		models.ErrorLogger.Print("Could not start server ", err)
	}

	// http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	// http.HandleFunc("/notfound/", models.NotFoundHandler)
	// http.HandleFunc("/", models.SessionHandler(models.IndexHandler))
	// http.HandleFunc("/login", models.UserLoginHandler)
	// http.HandleFunc("/logout", models.UserLogoutHandler)
	// http.HandleFunc("/view/", models.SessionHandler(models.MakeHandler(models.ViewHandler)))
	// http.HandleFunc("/edit/", models.SessionHandler(models.MakeHandler(models.EditHandler)))
	// http.HandleFunc("/save/", models.SessionHandler(models.MakeHandler(models.SaveHandler)))
	// http.HandleFunc("/users/", models.SessionHandler(models.UserHandler))
	// http.HandleFunc("/users/edit/", models.SessionHandler(models.UserEditHandler))
	// http.HandleFunc("/users/save/", models.SessionHandler(models.UserSaveHandler))
	// Handlers without security for adding the first user
	// http.HandleFunc("/users/edit/", models.UserEditHandler)
	// http.HandleFunc("/users/save/", models.UserSaveHandler)
	// http.HandleFunc("/folders/", models.SessionHandler(models.FoldersHandler))
	// http.HandleFunc("/folder/view/", models.SessionHandler(models.FolderHandler))
	// http.HandleFunc("/folder/edit/", models.SessionHandler(models.FolderEditHandler))
	// http.HandleFunc("/folder/save/", models.SessionHandler(models.FolderSaveHandler))
	// http.HandleFunc("/folder/permissions/", models.SessionHandler(models.FolderPermissionEditHandler))
	// http.HandleFunc("/folder/permissions/save/", models.SessionHandler(models.FolderPermissionSaveHandler))

}

func handleRecovery(err interface{}) {
	// lot fatal errors here and stuff.
	println("Fatal Error: ", err)
}
