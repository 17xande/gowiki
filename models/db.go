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

// DB is a DB implementation that talks to an external DB server.
// This will connect to a MongoDB server but a generic "DB" is used
// to make it simple to switch to a different database if necessary.
type DB struct {
	sess *mgo.Session
}

// DBConf defines the database config options
type DBConf struct {
	Host     string
	Name     string
	Username string
	Password string
}

// MongoWriter writes logs to the database
type MongoWriter struct {
	collection string
}

func init() {
	Conf = &Config{}
	Conf.Load()
	db = Conf.Databases["app"].Name

	ErrorLogger = log.New(&MongoWriter{"error"}, "", log.Lshortfile)
	InfoLogger = log.New(&MongoWriter{"info"}, "", log.Lshortfile)
}

// NewDB connects to a database server and returns a database implementation
// for that connection. Call Close() on the returned value when done.
func NewDB(dbc DBConf) (*DB, error) {
	// url := "mongodb://" + dbc.Username + ":" + dbc.Password + "@" + dbc.Host + "/" + dbc.Name
	url := "mongodb://" + dbc.Host + "/" + dbc.Name
	s, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	return &DB{sess: s}, nil
}

// Close releases the underlying connections. Always call this when done
// with the database operations.
func (d *DB) Close() {
	d.sess.Close()
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
