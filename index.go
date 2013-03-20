package main

import (
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

var templates = template.Must(template.ParseFiles("main.html", "addSalvage.html", "addSalvageTypes.html"))

///////////////////////////////////////////////////////////////////////////////
//
// Structures
//
///////////////////////////////////////////////////////////////////////////////

// Used to retrieve data from the salvage database
type Salvage struct {
	ID           string     "ID"
	SalvageCount string     "SalvageCount"
	Materials    []Material "Materials"
}

type Material struct {
	ID    string "ID"
	Count string "Count"
}

// Used to retrieve lists of items from GW2Spidy
type GW2SpidyItemList struct {
	Count string             `json:"count"`
	Items []GW2SpidyItemData `json:"results"`
}

// Main struct containing necessary data for items
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

// Handler to show the main page
// TODO: Find better way to execute/send html file in response
func mainHandler(response http.ResponseWriter, requeest *http.Request) {
	err := templates.ExecuteTemplate(response, "main.html", nil)
	handleError(err, response, "Unable to execute template")
}

// Handler to show page that allows the addition of new salvage data
func addSalvageHandler(response http.ResponseWriter, request *http.Request) {
	// Connect to the database
	session, err := mgo.Dial(DB_ADDR)
	handleError(err, response, "Unable to connect to database")
	defer session.Close()

	// Open up our collection
	collection := session.DB("GuildWars2").C("salvageMaterials")

	// Pull out all the data in the database
	var result []GW2SpidyItemData
	err = collection.Find(nil).All(&result)
	handleError(err, response, "Unable to execute template")

	// Execute the template with the data so we can show all the material items available
	err = templates.ExecuteTemplate(response, "addSalvage.html", map[string]interface{}{"Materials": result})
	handleError(err, response, "Unable to execute template")
}

// Handler to show page that allows the addition of new material types
func addSalvateTypeHandler(response http.ResponseWriter, request *http.Request) {
	// Retrieve items under the 'Crafting Material' category
	requestURL := GW2SPIDY_URL + "all-items/5"
	resp, err := http.Get(requestURL)
	handleError(err, response, "Get request failed")
	defer resp.Body.Close()

	// Read contents of request
	contents, err := ioutil.ReadAll(resp.Body)
	handleError(err, response, "Unable to read body of response")

	// Umarshal JSON into item data objects
	itemList := GW2SpidyItemList{}
	err = json.Unmarshal(contents, &itemList)
	handleError(err, response, "Unable to Unmarshal contents of response")

	// TODO: Remove items already existing in salvageMaterials to avoid duplication
	// Execute template to show checklist to add to database
	err = templates.ExecuteTemplate(response, "addSalvageTypes.html", itemList.Items)
	handleError(err, response, "Unable to execute template")
}

///////////////////////////////////////////////////////////////////////////////
//
// Lib Handler Functions
//
///////////////////////////////////////////////////////////////////////////////

// Handles the addition of new salvage data
func libAddSalvageHandler(response http.ResponseWriter, request *http.Request) {
	// Connect to the database
	session, err := mgo.Dial(DB_ADDR)
	handleError(err, response, "Unable to connect to database")
	defer session.Close()

	// Parse out the query parameters to make them available in the Form
	request.ParseForm()

	itemID := request.Form.Get("ID")
	salvageCount := request.Form.Get("SalvageCount")
	mat1 := request.Form.Get("material1")
	mat1Count := request.Form.Get("material1Count")
	mat2 := request.Form.Get("material2")
	mat2Count := request.Form.Get("material2Count")

	// Grab our collection
	c := session.DB("GuildWars2").C("salvage")
	query := c.Find(bson.M{"ID": itemID})
	count, err := query.Count()
	result := Salvage{}

	fmt.Println(count)

	if count == 0 {
		// Add the new entry to the database
		result.ID = itemID
		result.SalvageCount = salvageCount

		if mat2 != "" {
			result.Materials = make([]Material, 2)
			fmt.Println("Adding mat2!", mat2, mat2Count)
			result.Materials[1] = Material{mat2, mat2Count}
		} else {
			result.Materials = make([]Material, 1)
		}

		result.Materials[0] = Material{mat1, mat1Count}

		err = c.Insert(result)
		handleError(err, response, "Unable to insert new Salvage")
	} else {
		// Increment the current entry in the database
		err = query.One(&result)
		handleError(err, response, "Unable to parse Salvage from result")
	}
}

// Handles the addition of new salvage material data
func libAddSalvageTypeHandler(response http.ResponseWriter, request *http.Request) {
	// Connect to the database
	session, err := mgo.Dial(DB_ADDR)
	handleError(err, response, "Unable to connect to database")
	defer session.Close()

	// Parse out the query parameters to make them available in the Form
	request.ParseForm()

	// Grab our collection
	c := session.DB("GuildWars2").C("salvageMaterials")
	var itemData GW2SpidyItemData

	// Loop through all query paramters and add them to the database
	for id := range request.Form {
		itemData.DataID = id
		itemData.Name = request.Form.Get(id)
		// TODO: Check for duplicates before inserting
		err = c.Insert(itemData)

		if err != nil {
			fmt.Println("Unable to add id ", id, "to salvageMaterials. ", err.Error())
		}
	}

	// Return to main page
	// TODO: Return to a better page to show results/success
	// TODO: Proper redirection
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
	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/main", mainHandler)
	http.HandleFunc("/addSalvage", addSalvageHandler)
	http.HandleFunc("/lib/addSalvage", libAddSalvageHandler)

	//http.HandleFunc("/addSalvageType", addSalvateTypeHandler)
	//http.HandleFunc("/lib/addSalvageType", libAddSalvageTypeHandler)
	http.ListenAndServe(":8080", nil)
}
