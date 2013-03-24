package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
)

const (
	DB_ADDR                = "127.0.0.1"
	DB_NAME                = "GuildWars2"
	COLLECTION_SALVAGE     = "salvage"
	COLLECTION_SALVAGEMATS = "salvageMaterials"
	GW2SPIDY_URL           = "http://www.gw2spidy.com/api/v0.9/json/"
)

var templates = template.Must(template.ParseFiles("main.html", "addSalvage.html", "viewSalvage.html"))

///////////////////////////////////////////////////////////////////////////////
//
// Structures
//
///////////////////////////////////////////////////////////////////////////////

// Used to retrieve data from the salvage database
type Salvage struct {
	ID           string     "ID"
	SalvageCount int        "SalvageCount"
	Materials    []Material "Materials"
}

type Material struct {
	ID    string "ID"
	Count int    "Count"
}

// Used to retrieve lists of items from GW2Spidy
type GW2SpidyItemList struct {
	Count int                `json:"count"`
	Items []GW2SpidyItemData `json:"results"`
}

// Used to retrieve a single item from GW2Spidy
type GW2SpidyItemResult struct {
	Result GW2SpidyItemData `json:"result"`
}

// Main struct containing necessary data for items
type GW2SpidyItemData struct {
	DataID                   int    `bson:"data_id" json:"data_id"`
	Name                     string `bson:"name" json:"name"`
	Img                      string `bson:"img,omitempty" json:"img"`
	Rarity                   int    `bson:"rarity,omitempty" json:"rarity"`
	RestrictionLevel         int    `bson:"restriction_level,omitempty" json:"restriction_level"`
	TypeID                   int    `bson:"type_id,omitempty" json:"type_id"`
	SubTypeID                int    `bson:"sub_type_id,omitempty" json:"sub_type_id"`
	PriceLastChanged         string `bson:"price_last_changed,omitempty" json:"price_last_changed"`
	MaxOfferUnitPrice        int    `bson:"max_offer_unit_price,omitempty" json:"max_offer_unit_price"`
	MinSaleUnitPrice         int    `bson:"min_sale_unit_price,omitempty" json:"min_sale_unit_price"`
	OfferAvailability        int    `bson:"offer_availability,omitempty" json:"offer_availability"`
	SaleAvailability         int    `bson:"sale_availability,omitempty" json:"sale_availability"`
	GW2DBExternalID          int    `bson:"gw2db_external_id,omitempty" json:"gw2db_external_id"`
	SalePriceChangeLastHour  int    `bson:"sale_price_change_last_hour,omitempty" json:"sale_price_change_last_hour"`
	OfferPriceChangeLastHour int    `bson:"offer_price_change_last_hour,omitempty" json:"offer_price_change_last_hour"`
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
	collection := session.DB(DB_NAME).C(COLLECTION_SALVAGEMATS)

	// Pull out all the data in the database
	var result []GW2SpidyItemData
	err = collection.Find(nil).All(&result)
	handleError(err, response, "Unable to retrieve data from collection")

	// Execute the template with the data so we can show all the material items available
	err = templates.ExecuteTemplate(response, "addSalvage.html", map[string]interface{}{"Materials": result})
	handleError(err, response, "Unable to execute template")
}

