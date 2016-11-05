package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/crypto/scrypt"

	"gopkg.in/mgo.v2/bson"
)

// Document represents a document on the site
type Document struct {
	ID      bson.ObjectId `json:"id" bson:"_id"`
	Title   string        `json:"title" bson:"title"`
	Body    []byte        `json:"body" bson:"body"`
	URL     string        `json:"url" bson:"url"`
	Level   int           `json:"level" bson:"level"`
	Folders []Folder
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
	err := collection.Find(nil).All(&documents)
	if err != nil {
		return nil, err
	}

	return &documents, nil
}

// IndexHandler handles the index page request
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession()
	pages, err := findAllDocs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	data := map[string]interface{}{
		"pages": pages,
		"user":  user,
	}

	RenderTemplate(w, r, "index", data)
}

// ViewHandler handles the document view page
func ViewHandler(w http.ResponseWriter, r *http.Request, id string) {
	var body template.HTML
	d, err := loadPage(id)
	if err != nil {
		ErrorLogger.Print("Document not found. id: "+id, err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	user := getUserFromSession()

	if !levelCheck(w, r, d) {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	body, err = d.decrypt()
	if err != nil {
		InfoLogger.Print("Could not decrypt page id: "+id+"\nDisplaying blank body", err)
	}

	data := map[string]interface{}{
		"document": d,
		"body":     body,
		"user":     user,
	}

	RenderTemplate(w, r, "view", data)
}

// EditHandler handles the document edit page
func EditHandler(w http.ResponseWriter, r *http.Request, id string) {
	d := &Document{}
	var err error
	var body template.HTML
	user := getUserFromSession()

	if id != "" {
		d, err = loadPage(id)
		if err != nil {
			ErrorLogger.Print("Page not found. id: "+id, err)
			UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
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
		}
	}

	users, err := findAllUsers()
	if err != nil {
		ErrorLogger.Print("Could not find all users. page id: "+id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	data := map[string]interface{}{
		"document": d,
		"body":     body,
		"users":    users,
		"user":     user,
	}

	RenderTemplate(w, r, "edit", data)
}

// SaveHandler handles the document save page
func SaveHandler(w http.ResponseWriter, r *http.Request, idHex string) {
	title := r.FormValue("title")
	body := template.HTML(r.FormValue("body"))
	// userIds := strings.Split(r.FormValue("permissions"), ",")
	level, err := strconv.Atoi(r.FormValue("level"))

	d := &Document{
		Title: title,
		Level: level,
	}

	err = d.encrypt(body)
	if err != nil {
		ErrorLogger.Print("Could not encrypt body of page id: "+idHex+" \n ", err)
	}

	if idHex != "" {
		d.ID = bson.ObjectIdHex(idHex)
	} else {
		d.ID = bson.NewObjectId()
	}

	err = d.save()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// err = permissionsSave(userIds, id)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }

	http.Redirect(w, r, "/view/"+d.ID.Hex(), http.StatusFound)
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
