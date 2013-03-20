package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
)

const (
	DB_ADDR = "127.0.0.1"
)

var templates = template.Must(template.ParseFiles("types.html", "main.html"))

///////////////////////////////////////////////////////////////////////////////
//
// Structures
//
///////////////////////////////////////////////////////////////////////////////

type Salvage struct {
	ID           int "ID"
	SalvageCount int "SalvageCount"
}

type ItemType struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	Subtypes []ItemType `json:"subtypes"`
}

type ItemTypeResponse struct {
	Results []ItemType `json:"results"`
}

///////////////////////////////////////////////////////////////////////////////
//
// Handler Functions
//
///////////////////////////////////////////////////////////////////////////////

func defaultHandler(response http.ResponseWriter, request *http.Request) {
	session, err := mgo.Dial(DB_ADDR)

	if err != nil {
		panic(err)
	}
	defer session.Close()

	c := session.DB("GuildWars2").C("salvage")
	result := Salvage{}
	err = c.Find(bson.M{"ID": 187}).One(&result)

	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer

	tmpl, err := template.New("test").Parse("<html>ID: {{.ID}} <br />Salvage Count: {{.SalvageCount}}</html>")

	if err != nil {
		panic(err)
	}

	tmpl.Execute(&buffer, result)
	response.Write(buffer.Bytes())
}

func mainHandler(response http.ResponseWriter, requeest *http.Request) {
	err := templates.ExecuteTemplate(response, "main.html", nil)
	handleError(err, response, "Unable to execute template")
}

func typeHandler(response http.ResponseWriter, request *http.Request) {
	requestURL := "http://www.gw2spidy.com/api/v0.9/json/types"
	resp, err := http.Get(requestURL)
	handleError(err, response, "Get request failed")
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	handleError(err, response, "Unable to read body of response")

	var responseItemType ItemTypeResponse
	err = json.Unmarshal(contents, &responseItemType)
	handleError(err, response, "Unable to Unmarshal contents of response")

	err = templates.ExecuteTemplate(response, "types.html", responseItemType.Results)
	handleError(err, response, "Unable to execute template")
}

///////////////////////////////////////////////////////////////////////////////
//
// Error Functions
//
///////////////////////////////////////////////////////////////////////////////

func handleError(err error, response http.ResponseWriter, message string) {
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		panic(message)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// Initialization Functions
//
///////////////////////////////////////////////////////////////////////////////

func main() {
	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/main", mainHandler)
	http.HandleFunc("/types", typeHandler)
	http.ListenAndServe(":8080", nil)
}
