package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
)

const (
	DB_ADDR      = "127.0.0.1"
	GW2SPIDY_URL = "http://www.gw2spidy.com/api/v0.9/json/"
)

var templates = template.Must(template.ParseFiles("types.html", "main.html", "addSalvage.html", "addSalvageTypes.html"))

///////////////////////////////////////////////////////////////////////////////
//
// Structures
//
///////////////////////////////////////////////////////////////////////////////

type Salvage struct {
	ID           string "ID"
	SalvageCount string "SalvageCount"
}

type ItemType struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Subtypes []ItemType `json:"subtypes"`
}

type ItemTypeResponse struct {
	Results []ItemType `json:"results"`
}

type GW2SpidyItemList struct {
	Count int                `json:"count"`
	Items []GW2SpidyItemData `json:"results"`
}

type GW2SpidyItemData struct {
	DataID string `bson:"data_id" json:"data_id"`
	Name   string `bson:"name" json:"name"`
	Img    string `bson:"img,omitempty" json:"img"`
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
	requestURL := GW2SPIDY_URL + "types"
	resp, err := http.Get(requestURL)
	handleError(err, response, "Get request failed")
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	handleError(err, response, "Unable to read body of response")

	responseItemType := ItemTypeResponse{}
	err = json.Unmarshal(contents, &responseItemType)
	handleError(err, response, "Unable to Unmarshal contents of response")

	err = templates.ExecuteTemplate(response, "types.html", responseItemType.Results)
	handleError(err, response, "Unable to execute template")
}

func addSalvageHandler(response http.ResponseWriter, request *http.Request) {
	session, err := mgo.Dial(DB_ADDR)

	if err != nil {
		panic(err)
	}
	defer session.Close()

	c := session.DB("GuildWars2").C("salvageMaterials")
	var result []GW2SpidyItemData
	err = c.Find(nil).All(&result)

	if err != nil {
		panic(err)
	}

	err = templates.ExecuteTemplate(response, "addSalvage.html", map[string]interface{}{"Materials": result})
	handleError(err, response, "Unable to execute template")
}

func addSalvateTypeHandler(response http.ResponseWriter, request *http.Request) {
	requestURL := GW2SPIDY_URL + "all-items/5"
	resp, err := http.Get(requestURL)
	handleError(err, response, "Get request failed")
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	handleError(err, response, "Unable to read body of response")

	itemList := GW2SpidyItemList{}
	err = json.Unmarshal(contents, &itemList)
	handleError(err, response, "Unable to Unmarshal contents of response")

	err = templates.ExecuteTemplate(response, "addSalvageTypes.html", itemList.Items)
	handleError(err, response, "Unable to execute template")
}

///////////////////////////////////////////////////////////////////////////////
//
// Lib Handler Functions
//
///////////////////////////////////////////////////////////////////////////////

func libAddSalvageHandler(response http.ResponseWriter, request *http.Request) {

}

func libAddSalvageTypeHandler(response http.ResponseWriter, request *http.Request) {
	session, err := mgo.Dial(DB_ADDR)

	if err != nil {
		panic(err)
	}
	defer session.Close()

	request.ParseForm()

	c := session.DB("GuildWars2").C("salvageMaterials")
	var itemData GW2SpidyItemData

	for id := range request.Form {
		itemData.DataID = id
		itemData.Name = request.Form.Get(id)

		fmt.Println("id: ", id, "value: ", request.Form.Get(id))
		err = c.Insert(itemData)

		if err != nil {
			panic(err)
		}
	}

	http.Redirect(response, request, "../main", http.StatusOK)
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
	http.HandleFunc("/addSalvage", addSalvageHandler)
	http.HandleFunc("/lib/addSalvage", libAddSalvageHandler)

	//http.HandleFunc("/addSalvageType", addSalvateTypeHandler)
	//http.HandleFunc("/lib/addSalvageType", libAddSalvageTypeHandler)
	http.ListenAndServe(":8080", nil)
}
