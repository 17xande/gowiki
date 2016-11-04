package models

import (
	"fmt"
	"log"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	db string
	// ErrorLogger logs errors into the database
	ErrorLogger *log.Logger
	// InfoLogger logs infos into the database
	InfoLogger *log.Logger
)

func init() {
	Conf = &Config{}
	Conf.Load()
	db = Conf.Databases["app"].Name

	ErrorLogger = log.New(&MongoWriter{"error"}, "", log.Lshortfile)
	InfoLogger = log.New(&MongoWriter{"info"}, "", log.Lshortfile)
}

// MongoDB is an implementation that talks to an external mongodb
// type MongoDB struct {
// 	sess *mgo.Session
// }

// DB defines the database config options
type DB struct {
	Host     string
	Name     string
	Username string
	Password string
}

// MongoWriter writes logs to the database
type MongoWriter struct {
	collection string
}

// func NewMongoDB(dbConf DB) (*MongoDB, error) {
// 	s, err := mgo.Dial(dbConf.Host)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &MongoDB{sess: s}, nil
// }

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

	sess := dbConnect()
	defer sess.Close()

	c := sess.DB(Conf.Databases["log"].Name).C(mw.collection)
	err = c.Insert(data)

	if err != nil {
		return 0, err
	}

	// Print to console for debugging
	fmt.Println(string(p))

	return len(p), nil
}

func dbConnect() *mgo.Session {
	session, err := mgo.Dial(Conf.Databases["app"].Host)
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	return session
}
