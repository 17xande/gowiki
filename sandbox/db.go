package main

import (
	"fmt"
	"log"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Person struct {
	Name  string
	Phone string
}

func main() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}

	// A defer statement pushers a function onto a list.
	// The list of saved calls is executed after the surrounding function returns.
	// Commonly used for clean-up operations.
	defer session.Close()

	// Optional. Switch the session to a monotonic behaviour.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("test").C("people")
	err = c.Insert(&Person{"Ale", "+55 234 5678"},
		&Person{"Cla", "+55 098 7654"})
	if err != nil {
		log.Fatal(err)
	}

	result := Person{}
	err = c.Find(bson.M{"name": "Ale"}).One(&result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Phone:", result.Phone)
}
