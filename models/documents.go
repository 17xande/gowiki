package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"html/template"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/unrolled/render"

	"golang.org/x/crypto/scrypt"

	"gopkg.in/mgo.v2/bson"
)

// Document represents a document on the site
type Document struct {
	ID       bson.ObjectId   `json:"id" bson:"_id"`
	Title    string          `json:"title"`
	Body     []byte          `json:"body"`
	URL      string          `json:"url"`
	Level    int             `json:"level"`
	Created  time.Time       `json:"created"`
	Edited   time.Time       `json:"edited"`
	FolderID bson.ObjectId   `json:"folderID" bson:"folderID,omitempty"`
	UserIDs  []bson.ObjectId `json:"userIDs" bson:"userIDs"`
}

const documentCol = "documents"
const keyPlain = "Take these documents, both the sealed and unsealed copies of the deed of purchase, and put them in a clay jar so they will last a long time."

var keyHash []byte

func init() {
	var err error
	keyHash, err = scrypt.Key([]byte(keyPlain), []byte("verse"), 16384, 8, 1, 32)
	if err != nil {
		ErrorLogger.Print("Could not hash document crypto key.\n ", err)
	}
}

// Save saves the page to the database
func (d *Document) save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(documentCol)
	_, err := collection.UpsertId(d.ID, d)
	return err
}

// LoadPage loads retrieves the page data from the database
func loadPage(idHex string) (*Document, error) {
	id := bson.ObjectIdHex(idHex)
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(documentCol)
	d := &Document{}
	err := collection.FindId(id).One(d)

	if err != nil {
		return nil, err
	}
	return d, nil
}

func findAllDocs() (*[]Document, error) {
	session := dbConnect()
	defer session.Close()
	collection := session.DB(db).C(documentCol)
	var documents []Document

	err := collection.Find(nil).Sort("title").All(&documents)
	if err != nil {
		return nil, err
	}

	return &documents, nil
}

// IndexHandler handles the index page request
func IndexHandler(db *DB, rend *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// check for invalid request
		if r.URL.Path != "/" {
			http.Redirect(w, r, "/notfound/", http.StatusNotFound)
			return
		}

		user := getUserFromSession()
		folders, err := findFoldersAndDocuments()
		if err != nil {
			ErrorLogger.Print("Error getting users and folders on index page. ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"folders": folders,
			"user":    user,
		}

		RenderTemplate(w, r, "index", data)
	}
}

