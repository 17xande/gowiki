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
		models.ErrorLogger.Print("Error creating app DB instance.\n", err)
		err = nil
	}

	models.LoggerInit(db)

	mux := mux.NewRouter()
	mux.HandleFunc("/", models.IndexHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/notfound", models.NotFoundHandler(rend)).Methods("GET")
	mux.HandleFunc("/login", models.UserLoginHandler(db, rend))
	mux.HandleFunc("/logout", models.UserLogoutHandler).Methods("GET")
	mux.HandleFunc("/document/view/{id}", models.ViewHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/document/edit/{id}", models.EditHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/save/{id}", models.SaveHandler(db, rend)).Methods("POST")
	mux.HandleFunc("/users/", models.UserHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/user/edit/{id}", models.UserEditHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/user/save/{id}", models.UserSaveHandler(db, rend)).Methods("POST")
	mux.HandleFunc("/folders/", models.FoldersHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/folder/view/{id}", models.FolderHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/folder/edit/{id}", models.FolderEditHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/folder/edit/", models.FolderEditHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/folder/save/{id}", models.FolderSaveHandler(db, rend)).Methods("POST")
	mux.HandleFunc("/folder/permissions/{id}", models.FolderPermissionsEditHandler(db, rend)).Methods("GET")
	mux.HandleFunc("/folder/permissions/save/{id}", models.FolderPermissionsSaveHandler(db, rend)).Methods("POST")

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
}

func handleRecovery(err interface{}) {
	// lot fatal errors here and stuff.
	println("Fatal Error: ", err)
}
