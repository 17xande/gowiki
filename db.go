package main

import (
	mgo "gopkg.in/mgo.v2"
)

const host = "localhost"
const db = "rivers"

var db1 DB

// DB defines the database config options
type DB struct {
	Host     string
	Name     string
	Username string
	Password string
}

func dbConnect() *mgo.Session {
	session, err := mgo.Dial(host)
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	return session
}
