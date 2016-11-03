package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"gopkg.in/mgo.v2/bson"
)

// Page represents any webpage on the site
type Page struct {
	ID    bson.ObjectId `json:"id" bson:"_id"`
	Title string        `json:"title" bson:"title"`
	Body  []byte        `json:"body" bson:"body"`
	URL   string        `json:"url" bson:"url"`
	Level int           `json:"level" bson:"level"`
}

const pageCol = "pages"
const cryptoKey = "Take these documents, both the sealed and unsealed copies of the deed of purchase, and put them in a clay jar so they will last a long time."

// Save saves the page to the database
func (p *Page) Save() error {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	_, err := collection.UpsertId(p.ID, p)
	return err
}

// LoadPage loads retrieves the page data from the database
func LoadPage(idHex string) (*Page, error) {
	id := bson.ObjectIdHex(idHex)
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	page := Page{}
	err := collection.FindId(id).One(&page)

	if err != nil {
		return nil, err
	}
	return &page, nil
}

func findAllDocs() (*[]Page, error) {
	session := dbConnect()
	defer session.Close()

	collection := session.DB(db).C(pageCol)
	var pages []Page
	err := collection.Find(nil).All(&pages)
	if err != nil {
		return nil, err
	}

	return &pages, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, id string) {
	var body template.HTML
	p, err := LoadPage(id)
	if err != nil {
		errorLogger.Print("Page not found. id: "+id, err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	user := getUserFromSession()

	if !levelCheck(w, r, p) {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	body, err = p.decrypt()
	if err != nil {
		errorLogger.Print("Could not decrypt page id: "+id, err)
		UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"page": p,
		"body": body,
		"user": user,
	}

	renderTemplate(w, r, "view", data)
}

func editHandler(w http.ResponseWriter, r *http.Request, id string) {
	p := &Page{}
	var err error
	var body template.HTML
	user := getUserFromSession()

	if id != "" {
		p, err = LoadPage(id)
		if err != nil {
			errorLogger.Print("Page not found. id: "+id, err)
			UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if !levelCheck(w, r, p) {
			http.Redirect(w, r, "/", http.StatusFound)
		}

		body, err = p.decrypt()
		if err != nil {
			errorLogger.Print("Could not decrypt page id: "+id, err)
			UserSession.AddFlash("Looks like something went wrong. If this error persists, please contact support", "error")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	users, err := findAllUsers()
	if err != nil {
		errorLogger.Print("Could not find all users. page id: "+id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	data := map[string]interface{}{
		"page":  p,
		"body":  body,
		"users": users,
		"user":  user,
	}

	renderTemplate(w, r, "edit", data)
}

func saveHandler(w http.ResponseWriter, r *http.Request, idHex string) {
	title := r.FormValue("title")
	body := template.HTML(r.FormValue("body"))
	// userIds := strings.Split(r.FormValue("permissions"), ",")
	level, err := strconv.Atoi(r.FormValue("level"))

	p := &Page{
		Title: title,
		Level: level,
	}

	err = p.encrypt(body)

	if idHex != "" {
		p.ID = bson.ObjectIdHex(idHex)
	} else {
		p.ID = bson.NewObjectId()
	}

	err = p.Save()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// err = permissionsSave(userIds, id)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }

	http.Redirect(w, r, "/view/"+p.ID.Hex(), http.StatusFound)
}

func (p *Page) encrypt(body template.HTML) (err error) {
	var block cipher.Block
	key := []byte(cryptoKey)
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

	p.Body = ciphertext

	return err
}

func (p *Page) decrypt() (body template.HTML, err error) {
	var block cipher.Block
	key := []byte(cryptoKey)
	ciphertext := p.Body
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
