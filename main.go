package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/17xande/gowiki/models"

	"github.com/gorilla/context"
)

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9])$")

func init() {
	models.SessionInit()
}

func main() {
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public"))))
	http.HandleFunc("/", models.SessionHandler(models.IndexHandler))
	http.HandleFunc("/login", models.UserLoginHandler)
	http.HandleFunc("/logout", models.UserLogoutHandler)
	http.HandleFunc("/view/", models.SessionHandler(models.MakeHandler(models.ViewHandler)))
	http.HandleFunc("/edit/", models.SessionHandler(models.MakeHandler(models.EditHandler)))
	http.HandleFunc("/save/", models.SessionHandler(models.MakeHandler(models.SaveHandler)))
	http.HandleFunc("/users/", models.SessionHandler(models.UserHandler))
	http.HandleFunc("/users/edit/", models.SessionHandler(models.UserEditHandler))
	http.HandleFunc("/users/save/", models.SessionHandler(models.UserSaveHandler))
	// Handlers without security for adding the first user
	// http.HandleFunc("/users/edit/", models.UserEditHandler)
	// http.HandleFunc("/users/save/", models.UserSaveHandler)
	http.HandleFunc("/folders/", models.SessionHandler(models.FolderHandler))
	http.HandleFunc("/folders/edit/", models.SessionHandler(models.FolderEditHandler))
	http.HandleFunc("/folders/save/", models.SessionHandler(models.FolderSaveHandler))

	if models.Conf.Bools["setup"] {
		http.HandleFunc("/setup/", models.FirstUserHandler)
	}

	p := os.Getenv("PORT")
	if p == "" {
		p = ":8080"
	}

	fmt.Println("Server listening on port", p)
	err := http.ListenAndServe(p, context.ClearHandler(http.DefaultServeMux))

	if err != nil {
		fmt.Println("Error: Could not start server\n", err)
		models.ErrorLogger.Print("Could not start server ", err)
	}
}
