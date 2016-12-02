package main

import (
	"fmt"

	"github.com/17xande/gowiki/models"
)

func main() {
	fmt.Println("Creating default admin user...")

	a, err := models.InitAdmin()
	if err != nil {
		fmt.Println("Error attempting to create default Admin user:\n", err)
	}

	fmt.Println("Authenticating Admin account...")
	found, err := a.Authenticate()
	if err != nil {
		fmt.Println("Error authenticating admin account.:\n", err)
		return
	}
	if !found {
		fmt.Println("Default Admin account not found, adding it now...")
		err := a.Save()
		if err != nil {
			fmt.Println("Could not save default Admin user to the database.\n", err)
			return
		}
	}

	fmt.Println("Default Admin user has been created or reset.")
}