// View all the item data
func viewSalvageDataHandler(response http.ResponseWriter, request *http.Request) {
	// Connect to the database
	session, err := mgo.Dial(DB_ADDR)
	handleError(err, response, "Unable to connect to database")
	defer session.Close()

	// Open up our collection
	collection := session.DB(DB_NAME).C(COLLECTION_SALVAGE)

	// Pull out all the data in the database
	var salvages []Salvage
	err = collection.Find(nil).All(&salvages)
	handleError(err, response, "Unable to retrieve salvage data from collection")

	salvageItems := make([]GW2SpidyItemData, 0, 100)
	materialItems := make([]GW2SpidyItemData, 0, 100)
	requestURL := GW2SPIDY_URL + "item/"

	for _, salvageItem := range salvages {
		resp, err := http.Get(requestURL + salvageItem.ID)

		if err != nil {
			fmt.Println("Get request failed with id", salvageItem.ID)
			continue
		}

		// Read contents of request
		contents, err := ioutil.ReadAll(resp.Body)
		handleError(err, response, "Unable to read body of response")

		// Unmarshal JSON into item data
		item := GW2SpidyItemResult{}
		err = json.Unmarshal(contents, &item)
		handleError(err, response, "Unable to Unmarshal contents of response")

		salvageItems = append(salvageItems, item.Result)

		// Retrieve all material data as needed
		for _, materialItem := range salvageItem.Materials {
			if isMaterialInSlice(materialItems, materialItem.ID) == false {
				resp, err := http.Get(requestURL + materialItem.ID)

				if err != nil {
					fmt.Println("Get request failed with id", materialItem.ID)
					continue
				}

				// Read contents of request
				contents, err := ioutil.ReadAll(resp.Body)
				handleError(err, response, "Unable to read body of response")

				// Unmarshal JSON into item data
				item := GW2SpidyItemResult{}
				err = json.Unmarshal(contents, &item)
				handleError(err, response, "Unable to Unmarshal contents of response")

				materialItems = append(materialItems, item.Result)
			}
		}
	}

	// Execute the template with the data so we can show all the data available
	err = templates.ExecuteTemplate(response, "viewSalvage.html", map[string]interface{}{"Materials": materialItems, "Items": salvageItems})
	handleError(err, response, "Unable to execute template")
}

func isMaterialInSlice(materials []GW2SpidyItemData, id string) bool {
	data_id, err := strconv.Atoi(id)

	if err != nil {
		fmt.Println("Unable to conver id", id, "to an int.")
		return true
	}

	for _, mat := range materials {
		if mat.DataID == data_id {
			return true
		}
	}

	return false
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
	salvageCount, err := strconv.Atoi(request.Form.Get("SalvageCount"))
	handleError(err, response, "Atoi SalvageCount failure")

	mat1 := request.Form.Get("material1")
	mat1Count, _ := strconv.Atoi(request.Form.Get("material1Count"))

	mat2 := request.Form.Get("material2")
	mat2Count, _ := strconv.Atoi(request.Form.Get("material2Count"))

	// Grab our collection
	c := session.DB(DB_NAME).C(COLLECTION_SALVAGE)
	query := c.Find(bson.M{"ID": itemID})
	count, err := query.Count()
	handleError(err, response, "Unable to get count of documents found")

	result := Salvage{}

	if count == 0 {
		// Add the new entry to the database
		result.ID = itemID
		result.SalvageCount = salvageCount

		if mat1 != "" {
			result.Materials = append(result.Materials, Material{ID: mat1, Count: mat1Count})
		}

		if mat2 != "" {
			result.Materials = append(result.Materials, Material{ID: mat2, Count: mat2Count})
		}

		err = c.Insert(result)
		handleError(err, response, "Unable to insert new Salvage")
	} else {
		// Increment the current entry in the database
		err = query.One(&result)
		handleError(err, response, "Unable to parse Salvage from result")

		result.SalvageCount += salvageCount
		newMatStats := []Material{}

		if mat1 != "" {
			newMatStats = append(result.Materials, Material{ID: mat1, Count: mat1Count})
		}

		if mat2 != "" {
			newMatStats = append(result.Materials, Material{ID: mat2, Count: mat2Count})
		}

		for m := range newMatStats {
			found := false

			for i := range result.Materials {
				if newMatStats[m].ID == result.Materials[i].ID {
					found = true
					result.Materials[i].Count += newMatStats[m].Count
				}
			}

			if found == false {
				result.Materials = append(result.Materials, newMatStats[m])
			}
		}

		err = c.Update(bson.M{"ID": itemID}, result)
		handleError(err, response, "Update failed")
	}

	// Return to main page
	// TODO: Return to a better page to show results/success
	// TODO: Proper redirection
	http.Redirect(response, request, "../addSalvage", http.StatusFound)
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
	http.HandleFunc("/viewSalvage", viewSalvageDataHandler)
	http.ListenAndServe(":8080", nil)
}
