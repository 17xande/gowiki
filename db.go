package main

import mgo "gopkg.in/mgo.v2"

const host = "localhost"
const db = "rivers"

func dbConnect() *mgo.Session {
	session, err := mgo.Dial(host)
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	return session
}