// ViewHandler handles the document view page
func ViewHandler(w http.ResponseWriter, r *http.Request) {
	_, id := path.Split(r.URL.Path)
	var body template.HTML
	d, err := loadPage(id)
	if err != nil {
		ErrorLogger.Print("Document not found. id: "+id, err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		UserSession.Save(r, w)
		err = nil
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	user := getUserFromSession()

	if !levelCheck(w, r, d) {
		UserSession.AddFlash("Sorry, but you don't have permission to view this document", "warning")
		UserSession.Save(r, w)
		InfoLogger.Print("User tried to access restricted document: {userID: " + user.ID.Hex() + ", documentID: " + id + "}")
		http.Redirect(w, r, "/", http.StatusFound)
	}

	if user.Tech {
		body = "<p>This is a sample text that tech users can see.</p><p>Shalom.</p>"
	} else {
		body, err = d.decrypt()
		if err != nil {
			InfoLogger.Print("Could not decrypt page id: "+id+"\nDisplaying blank body", err)
			UserSession.AddFlash("There was a problem decrypting the page. If this error persists, please contact support.")
			UserSession.Save(r, w)
			err = nil
		}
	}

	data := map[string]interface{}{
		"document": d,
		"body":     body,
		"user":     user,
	}

	RenderTemplate(w, r, "view", data)
}

// EditHandler handles the document edit page
func EditHandler(w http.ResponseWriter, r *http.Request) {
	_, id := path.Split(r.URL.Path)
	d := &Document{}
	var err error
	var body template.HTML
	user := getUserFromSession()

	if id != "" {
		d, err = loadPage(id)
		if err != nil {
			ErrorLogger.Print("Page not found. id: "+id, err)
			UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			UserSession.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if !levelCheck(w, r, d) {
			http.Redirect(w, r, "/", http.StatusFound)
		}

		body, err = d.decrypt()
		if err != nil {
			ErrorLogger.Print("Could not decrypt page id: "+id+" \nDisplaying blank body\n ", err)
			UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			UserSession.Save(r, w)
			err = nil
		}
	}

	users, err := findAllUsers()
	if err != nil {
		ErrorLogger.Print("Could not find all users. Document {id: "+id+"} ", err)
		err = nil
	}

	folders, err := findAllFolders()
	if err != nil {
		ErrorLogger.Print("Could not find all folders. Document {id: "+id+"} ", err)
		err = nil
	}

	query := r.URL.Query()
	if len(query["folder-id"]) > 0 && query["folder-id"][0] != "" {
		d.FolderID = bson.ObjectIdHex(query["folder-id"][0])
	}

	data := map[string]interface{}{
		"document": d,
		"body":     body,
		"users":    users,
		"user":     user,
		"folders":  folders,
	}

	RenderTemplate(w, r, "edit", data)
}

// SaveHandler handles the document save page
func SaveHandler(w http.ResponseWriter, r *http.Request) {
	_, idHex := path.Split(r.URL.Path)
	var d *Document

	if r.Method == "GET" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	} else if r.Method == "POST" {
		var userIDs []bson.ObjectId
		r.ParseForm()
		title := r.Form["title"][0]
		body := template.HTML(r.Form["body"][0])
		strUserIDs := r.Form["users"]
		strFolderID := r.Form["folder"][0]

		d = &Document{
			Title:  title,
			Edited: time.Now(),
		}

		level, err := strconv.Atoi(r.Form["level"][0])

		if err != nil {
			ErrorLogger.Print("Error parsing document level POST. {id: "+idHex+"} ", err.Error())
			err = nil
		} else {
			d.Level = level
		}

		for _, uID := range strUserIDs {
			userIDs = append(userIDs, bson.ObjectIdHex(uID))
		}

		if len(userIDs) > 0 {
			d.UserIDs = userIDs
		}

		// if document is in a folder, write it to the folder
		if strFolderID != "" {
			d.FolderID = bson.ObjectIdHex(strFolderID)
		}

		err = d.encrypt(body)
		if err != nil {
			ErrorLogger.Print("Could not encrypt body of document id: "+idHex+" \n ", err)
			err = nil
		}

		if idHex != "" {
			d.ID = bson.ObjectIdHex(idHex)
		} else {
			d.ID = bson.NewObjectId()
			d.Created = time.Now()
		}

		err = d.save()

		if err != nil {
			ErrorLogger.Print("Could not save page id: "+d.ID.Hex()+" \n ", err)
			UserSession.AddFlash("Error! Could not save page. If this error persists please contact support", "error")
			UserSession.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
			err = nil
			return
		}

		InfoLogger.Print("Document saved {id: " + d.ID.Hex() + "}")
	}

	redir := "/view/" + d.ID.Hex()
	if d.FolderID.Hex() != "" {
		redir = "/folder/view/" + d.FolderID.Hex()
	}

	http.Redirect(w, r, redir, http.StatusFound)
}

func (d *Document) encrypt(body template.HTML) (err error) {
	var block cipher.Block
	key := []byte(keyHash)
	plaintext := []byte(body)
	block, err = aes.NewCipher(key)
	if err != nil {
		return err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	d.Body = ciphertext

	return err
}

func (d *Document) decrypt() (body template.HTML, err error) {
	var block cipher.Block
	key := []byte(keyHash)
	ciphertext := d.Body
	block, err = aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("Can't decrypt document, ciphertext too short.")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	body = template.HTML(ciphertext)
	return body, err
}
