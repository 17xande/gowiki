package main

import (
	"fmt"
	"log"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const host = "localhost"
const db = "scms"
const dbLog = "scms_log"

var (
	errorLogger *log.Logger
	infoLogger  *log.Logger
)

// DB defines the database config options
type DB struct {
	Host     string
	Name     string
	Username string
	Password string
}

// MongoWriter writes logs to the database
type MongoWriter struct {
	sess       *mgo.Session
	collection string
}

func (mw *MongoWriter) Write(p []byte) (n int, err error) {
	var data bson.M

	if mw.collection == "error" { // error logging
		// TODO: implement stacktrace logging
		data = bson.M{
			"timestamp": time.Now(),
			"msg":       string(p),
			"stack":     "",
		}
	} else { // info logging
		data = bson.M{
			"timestamp": time.Now(),
			"msg":       string(p),
		}
	}

	c := mw.sess.DB(dbLog).C(mw.collection)
	err = c.Insert(data)

	if err != nil {
		return 0, err
	}

	// Print to console for debugging
	fmt.Println(string(p))

	return len(p), nil
}

func dbConnect() *mgo.Session {
	session, err := mgo.Dial(host)
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	return session
}
